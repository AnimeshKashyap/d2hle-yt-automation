package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/d2hle/yt-automation/internal/api"
	"github.com/d2hle/yt-automation/internal/bootstrap"
	"github.com/d2hle/yt-automation/internal/config"
	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/d2hle/yt-automation/internal/job"
	"github.com/d2hle/yt-automation/internal/paths"
	"github.com/d2hle/yt-automation/internal/profile"
	"github.com/d2hle/yt-automation/internal/runtime"
	"github.com/d2hle/yt-automation/internal/studio"
	"github.com/d2hle/yt-automation/internal/validate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "serve":
		runServe(os.Args[2:])
	case "profile":
		runProfile(os.Args[2:])
	case "job":
		runJob(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n  d2hle-yt serve\n  d2hle-yt profile create --id <profile_id> [--name <display>]\n  d2hle-yt profile login --id <profile_id>\n  d2hle-yt profile check --id <profile_id>\n  d2hle-yt profile refresh --id <profile_id>\n  d2hle-yt profile list\n  d2hle-yt job run --id <job_id>\n")
}

func loadCommon() (config.Config, paths.Layout, *profile.Store, *job.Store, *runtime.ProfileLocks, *slog.Logger, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, paths.Layout{}, nil, nil, nil, nil, err
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}))
	layout := paths.NewLayout(cfg)
	profiles := profile.NewStore(layout)
	jobs := job.NewStore(layout)
	locks := runtime.NewProfileLocks()
	if err := bootstrap.EnsureDataDirs(layout, profiles, jobs); err != nil {
		return config.Config{}, paths.Layout{}, nil, nil, nil, nil, err
	}
	return cfg, layout, profiles, jobs, locks, log, nil
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	_ = fs.Parse(args)

	cfg, layout, profiles, jobs, locks, log, err := loadCommon()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	studioSvc := studio.NewService(cfg, profiles, locks, log)
	slots := runtime.NewSlotPool(cfg.MaxConcurrentJobs)
	executor := job.NewExecutor(cfg, jobs, profiles, studioSvc, slots, log)
	srv := api.NewServer(cfg, layout, profiles, jobs, slots, studioSvc, executor, log)
	httpSrv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: srv.Handler(),
	}

	docsURL := config.DocsBaseURL(cfg.HTTPHost, cfg.HTTPPort) + "/v1/docs"
	log.Info("starting http server", "addr", cfg.HTTPAddr, "data_root", cfg.DataRoot, "docs", docsURL)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	_ = httpSrv.Shutdown(context.Background())
}

func runProfile(args []string) {
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	switch args[0] {
	case "create":
		runProfileCreate(args[1:])
	case "list":
		runProfileList()
	case "login":
		runProfileLogin(args[1:])
	case "check":
		runProfileCheck(args[1:])
	case "refresh":
		runProfileRefresh(args[1:])
	default:
		usage()
		os.Exit(1)
	}
}

func runProfileCreate(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	id := fs.String("id", "", "profile id")
	name := fs.String("name", "", "display name")
	_ = fs.Parse(args)
	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		os.Exit(1)
	}
	if err := validate.ProfileID(*id); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _, profiles, _, _, log, err := loadCommon()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	display := *name
	if display == "" {
		display = *id
	}
	p, err := profiles.Create(domain.CreateProfileRequest{ProfileID: *id, DisplayName: display})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log.Info("profile created", "profile_id", p.ID, "user_data_dir", p.UserDataDir)
}

func runProfileList() {
	_, _, profiles, _, _, _, err := loadCommon()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	list, err := profiles.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, p := range list {
		fmt.Printf("%s\t%s\t%s\n", p.ID, p.SessionStatus, p.UserDataDir)
	}
}

func runProfileLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	id := fs.String("id", "", "profile id")
	_ = fs.Parse(args)
	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		os.Exit(1)
	}
	cfg, _, profiles, _, locks, log, err := loadCommon()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := profiles.Get(*id); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	studioSvc := studio.NewService(cfg, profiles, locks, log)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.LoginWaitTimeout)
	defer cancel()

	fmt.Println("A Chrome/Chromium window will open using your automation profile (not your daily browser profile).")
	fmt.Println("Google blocks many automated browsers; we use installed Chrome + stealth flags. See BROWSER_CHANNEL in .env.example.")
	fmt.Println("Sign in to Google/YouTube; MFA via phone/email is expected.")
	p, err := studioSvc.InteractiveLogin(ctx, *id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "login failed:", err)
		if p != nil {
			fmt.Fprintf(os.Stderr, "profile status: %s\n", p.SessionStatus)
		}
		os.Exit(1)
	}
	log.Info("profile ready", "profile_id", p.ID, "session_status", p.SessionStatus, "account_hint", p.AccountHint)
}

func runProfileCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	id := fs.String("id", "", "profile id")
	_ = fs.Parse(args)
	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		os.Exit(1)
	}
	cfg, _, profiles, _, locks, log, err := loadCommon()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	studioSvc := studio.NewService(cfg, profiles, locks, log)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.BrowserNavTimeout+2*cfg.BrowserLaunchTimeout)
	defer cancel()

	p, err := studioSvc.CheckSession(ctx, *id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s\t%s\t%s\n", p.ID, p.SessionStatus, p.AccountHint)
}

func runProfileRefresh(args []string) {
	fs := flag.NewFlagSet("refresh", flag.ExitOnError)
	id := fs.String("id", "", "profile id")
	_ = fs.Parse(args)
	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		os.Exit(1)
	}
	cfg, _, profiles, _, locks, log, err := loadCommon()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	studioSvc := studio.NewService(cfg, profiles, locks, log)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.SessionRefreshTimeout+2*cfg.BrowserLaunchTimeout)
	defer cancel()

	fmt.Println("Opening browser to refresh YouTube session (cookie-based profile, not OAuth).")
	fmt.Println("Complete any Google challenge in the window if prompted.")
	p, err := studioSvc.RefreshSession(ctx, *id)
	if err != nil {
		if p != nil {
			fmt.Fprintf(os.Stderr, "%s\t%s\t%s\n", p.ID, p.SessionStatus, p.AccountHint)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log.Info("session refreshed", "profile_id", p.ID, "session_status", p.SessionStatus, "last_refreshed_at", p.LastRefreshedAt)
	fmt.Printf("%s\t%s\t%s\n", p.ID, p.SessionStatus, p.AccountHint)
}

func runJob(args []string) {
	if len(args) == 0 || args[0] != "run" {
		usage()
		os.Exit(1)
	}
	fs := flag.NewFlagSet("job run", flag.ExitOnError)
	id := fs.String("id", "", "job id")
	_ = fs.Parse(args[1:])
	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		os.Exit(1)
	}
	cfg, _, profiles, jobs, locks, log, err := loadCommon()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	studioSvc := studio.NewService(cfg, profiles, locks, log)
	slots := runtime.NewSlotPool(cfg.MaxConcurrentJobs)
	executor := job.NewExecutor(cfg, jobs, profiles, studioSvc, slots, log)
	if !slots.TryAcquire() {
		fmt.Fprintln(os.Stderr, "executor busy")
		os.Exit(1)
	}
	defer slots.Release()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.UploadJobTimeout)
	defer cancel()
	if err := executor.Run(ctx, *id); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	j, _ := jobs.Get(*id)
	if j != nil {
		fmt.Printf("%s\t%s\t%s\n", j.ID, j.Status, j.CurrentStep)
	}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
