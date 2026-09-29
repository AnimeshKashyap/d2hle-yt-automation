package job

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

var ErrNotFound = errors.New("job not found")

type Store struct {
	layout paths.Layout
	mu     sync.Mutex
}

func NewStore(layout paths.Layout) *Store {
	return &Store{layout: layout}
}

func (s *Store) EnsureDirs() error {
	return os.MkdirAll(s.layout.Jobs, 0o700)
}

func (s *Store) Create(job *domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(job)
}

func (s *Store) Save(job *domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(job)
}

func (s *Store) FindRunningForProfile(profileID string) (*domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.layout.Jobs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		j, err := s.load(trimJSONExt(e.Name()))
		if err != nil {
			continue
		}
		if j.ProfileID != profileID {
			continue
		}
		switch j.Status {
		case domain.JobRunning, domain.JobAwaitingAuth, domain.JobProcessing, domain.JobAccepted:
			return j, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) Get(id string) (*domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load(id)
}

func (s *Store) GetByIdempotencyKey(key string) (*domain.Job, error) {
	if key == "" {
		return nil, ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.layout.Jobs)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		j, err := s.load(trimJSONExt(e.Name()))
		if err != nil {
			continue
		}
		if j.IdempotencyKey == key {
			return j, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) FindRunning() (*domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.layout.Jobs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		j, err := s.load(trimJSONExt(e.Name()))
		if err != nil {
			continue
		}
		switch j.Status {
		case domain.JobAccepted, domain.JobRunning, domain.JobAwaitingAuth, domain.JobProcessing:
			return j, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) save(job *domain.Job) error {
	if err := os.MkdirAll(s.layout.Jobs, 0o700); err != nil {
		return err
	}
	path := paths.JobPath(s.layout, job.ID)
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *Store) load(id string) (*domain.Job, error) {
	path := paths.JobPath(s.layout, id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var j domain.Job
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return &j, nil
}

func trimJSONExt(name string) string {
	return name[:len(name)-len(".json")]
}

func NewID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
