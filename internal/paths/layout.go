package paths

import (
	"path/filepath"

	"github.com/d2hle/yt-automation/internal/config"
)

type Layout struct {
	Root      string
	Profiles  string
	Jobs      string
	Staging   string
	Artifacts string
	Media     string
}

func NewLayout(cfg config.Config) Layout {
	root := cfg.DataRoot
	return Layout{
		Root:      root,
		Profiles:  filepath.Join(root, "state", "profiles"),
		Jobs:      filepath.Join(root, "state", "jobs"),
		Staging:   filepath.Join(root, "staging"),
		Artifacts: filepath.Join(root, "state", "artifacts"),
		Media:     filepath.Join(root, "media"),
	}
}

func ProfileMetaPath(layout Layout, profileID string) string {
	return filepath.Join(layout.Profiles, profileID, "profile.json")
}

func ProfileUserDataDir(layout Layout, profileID string) string {
	return filepath.Join(layout.Profiles, profileID, "browser")
}

func JobPath(layout Layout, jobID string) string {
	return filepath.Join(layout.Jobs, jobID+".json")
}

func JobArtifactsDir(layout Layout, jobID string) string {
	return filepath.Join(layout.Artifacts, jobID)
}
