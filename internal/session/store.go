package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store handles session persistence to disk.
type Store struct {
	mu    sync.RWMutex
	dir   string
}

// NewStore creates a new session store.
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dataDir}, nil
}

// sessionData is the persisted representation of a session.
type sessionData struct {
	ID          string        `json:"id"`
	UserTask    string        `json:"userTask"`
	RepoPath    string        `json:"repoPath"`
	Status      SessionStatus `json:"status"`
	CreatedAt   string        `json:"createdAt"`
	StartedAt   *string       `json:"startedAt,omitempty"`
	CompletedAt *string       `json:"completedAt,omitempty"`
}

// Save saves a session to disk.
func (s *Store) Save(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess.mu.RLock()
	defer sess.mu.RUnlock()

	data := sessionData{
		ID:        sess.ID,
		UserTask:  sess.UserTask,
		RepoPath:  sess.RepoPath,
		Status:    sess.Status,
		CreatedAt: sess.CreatedAt.Format(timeFormat),
	}

	if sess.StartedAt != nil {
		t := sess.StartedAt.Format(timeFormat)
		data.StartedAt = &t
	}
	if sess.CompletedAt != nil {
		t := sess.CompletedAt.Format(timeFormat)
		data.CompletedAt = &t
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	path := s.sessionPath(sess.ID)
	return os.WriteFile(path, bytes, 0644)
}

// LoadAll loads all sessions from disk.
func (s *Store) LoadAll() ([]sessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []sessionData
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.dir, entry.Name())
		bytes, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var data sessionData
		if err := json.Unmarshal(bytes, &data); err != nil {
			continue
		}
		sessions = append(sessions, data)
	}
	return sessions, nil
}

// Delete removes a session from disk.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.sessionPath(id)
	return os.Remove(path)
}

// sessionPath returns the file path for a session.
func (s *Store) sessionPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

const timeFormat = "2006-01-02T15:04:05.000000000Z07:00"
