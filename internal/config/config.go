package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DataRoot              string
	HTTPAddr              string
	MaxConcurrentJobs     int
	Headless              bool
	AllowedMediaRoots     []string
	APIToken              string
	LogLevel              string
	StudioURL             string
	BrowserLaunchTimeout  time.Duration
	BrowserNavTimeout     time.Duration
	LoginWaitTimeout      time.Duration
	InteractiveLogin      bool
	BrowserChannel        string
	BrowserExecutablePath string
	BrowserStealth        bool
	SessionCheckHeadless  bool
	SessionRefreshHeadless bool
	SessionRefreshTimeout  time.Duration
	SessionRefreshDwell    time.Duration
	UploadHeadless        bool
	UploadJobTimeout      time.Duration
	StudioActionTimeout   time.Duration
	UploadUIWaitTimeout   time.Duration
	StepOpenStudio        time.Duration
	StepUploadDialog      time.Duration
	StepMetadata          time.Duration
	StepWizard            time.Duration
	StepPublish           time.Duration
	StepVerify            time.Duration
}

func Load() (Config, error) {
	dataRoot := envOr("DATA_ROOT", "./data")
	mediaRoots := envOr("ALLOWED_MEDIA_ROOTS", joinPath(dataRoot, "media"))
	roots := splitRoots(mediaRoots)
	if len(roots) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_MEDIA_ROOTS must not be empty")
	}

	maxJobs, err := envInt("MAX_CONCURRENT_JOBS", 1)
	if err != nil {
		return Config{}, err
	}
	if maxJobs < 1 {
		return Config{}, fmt.Errorf("MAX_CONCURRENT_JOBS must be >= 1")
	}

	headless := true
	if v := os.Getenv("HEADLESS"); v != "" {
		headless, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("HEADLESS: %w", err)
		}
	}

	launchTimeout, err := envDuration("BROWSER_LAUNCH_TIMEOUT", 3*time.Minute)
	if err != nil {
		return Config{}, err
	}
	navTimeout, err := envDuration("BROWSER_NAV_TIMEOUT", 90*time.Second)
	if err != nil {
		return Config{}, err
	}
	loginWait, err := envDuration("LOGIN_WAIT_TIMEOUT", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	interactiveLogin := true
	if v := os.Getenv("INTERACTIVE_LOGIN"); v != "" {
		interactiveLogin, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("INTERACTIVE_LOGIN: %w", err)
		}
	}

	browserStealth := true
	if v := os.Getenv("BROWSER_STEALTH"); v != "" {
		browserStealth, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("BROWSER_STEALTH: %w", err)
		}
	}

	// Headless session checks often look logged-out to Google even with valid cookies.
	// Default to headed checks on the host; set SESSION_CHECK_HEADLESS=true in Docker.
	sessionCheckHeadless := false
	if v := os.Getenv("SESSION_CHECK_HEADLESS"); v != "" {
		sessionCheckHeadless, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("SESSION_CHECK_HEADLESS: %w", err)
		}
	}

	sessionRefreshHeadless := false
	if v := os.Getenv("SESSION_REFRESH_HEADLESS"); v != "" {
		sessionRefreshHeadless, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("SESSION_REFRESH_HEADLESS: %w", err)
		}
	}

	sessionRefreshTimeout, err := envDuration("SESSION_REFRESH_TIMEOUT", 10*time.Minute)
	if err != nil {
		return Config{}, err
	}
	sessionRefreshDwell, err := envDuration("SESSION_REFRESH_DWELL", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	uploadHeadless := false
	if v := os.Getenv("UPLOAD_HEADLESS"); v != "" {
		uploadHeadless, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("UPLOAD_HEADLESS: %w", err)
		}
	}

	uploadJobTimeout, err := envDuration("UPLOAD_JOB_TIMEOUT", 120*time.Minute)
	if err != nil {
		return Config{}, err
	}
	studioActionTimeout, err := envDuration("STUDIO_ACTION_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	uploadUIWait, err := envDuration("UPLOAD_UI_WAIT_TIMEOUT", 120*time.Minute)
	if err != nil {
		return Config{}, err
	}

	stepOpenStudio, err := envDuration("STUDIO_STEP_OPEN_STUDIO_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepUploadDialog, err := envDuration("STUDIO_STEP_UPLOAD_DIALOG_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepMetadata, err := envDuration("STUDIO_STEP_METADATA_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepWizard, err := envDuration("STUDIO_STEP_WIZARD_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepPublish, err := envDuration("STUDIO_STEP_PUBLISH_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepVerify, err := envDuration("STUDIO_STEP_VERIFY_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DataRoot:             dataRoot,
		HTTPAddr:             envOr("HTTP_ADDR", "127.0.0.1:8081"),
		MaxConcurrentJobs:    maxJobs,
		Headless:             headless,
		AllowedMediaRoots:    roots,
		APIToken:             os.Getenv("API_TOKEN"),
		LogLevel:             envOr("LOG_LEVEL", "info"),
		StudioURL:            envOr("STUDIO_URL", "https://studio.youtube.com"),
		BrowserLaunchTimeout: launchTimeout,
		BrowserNavTimeout:    navTimeout,
		LoginWaitTimeout:      loginWait,
		InteractiveLogin:      interactiveLogin,
		BrowserChannel:        envOr("BROWSER_CHANNEL", "chrome"),
		BrowserExecutablePath: os.Getenv("BROWSER_EXECUTABLE_PATH"),
		BrowserStealth:       browserStealth,
		SessionCheckHeadless:   sessionCheckHeadless,
		SessionRefreshHeadless: sessionRefreshHeadless,
		SessionRefreshTimeout:  sessionRefreshTimeout,
		SessionRefreshDwell:    sessionRefreshDwell,
		UploadHeadless:         uploadHeadless,
		UploadJobTimeout:    uploadJobTimeout,
		StudioActionTimeout: studioActionTimeout,
		UploadUIWaitTimeout: uploadUIWait,
		StepOpenStudio:      stepOpenStudio,
		StepUploadDialog:    stepUploadDialog,
		StepMetadata:        stepMetadata,
		StepWizard:          stepWizard,
		StepPublish:         stepPublish,
		StepVerify:          stepVerify,
	}, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	if strings.HasSuffix(v, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(v, "m"))
		if err != nil {
			return 0, fmt.Errorf("%s: %w", key, err)
		}
		return time.Duration(n) * time.Minute, nil
	}
	if strings.HasSuffix(v, "s") {
		n, err := strconv.Atoi(strings.TrimSuffix(v, "s"))
		if err != nil {
			return 0, fmt.Errorf("%s: %w", key, err)
		}
		return time.Duration(n) * time.Second, nil
	}
	sec, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return time.Duration(sec) * time.Second, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}

func splitRoots(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func joinPath(a, b string) string {
	if strings.HasSuffix(a, "/") || strings.HasSuffix(a, "\\") {
		return a + b
	}
	sep := "/"
	if strings.Contains(a, "\\") {
		sep = "\\"
	}
	return a + sep + b
}
