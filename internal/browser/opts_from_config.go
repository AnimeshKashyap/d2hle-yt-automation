package browser

import "github.com/d2hle/yt-automation/internal/config"

func PersistentFromConfig(cfg config.Config, userDataDir string, headless bool) PersistentOptions {
	return PersistentOptions{
		UserDataDir:    userDataDir,
		Headless:       headless,
		Timeout:        cfg.BrowserLaunchTimeout,
		Channel:        cfg.BrowserChannel,
		ExecutablePath: cfg.BrowserExecutablePath,
		Stealth:        cfg.BrowserStealth,
	}
}

// LoginPersistent uses headed Chrome-friendly settings for Google sign-in.
func LoginPersistent(cfg config.Config, userDataDir string) PersistentOptions {
	o := PersistentFromConfig(cfg, userDataDir, false)
	o.Stealth = true
	if o.Channel == "" {
		o.Channel = "chrome"
	}
	return o
}

// SessionCheckPersistent probes Studio (headed by default; see SESSION_CHECK_HEADLESS).
func SessionCheckPersistent(cfg config.Config, userDataDir string) PersistentOptions {
	o := PersistentFromConfig(cfg, userDataDir, cfg.SessionCheckHeadless)
	o.Stealth = true
	if o.Channel == "" {
		o.Channel = "chrome"
	}
	return o
}

// SessionRefreshPersistent keeps cookie auth warm (headed by default; see SESSION_REFRESH_HEADLESS).
func SessionRefreshPersistent(cfg config.Config, userDataDir string) PersistentOptions {
	o := PersistentFromConfig(cfg, userDataDir, cfg.SessionRefreshHeadless)
	o.Stealth = true
	if o.Channel == "" {
		o.Channel = "chrome"
	}
	return o
}

// UploadPersistent runs Studio uploads (headed by default; see UPLOAD_HEADLESS).
func UploadPersistent(cfg config.Config, userDataDir string) PersistentOptions {
	o := PersistentFromConfig(cfg, userDataDir, cfg.UploadHeadless)
	o.Stealth = true
	if o.Channel == "" {
		o.Channel = "chrome"
	}
	return o
}
