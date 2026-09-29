package studio

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/d2hle/yt-automation/internal/browser"
	"github.com/d2hle/yt-automation/internal/config"
	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/d2hle/yt-automation/internal/profile"
	"github.com/d2hle/yt-automation/internal/runtime"
	"github.com/mxschmitt/playwright-go"
)

type Service struct {
	cfg      config.Config
	profiles *profile.Store
	locks    *runtime.ProfileLocks
	log      *slog.Logger
}

func NewService(cfg config.Config, profiles *profile.Store, locks *runtime.ProfileLocks, log *slog.Logger) *Service {
	return &Service{cfg: cfg, profiles: profiles, locks: locks, log: log}
}

func (s *Service) CheckSession(ctx context.Context, profileID string) (*domain.Profile, error) {
	p, err := s.profiles.Get(profileID)
	if err != nil {
		return nil, err
	}
	unlock := s.locks.Lock(profileID)
	defer unlock()

	var result DetectResult
	err = browser.WithPersistentContext(ctx, browser.SessionCheckPersistent(s.cfg, p.UserDataDir), func(bctx playwright.BrowserContext) error {
		page, err := browser.FirstPage(bctx)
		if err != nil {
			return err
		}
		result, err = DetectOnPage(page, s.cfg.StudioURL, s.cfg.BrowserNavTimeout)
		return err
	})
	if err != nil {
		s.log.Warn("session check failed", "profile_id", profileID, "err", err)
		return s.profiles.UpdateStatus(profileID, domain.SessionCheckFailed, "")
	}
	s.log.Info("session check result",
		"profile_id", profileID,
		"status", result.Status,
		"headless", s.cfg.SessionCheckHeadless,
	)
	return s.profiles.UpdateStatus(profileID, result.Status, result.AccountHint)
}

func (s *Service) InteractiveLogin(ctx context.Context, profileID string) (*domain.Profile, error) {
	p, err := s.profiles.Get(profileID)
	if err != nil {
		return nil, err
	}

	if _, err := s.profiles.UpdateStatus(profileID, domain.SessionLoginInProgress, ""); err != nil {
		return nil, err
	}

	unlock := s.locks.Lock(profileID)
	defer unlock()

	deadline := time.Now().Add(s.cfg.LoginWaitTimeout)
	poll := time.NewTicker(3 * time.Second)
	defer poll.Stop()

	s.log.Info("opening headed browser for login",
		"profile_id", profileID,
		"user_data_dir", p.UserDataDir,
		"browser_channel", s.cfg.BrowserChannel,
		"browser_stealth", s.cfg.BrowserStealth,
	)

	launchErr := browser.WithPersistentContext(ctx, browser.LoginPersistent(s.cfg, p.UserDataDir), func(bc playwright.BrowserContext) error {
		page, err := browser.FirstPage(bc)
		if err != nil {
			return err
		}
		loginURL := "https://accounts.google.com/AccountChooser?continue=" + urlEncode(s.cfg.StudioURL)
		if _, err := DetectOnPage(page, loginURL, s.cfg.BrowserNavTimeout); err != nil {
			return err
		}

		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("login timed out after %s", s.cfg.LoginWaitTimeout)
			}

			res := DetectCurrent(page)
			switch res.Status {
			case domain.SessionActive:
				_, _ = s.profiles.UpdateStatus(profileID, domain.SessionActive, res.AccountHint)
				s.log.Info("login complete", "profile_id", profileID, "account_hint", res.AccountHint)
				return nil
			case domain.SessionNeedsUserAction:
				s.log.Info("waiting for user action (MFA/CAPTCHA)", "profile_id", profileID)
			case domain.SessionNeedsReauth:
				s.log.Debug("waiting for sign-in", "profile_id", profileID)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-poll.C:
			}
		}
	})

	if launchErr != nil {
		updated, _ := s.profiles.UpdateStatus(profileID, domain.SessionNeedsReauth, "")
		if updated != nil {
			return updated, launchErr
		}
		return nil, launchErr
	}

	return s.profiles.Get(profileID)
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}
