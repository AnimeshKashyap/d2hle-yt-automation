package bootstrap

import (
	"os"

	"github.com/d2hle/yt-automation/internal/job"
	"github.com/d2hle/yt-automation/internal/paths"
	"github.com/d2hle/yt-automation/internal/profile"
)

func EnsureDataDirs(layout paths.Layout, profiles *profile.Store, jobs *job.Store) error {
	dirs := []string{
		layout.Root,
		layout.Profiles,
		layout.Jobs,
		layout.Staging,
		layout.Artifacts,
		layout.Media,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	if err := profiles.EnsureDirs(); err != nil {
		return err
	}
	return jobs.EnsureDirs()
}
