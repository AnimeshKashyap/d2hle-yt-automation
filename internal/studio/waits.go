package studio

import (
	"fmt"
	"time"

	"github.com/mxschmitt/playwright-go"
)

func waitVisible(page playwright.Page, selector string, timeout time.Duration) error {
	ms := float64(timeout.Milliseconds())
	return page.Locator(selector).First().WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: &ms,
	})
}

func clickFirst(page playwright.Page, selectors []string, timeout time.Duration) error {
	perTry := timeout / time.Duration(len(selectors))
	if perTry < 2*time.Second {
		perTry = 2 * time.Second
	}
	if perTry > timeout {
		perTry = timeout
	}
	tryMs := float64(perTry.Milliseconds())
	for _, sel := range selectors {
		loc := page.Locator(sel).First()
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		if err := loc.Click(playwright.LocatorClickOptions{Timeout: &tryMs}); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clickable element for selectors %v", selectors)
}

func fillTextbox(page playwright.Page, selectors []string, value string, timeout time.Duration) error {
	ms := float64(timeout.Milliseconds())
	for _, sel := range selectors {
		loc := page.Locator(sel).First()
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		if err := loc.Fill(value, playwright.LocatorFillOptions{Timeout: &ms}); err == nil {
			return nil
		}
		if err := loc.Click(playwright.LocatorClickOptions{Timeout: &ms}); err != nil {
			continue
		}
		_ = loc.Fill(value)
		return nil
	}
	return fmt.Errorf("could not fill textbox")
}

func clickFirstIn(root playwright.Locator, selectors []string, timeout time.Duration) error {
	perTry := timeout / time.Duration(len(selectors))
	if perTry < 2*time.Second {
		perTry = 2 * time.Second
	}
	if perTry > timeout {
		perTry = timeout
	}
	tryMs := float64(perTry.Milliseconds())
	for _, sel := range selectors {
		loc := root.Locator(sel).First()
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		if err := loc.Click(playwright.LocatorClickOptions{Timeout: &tryMs}); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clickable element for selectors %v", selectors)
}

func fillTextboxIn(root playwright.Locator, selectors []string, value string, timeout time.Duration) error {
	ms := float64(timeout.Milliseconds())
	for _, sel := range selectors {
		loc := root.Locator(sel).First()
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		if err := loc.Fill(value, playwright.LocatorFillOptions{Timeout: &ms}); err == nil {
			return nil
		}
		if err := loc.Click(playwright.LocatorClickOptions{Timeout: &ms}); err != nil {
			continue
		}
		_ = loc.Fill(value)
		return nil
	}
	return fmt.Errorf("could not fill textbox")
}
