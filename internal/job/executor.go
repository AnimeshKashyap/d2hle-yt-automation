package job

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/d2hle/yt-automation/internal/config"
	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/d2hle/yt-automation/internal/profile"
	"github.com/d2hle/yt-automation/internal/runtime"
	"github.com/d2hle/yt-automation/internal/studio"
)

type Executor struct {
	cfg      config.Config
	jobs     *Store
	profiles *profile.Store
	studio   *studio.Service
	slots    *runtime.SlotPool
	log      *slog.Logger
}

func NewExecutor(cfg config.Config, jobs *Store, profiles *profile.Store, studioSvc *studio.Service, slots *runtime.SlotPool, log *slog.Logger) *Executor {
	return &Executor{
		cfg:      cfg,
		jobs:     jobs,
		profiles: profiles,
		studio:   studioSvc,
		slots:    slots,
		log:      log,
	}
}

func (e *Executor) StartAsync(jobID string) bool {
	if !e.slots.TryAcquire() {
		return false
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), e.cfg.UploadJobTimeout)
		defer cancel()
		defer e.slots.Release()
		if err := e.Run(ctx, jobID); err != nil {
			e.log.Warn("job run finished with error", "job_id", jobID, "err", err)
		}
	}()
	return true
}

func (e *Executor) Run(ctx context.Context, jobID string) error {
	j, err := e.jobs.Get(jobID)
	if err != nil {
		return err
	}
	p, err := e.profiles.Get(j.ProfileID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	j.Status = domain.JobRunning
	j.CurrentStep = "starting"
	j.UpdatedAt = now
	if err := e.jobs.Save(j); err != nil {
		return err
	}

	onStep := func(step string) {
		latest, err := e.jobs.Get(jobID)
		if err != nil {
			return
		}
		latest.CurrentStep = step
		if step == "uploading_file" || step == "wizard" || step == "waiting_upload" {
			latest.Status = domain.JobProcessing
		}
		latest.UpdatedAt = time.Now().UTC()
		_ = e.jobs.Save(latest)
	}

	artifacts, err := e.studio.RunUploadJob(ctx, p, j, onStep)

	latest, loadErr := e.jobs.Get(jobID)
	if loadErr != nil {
		return loadErr
	}
	finished := time.Now().UTC()
	latest.UpdatedAt = finished
	latest.FinishedAt = &finished
	latest.ScreenshotPath = artifacts.ScreenshotPath
	latest.TracePath = artifacts.TracePath
	latest.YouTubeVideoURL = artifacts.VideoURL

	code, awaitingAuth := studio.ClassifyUploadError(err)
	if err != nil {
		studio.LogUploadError(e.log, jobID, err)
		if awaitingAuth {
			latest.Status = domain.JobAwaitingAuth
			latest.ErrorCode = code
			latest.ErrorMessage = err.Error()
			_, _ = e.profiles.UpdateStatus(j.ProfileID, domain.SessionNeedsReauth, "")
		} else {
			latest.Status = domain.JobFailed
			latest.ErrorCode = code
			latest.ErrorMessage = err.Error()
		}
		return e.jobs.Save(latest)
	}

	latest.Status = domain.JobPublished
	latest.CurrentStep = "published"
	latest.ErrorCode = ""
	latest.ErrorMessage = ""
	return e.jobs.Save(latest)
}

func (e *Executor) CanAccept() bool {
	if !e.slots.Available() {
		return false
	}
	_, err := e.jobs.FindRunning()
	return errors.Is(err, ErrNotFound)
}
