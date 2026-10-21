package config

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	DataRoot              string
	HTTPHost              string
	HTTPPort              int
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
	src := loadEnvSource()

	dataRoot := src.envOr("DATA_ROOT", "./data")
	mediaRoots := src.envOr("ALLOWED_MEDIA_ROOTS", joinPath(dataRoot, "media"))
	roots := splitRoots(mediaRoots)
	if len(roots) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_MEDIA_ROOTS must not be empty")
	}

	maxJobs, err := src.envInt("MAX_CONCURRENT_JOBS", 1)
	if err != nil {
		return Config{}, err
	}
	if maxJobs < 1 {
		return Config{}, fmt.Errorf("MAX_CONCURRENT_JOBS must be >= 1")
	}

	headless, err := src.envBool("HEADLESS", true)
	if err != nil {
		return Config{}, err
	}

	launchTimeout, err := src.envDuration("BROWSER_LAUNCH_TIMEOUT", 3*time.Minute)
	if err != nil {
		return Config{}, err
	}
	navTimeout, err := src.envDuration("BROWSER_NAV_TIMEOUT", 90*time.Second)
	if err != nil {
		return Config{}, err
	}
	loginWait, err := src.envDuration("LOGIN_WAIT_TIMEOUT", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	interactiveLogin, err := src.envBool("INTERACTIVE_LOGIN", true)
	if err != nil {
		return Config{}, err
	}

	browserStealth, err := src.envBool("BROWSER_STEALTH", true)
	if err != nil {
		return Config{}, err
	}

	// Headless session checks often look logged-out to Google even with valid cookies.
	// Default to headed checks on the host; set SESSION_CHECK_HEADLESS=true in Docker.
	sessionCheckHeadless, err := src.envBool("SESSION_CHECK_HEADLESS", false)
	if err != nil {
		return Config{}, err
	}

	sessionRefreshHeadless, err := src.envBool("SESSION_REFRESH_HEADLESS", false)
	if err != nil {
		return Config{}, err
	}

	sessionRefreshTimeout, err := src.envDuration("SESSION_REFRESH_TIMEOUT", 10*time.Minute)
	if err != nil {
		return Config{}, err
	}
	sessionRefreshDwell, err := src.envDuration("SESSION_REFRESH_DWELL", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	uploadHeadless, err := src.envBool("UPLOAD_HEADLESS", false)
	if err != nil {
		return Config{}, err
	}

	uploadJobTimeout, err := src.envDuration("UPLOAD_JOB_TIMEOUT", 120*time.Minute)
	if err != nil {
		return Config{}, err
	}
	studioActionTimeout, err := src.envDuration("STUDIO_ACTION_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	uploadUIWait, err := src.envDuration("UPLOAD_UI_WAIT_TIMEOUT", 120*time.Minute)
	if err != nil {
		return Config{}, err
	}

	stepOpenStudio, err := src.envDuration("STUDIO_STEP_OPEN_STUDIO_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepUploadDialog, err := src.envDuration("STUDIO_STEP_UPLOAD_DIALOG_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepMetadata, err := src.envDuration("STUDIO_STEP_METADATA_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepWizard, err := src.envDuration("STUDIO_STEP_WIZARD_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepPublish, err := src.envDuration("STUDIO_STEP_PUBLISH_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	stepVerify, err := src.envDuration("STUDIO_STEP_VERIFY_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	httpHost, httpPort, httpAddr, err := resolveHTTPListenAddr(src)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DataRoot:               dataRoot,
		HTTPHost:               httpHost,
		HTTPPort:               httpPort,
		HTTPAddr:               httpAddr,
		MaxConcurrentJobs:      maxJobs,
		Headless:               headless,
		AllowedMediaRoots:      roots,
		APIToken:               src.envString("API_TOKEN"),
		LogLevel:               src.envOr("LOG_LEVEL", "info"),
		StudioURL:              src.envOr("STUDIO_URL", "https://studio.youtube.com"),
		BrowserLaunchTimeout:   launchTimeout,
		BrowserNavTimeout:      navTimeout,
		LoginWaitTimeout:       loginWait,
		InteractiveLogin:       interactiveLogin,
		BrowserChannel:         src.envOr("BROWSER_CHANNEL", "chrome"),
		BrowserExecutablePath:  src.envString("BROWSER_EXECUTABLE_PATH"),
		BrowserStealth:         browserStealth,
		SessionCheckHeadless:   sessionCheckHeadless,
		SessionRefreshHeadless: sessionRefreshHeadless,
		SessionRefreshTimeout:  sessionRefreshTimeout,
		SessionRefreshDwell:    sessionRefreshDwell,
		UploadHeadless:         uploadHeadless,
		UploadJobTimeout:       uploadJobTimeout,
		StudioActionTimeout:    studioActionTimeout,
		UploadUIWaitTimeout:    uploadUIWait,
		StepOpenStudio:         stepOpenStudio,
		StepUploadDialog:       stepUploadDialog,
		StepMetadata:           stepMetadata,
		StepWizard:             stepWizard,
		StepPublish:            stepPublish,
		StepVerify:             stepVerify,
	}, nil
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
