package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/d2hle/yt-automation/internal/paths"
)

var ErrNotFound = errors.New("profile not found")

type Store struct {
	layout paths.Layout
	mu     sync.Mutex
}

func NewStore(layout paths.Layout) *Store {
	return &Store{layout: layout}
}

func (s *Store) EnsureDirs() error {
	return os.MkdirAll(s.layout.Profiles, 0o700)
}

func (s *Store) Create(req domain.CreateProfileRequest) (*domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.layout.Profiles, req.ProfileID)
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("profile %q already exists", req.ProfileID)
	}
	if err := os.MkdirAll(paths.ProfileUserDataDir(s.layout, req.ProfileID), 0o700); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	p := &domain.Profile{
		ID:            req.ProfileID,
		DisplayName:   req.DisplayName,
		UserDataDir:   paths.ProfileUserDataDir(s.layout, req.ProfileID),
		SessionStatus: domain.SessionUnconfigured,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.save(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) Get(id string) (*domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load(id)
}

func (s *Store) List() ([]*domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.layout.Profiles)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*domain.Profile
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := s.load(e.Name())
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Store) UpdateStatus(id string, status domain.SessionStatus, accountHint string) (*domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.load(id)
	if err != nil {
		return nil, err
	}
	p.SessionStatus = status
	if accountHint != "" {
		p.AccountHint = accountHint
	}
	now := time.Now().UTC()
	p.UpdatedAt = now
	if status == domain.SessionActive {
		p.LastCheckedAt = &now
	}
	if err := s.save(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) UpdateAfterRefresh(id string, status domain.SessionStatus, accountHint string) (*domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.load(id)
	if err != nil {
		return nil, err
	}
	p.SessionStatus = status
	if accountHint != "" {
		p.AccountHint = accountHint
	}
	now := time.Now().UTC()
	p.UpdatedAt = now
	if status == domain.SessionActive {
		p.LastCheckedAt = &now
		p.LastRefreshedAt = &now
	}
	if err := s.save(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) AddToken(id string, token domain.APIToken) (*domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.load(id)
	if err != nil {
		return nil, err
	}
	p.Tokens = append(p.Tokens, token)
	p.UpdatedAt = time.Now().UTC()
	if err := s.save(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) RemoveToken(id, tokenID string) (*domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.load(id)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.APIToken, 0, len(p.Tokens))
	found := false
	for _, t := range p.Tokens {
		if t.ID == tokenID {
			found = true
			continue
		}
		filtered = append(filtered, t)
	}
	if !found {
		return nil, fmt.Errorf("token not found")
	}
	p.Tokens = filtered
	p.UpdatedAt = time.Now().UTC()
	if err := s.save(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) load(id string) (*domain.Profile, error) {
	meta := paths.ProfileMetaPath(s.layout, id)
	data, err := os.ReadFile(meta)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var p domain.Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) save(p *domain.Profile) error {
	meta := paths.ProfileMetaPath(s.layout, p.ID)
	if err := os.MkdirAll(filepath.Dir(meta), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(meta, data, 0o600)
}
