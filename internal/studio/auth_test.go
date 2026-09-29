package studio

import (
	"testing"

	"github.com/d2hle/yt-automation/internal/domain"
)

func TestClassifyURL(t *testing.T) {
	cases := []struct {
		url string
		want domain.SessionStatus
	}{
		{"https://accounts.google.com/signin/v2/challenge/pwd", domain.SessionNeedsUserAction},
		{"https://accounts.google.com/signin/v2/challenge/totp", domain.SessionNeedsUserAction},
		{"https://accounts.google.com/ServiceLogin", domain.SessionNeedsReauth},
		{"https://studio.youtube.com/channel/UCabc", ""},
	}
	for _, tc := range cases {
		got := classifyURL(tc.url)
		if got != tc.want {
			t.Fatalf("classifyURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}
