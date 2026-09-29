package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/d2hle/yt-automation/internal/config"
	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/d2hle/yt-automation/internal/job"
	"github.com/d2hle/yt-automation/internal/media"
	"github.com/d2hle/yt-automation/internal/paths"
	"github.com/d2hle/yt-automation/internal/profile"
	"github.com/d2hle/yt-automation/internal/runtime"
	"github.com/d2hle/yt-automation/internal/studio"
	"github.com/d2hle/yt-automation/internal/validate"
)

type Server struct {
	cfg      config.Config
	layout   paths.Layout
	profiles *profile.Store
	jobs     *job.Store
	slots    *runtime.SlotPool
	studio   *studio.Service
	executor *job.Executor
	log      *slog.Logger
}

func NewServer(cfg config.Config, layout paths.Layout, profiles *profile.Store, jobs *job.Store, slots *runtime.SlotPool, studioSvc *studio.Service, executor *job.Executor, log *slog.Logger) *Server {
	return &Server{
		cfg:      cfg,
		layout:   layout,
		profiles: profiles,
		jobs:     jobs,
		slots:    slots,
		studio:   studioSvc,
		executor: executor,
		log:      log,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/docs", s.handleDocs)
	mux.HandleFunc("GET /v1/health", s.handleHealth)
	mux.HandleFunc("GET /v1/ready", s.handleReady)
	mux.HandleFunc("POST /v1/profiles", s.auth(s.handleCreateProfile))
	mux.HandleFunc("GET /v1/profiles", s.handleListProfiles)
	mux.HandleFunc("GET /v1/profiles/{profile_id}", s.handleGetProfile)
	mux.HandleFunc("GET /v1/profiles/{profile_id}/status", s.handleGetProfile)
	mux.HandleFunc("POST /v1/profiles/{profile_id}/tokens", s.auth(s.handleIssueToken))
	mux.HandleFunc("DELETE /v1/profiles/{profile_id}/tokens/{token_id}", s.auth(s.handleRevokeToken))
	mux.HandleFunc("POST /v1/profiles/{profile_id}/session/check", s.auth(s.handleSessionCheck))
	mux.HandleFunc("POST /v1/profiles/{profile_id}/session/refresh", s.auth(s.handleSessionRefresh))
	mux.HandleFunc("POST /v1/profiles/{profile_id}/login", s.auth(s.handleLogin))
	mux.HandleFunc("POST /v1/jobs", s.auth(s.handleCreateJob))
	mux.HandleFunc("GET /v1/jobs/current", s.handleCurrentJob)
	mux.HandleFunc("GET /v1/jobs/{job_id}", s.handleGetJob)
	return mux
}

func (s *Server) handleDocs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "d2hle-yt-automation",
		"version": "v1",
		"routes":  routeCatalog(),
	})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.APIToken == "" {
			next(w, r)
			return
		}
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		plain := strings.TrimPrefix(h, "Bearer ")
		if !s.tokenAllowed(plain) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}
		next(w, r)
	}
}

func (s *Server) tokenAllowed(plain string) bool {
	if s.cfg.APIToken != "" && plain == s.cfg.APIToken {
		return true
	}
	list, err := s.profiles.List()
	if err != nil {
		return false
	}
	for _, p := range list {
		for _, t := range p.Tokens {
			if profile.TokenMatches(plain, t.Hash) {
				return true
			}
		}
	}
	return false
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	if s.slots.Available() {
		writeJSON(w, http.StatusOK, map[string]any{"ready": true, "active_jobs": s.slots.ActiveCount()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ready": false, "active_jobs": s.slots.ActiveCount()})
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validate.ProfileID(req.ProfileID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", err.Error())
		return
	}
	if req.DisplayName == "" {
		req.DisplayName = req.ProfileID
	}
	p, err := s.profiles.Create(req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, "profile_exists", err.Error())
			return
		}
		s.log.Error("create profile", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not create profile")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListProfiles(w http.ResponseWriter, _ *http.Request) {
	list, err := s.profiles.List()
	if err != nil {
		s.log.Error("list profiles", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not list profiles")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": list})
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	p, err := s.profiles.Get(id)
	if err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not load profile")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleIssueToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	if _, err := s.profiles.Get(id); err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not load profile")
		return
	}
	var req domain.IssueTokenRequest
	if err := decodeJSON(r, &req); err != nil && r.ContentLength != 0 {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	tok, plain, err := profile.NewAPIToken(req.Label)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not issue token")
		return
	}
	if _, err := s.profiles.AddToken(id, tok); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not save token")
		return
	}
	writeJSON(w, http.StatusCreated, domain.IssueTokenResponse{TokenID: tok.ID, Token: plain})
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	tokenID := r.PathValue("token_id")
	if _, err := s.profiles.RemoveToken(id, tokenID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not revoke token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSessionCheck(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.BrowserNavTimeout+2*s.cfg.BrowserLaunchTimeout)
	defer cancel()

	p, err := s.studio.CheckSession(ctx, id)
	if err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}
		s.log.Error("session check", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "session check failed")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleSessionRefresh(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.SessionRefreshTimeout+2*s.cfg.BrowserLaunchTimeout)
	defer cancel()

	p, err := s.studio.RefreshSession(ctx, id)
	if err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}
		if p != nil && errors.Is(err, studio.ErrSessionRefreshIncomplete) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":   "session_not_refreshed",
				"message": "session is not active after refresh; run profile login if needed",
				"profile": p,
			})
			return
		}
		if p != nil && strings.Contains(err.Error(), "login already in progress") {
			writeError(w, http.StatusConflict, "login_in_progress", err.Error())
			return
		}
		s.log.Error("session refresh", "profile_id", id, "err", err)
		if p != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"error":   "refresh_failed",
				"message": err.Error(),
				"profile": p,
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "session refresh failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profile": p,
		"message": "session refreshed",
	})
}

type loginRequest struct {
	Headed bool `json:"headed"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	p, err := s.profiles.Get(id)
	if err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not load profile")
		return
	}
	if p.SessionStatus == domain.SessionLoginInProgress {
		writeError(w, http.StatusConflict, "login_in_progress", "login already in progress for this profile")
		return
	}

	var body loginRequest
	_ = decodeJSON(r, &body)
	headed := body.Headed || strings.EqualFold(r.URL.Query().Get("headed"), "true")

	if headed && s.cfg.InteractiveLogin {
		p, err = s.profiles.UpdateStatus(id, domain.SessionLoginInProgress, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not update profile")
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.LoginWaitTimeout)
			defer cancel()
			if _, err := s.studio.InteractiveLogin(ctx, id); err != nil {
				s.log.Warn("interactive login ended", "profile_id", id, "err", err)
			}
		}()
		writeJSON(w, http.StatusAccepted, map[string]any{
			"profile": p,
			"message": "headed browser login started; complete sign-in in the opened window",
		})
		return
	}

	p, err = s.profiles.UpdateStatus(id, domain.SessionLoginInProgress, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not update profile")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"profile": p,
		"message": "Run headed login on the host (recommended for Docker): d2hle-yt profile login --id=" + id,
		"cli":     "d2hle-yt profile login --id=" + id,
	})
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	if !s.executor.CanAccept() {
		running, _ := s.jobs.FindRunning()
		activeID := ""
		if running != nil {
			activeID = running.ID
		}
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":         "busy",
			"message":       "an upload job is already active",
			"active_job_id": activeID,
		})
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	if idem != "" {
		existing, err := s.jobs.GetByIdempotencyKey(idem)
		if err == nil {
			writeJSON(w, http.StatusOK, existing)
			return
		}
	}

	var req domain.CreateJobRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validate.ProfileID(req.ProfileID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", err.Error())
		return
	}
	p, err := s.profiles.Get(req.ProfileID)
	if err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "unknown_profile", "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not load profile")
		return
	}
	if p.SessionStatus != domain.SessionActive {
		writeError(w, http.StatusConflict, "profile_not_ready", "profile session is not active; status="+string(p.SessionStatus))
		return
	}
	if busy, err := s.jobs.FindRunningForProfile(req.ProfileID); err == nil && busy != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":         "profile_busy",
			"message":       "this profile already has a job in progress",
			"active_job_id": busy.ID,
		})
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "title is required")
		return
	}
	switch req.Visibility {
	case domain.VisibilityPrivate, domain.VisibilityUnlisted, domain.VisibilityPublic:
	default:
		writeError(w, http.StatusBadRequest, "invalid_request", "visibility must be private, unlisted, or public")
		return
	}
	videoPath, err := media.ValidateRef(req.Video, s.cfg.AllowedMediaRoots, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_video", err.Error())
		return
	}
	req.Video.LocalPath = videoPath
	if req.Thumbnail != nil {
		thumbPath, err := media.ValidateRef(*req.Thumbnail, s.cfg.AllowedMediaRoots, false)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_thumbnail", err.Error())
			return
		}
		if thumbPath != "" {
			req.Thumbnail.LocalPath = thumbPath
		}
	}

	now := time.Now().UTC()
	jobID := job.NewID()
	j := &domain.Job{
		ID:             jobID,
		IdempotencyKey: idem,
		Status:         domain.JobAccepted,
		CurrentStep:    "queued",
		ProfileID:      req.ProfileID,
		Request:        req,
		ArtifactDir:    paths.JobArtifactsDir(s.layout, jobID),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.jobs.Create(j); err != nil {
		s.log.Error("create job", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not create job")
		return
	}
	if !s.executor.StartAsync(j.ID) {
		j.Status = domain.JobFailed
		j.ErrorCode = "busy"
		j.ErrorMessage = "executor slot unavailable"
		_ = s.jobs.Save(j)
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "busy",
			"message": "could not start upload executor",
			"job_id":  j.ID,
		})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"job":     j,
		"message": "upload started",
	})
}

func (s *Server) handleCurrentJob(w http.ResponseWriter, _ *http.Request) {
	j, err := s.jobs.FindRunning()
	if err != nil {
		if errors.Is(err, job.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"job": nil})
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not load job")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": j})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("job_id")
	j, err := s.jobs.Get(id)
	if err != nil {
		if errors.Is(err, job.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not load job")
		return
	}
	writeJSON(w, http.StatusOK, j)
}
