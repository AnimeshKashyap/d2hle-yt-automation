package studio

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/d2hle/yt-automation/internal/browser"
	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/mxschmitt/playwright-go"
)

type UploadArtifacts struct {
	ScreenshotPath string
	TracePath      string
	VideoURL       string
}

func (s *Service) RunUploadJob(ctx context.Context, profile *domain.Profile, job *domain.Job, onStep func(string)) (UploadArtifacts, error) {
	artifacts := UploadArtifacts{}
	unlock := s.locks.Lock(profile.ID)
	defer unlock()

	uploader := &Uploader{
		StudioURL:        s.cfg.StudioURL,
		NavTimeout:       s.cfg.BrowserNavTimeout,
		ActionTimeout:    s.cfg.StudioActionTimeout,
		UploadUIWait:     s.cfg.UploadUIWaitTimeout,
		StepOpenStudio:   s.cfg.StepOpenStudio,
		StepUploadDialog: s.cfg.StepUploadDialog,
		StepMetadata:     s.cfg.StepMetadata,
		StepWizard:       s.cfg.StepWizard,
		StepPublish:      s.cfg.StepPublish,
		StepVerify:       s.cfg.StepVerify,
	}

	err := browser.WithPersistentContext(ctx, browser.UploadPersistent(s.cfg, profile.UserDataDir), func(bctx playwright.BrowserContext) error {
		if err := os.MkdirAll(job.ArtifactDir, 0o700); err != nil {
			return err
		}
		if err := bctx.Tracing().Start(playwright.TracingStartOptions{
			Screenshots: playwright.Bool(true),
			Snapshots:   playwright.Bool(true),
			Sources:     playwright.Bool(true),
		}); err != nil {
			s.log.Warn("tracing start failed", "err", err)
		}

		page, err := browser.FirstPage(bctx)
		if err != nil {
			return err
		}

		result, runErr := uploader.Run(ctx, page, job.Request, UploadCallbacks{OnStep: onStep})
		if result.VideoURL != "" {
			artifacts.VideoURL = result.VideoURL
		}
		tracePath := filepath.Join(job.ArtifactDir, "trace.zip")
		if err := bctx.Tracing().Stop(tracePath); err == nil {
			artifacts.TracePath = tracePath
		}
		if runErr != nil {
			shot := filepath.Join(job.ArtifactDir, "failure.png")
			if _, err := page.Screenshot(playwright.PageScreenshotOptions{
				Path:     playwright.String(shot),
				FullPage: playwright.Bool(true),
			}); err == nil {
				artifacts.ScreenshotPath = shot
			}
		}
		return runErr
	})
	return artifacts, err
}

func ClassifyUploadError(err error) (code string, awaitingAuth bool) {
	if err == nil {
		return "", false
	}
	if errors.Is(err, ErrNeedsAuth) {
		return "needs_auth", true
	}
	var step *StepError
	if errors.As(err, &step) {
		if errors.Is(step.Err, ErrNeedsAuth) {
			return "needs_auth", true
		}
		if step.Step == "open_studio" && step.Err != nil && containsTimeout(step.Err.Error()) {
			return "needs_auth", true
		}
		return "upload_" + step.Step, false
	}
	return "upload_failed", false
}

func LogUploadError(log *slog.Logger, jobID string, err error) {
	log.Error("upload job failed", "job_id", jobID, "err", err)
}
