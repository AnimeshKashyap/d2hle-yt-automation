package studio

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/d2hle/yt-automation/internal/browser"
	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/mxschmitt/playwright-go"
)

// ErrSessionRefreshIncomplete means the browser session could not be confirmed active after refresh.
var ErrSessionRefreshIncomplete = errors.New("session refresh did not reach active state")

// RefreshSession opens Studio in a real browser profile (headed by default), touches Google/YouTube
// pages to keep cookie-based auth warm, and updates profile session metadata.
// This is not an OAuth refresh token; YouTube auth lives in the profile userDataDir.
func (s *Service) RefreshSession(ctx context.Context, profileID string) (*domain.Profile, error) {
	p, err := s.profiles.Get(profileID)
	if err != nil {
		return nil, err
	}
	if p.SessionStatus == domain.SessionLoginInProgress {
		return p, fmt.Errorf("login already in progress for profile %q", profileID)
	}

	unlock := s.locks.Lock(profileID)
	defer unlock()

	headless := s.cfg.SessionRefreshHeadless
	var final DetectResult

	runErr := browser.WithPersistentContext(ctx, browser.SessionRefreshPersistent(s.cfg, p.UserDataDir), func(bctx playwright.BrowserContext) error {
		page, err := browser.FirstPage(bctx)
		if err != nil {
			return err
		}

		result, err := DetectOnPage(page, s.cfg.StudioURL, s.cfg.BrowserNavTimeout)
		if err != nil {
			return err
		}
		final = result

		if result.Status == domain.SessionActive {
			if err := warmSession(page, s.cfg.BrowserNavTimeout, s.cfg.SessionRefreshDwell); err != nil {
				s.log.Warn("session warm-up had errors", "profile_id", profileID, "err", err)
			}
			final = DetectCurrent(page)
			return nil
		}

		if !headless && (result.Status == domain.SessionNeedsUserAction || result.Status == domain.SessionNeedsReauth) {
			s.log.Info("session refresh waiting for user", "profile_id", profileID, "status", result.Status)
			final = waitForRefreshResolution(ctx, page, s.cfg.SessionRefreshTimeout)
			if final.Status == domain.SessionActive {
				if err := warmSession(page, s.cfg.BrowserNavTimeout, s.cfg.SessionRefreshDwell); err != nil {
					s.log.Warn("session warm-up had errors", "profile_id", profileID, "err", err)
				}
				final = DetectCurrent(page)
			}
			return nil
		}

		return nil
	})

	if runErr != nil {
		s.log.Warn("session refresh failed", "profile_id", profileID, "err", runErr)
		updated, _ := s.profiles.UpdateStatus(profileID, domain.SessionCheckFailed, "")
		if updated != nil {
			return updated, runErr
		}
		return nil, runErr
	}

	s.log.Info("session refresh result",
		"profile_id", profileID,
		"status", final.Status,
		"headless", headless,
	)

	var updated *domain.Profile
	if final.Status == domain.SessionActive {
		updated, err = s.profiles.UpdateAfterRefresh(profileID, final.Status, final.AccountHint)
	} else {
		updated, err = s.profiles.UpdateStatus(profileID, final.Status, final.AccountHint)
	}
	if err != nil {
		return nil, err
	}
	if final.Status != domain.SessionActive {
		return updated, ErrSessionRefreshIncomplete
	}
	return updated, nil
}

func warmSession(page playwright.Page, navTimeout, dwell time.Duration) error {
	ms := float64(navTimeout.Milliseconds())
	targets := []string{
		"https://studio.youtube.com/",
		"https://www.youtube.com/",
	}
	for _, url := range targets {
		if _, err := page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   &ms,
		}); err != nil {
			return err
		}
		if dwell > 0 {
			page.WaitForTimeout(float64(dwell.Milliseconds()))
		}
	}
	if _, err := page.Goto("https://studio.youtube.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   &ms,
	}); err != nil {
		return err
	}
	return nil
}

func waitForRefreshResolution(ctx context.Context, page playwright.Page, budget time.Duration) DetectResult {
	deadline := time.Now().Add(budget)
	poll := time.NewTicker(2 * time.Second)
	defer poll.Stop()

	var last DetectResult
	for {
		last = DetectCurrent(page)
		if last.Status == domain.SessionActive {
			return last
		}
		if last.Status == domain.SessionNeedsReauth && definiteSignOut(page) {
			return last
		}
		if time.Now().After(deadline) {
			return last
		}
		select {
		case <-ctx.Done():
			return last
		case <-poll.C:
		}
	}
}
