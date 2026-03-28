package bridge

import (
	"os"
	"sync"

	"github.com/plexusone/agentpair/pkg/jsonl"
)

// Storage handles persistent storage of bridge messages in JSONL format.
type Storage struct {
	path   string
	mu     sync.Mutex
	writer *jsonl.FileWriter
}

// NewStorage creates a new storage instance for the given file path.
func NewStorage(path string) *Storage {
	return &Storage{path: path}
}

// Open opens the storage file for appending.
func (s *Storage) Open() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer != nil {
		return nil
	}

	w, err := jsonl.OpenFile(s.path)
	if err != nil {
		return err
	}
	s.writer = w
	return nil
}

// Close closes the storage file.
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer == nil {
		return nil
	}

	err := s.writer.Close()
	s.writer = nil
	return err
}

// Append writes a message to the storage file.
func (s *Storage) Append(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer == nil {
		w, err := jsonl.OpenFile(s.path)
		if err != nil {
			return err
		}
		s.writer = w
	}

	if err := s.writer.Write(msg); err != nil {
		return err
	}
	return s.writer.Sync()
}

// ReadAll reads all messages from the storage file.
func (s *Storage) ReadAll() ([]*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgs, err := jsonl.ReadAll[*Message](s.path)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// ReadFiltered reads messages matching the given filter function.
func (s *Storage) ReadFiltered(filter func(*Message) bool) ([]*Message, error) {
	all, err := s.ReadAll()
	if err != nil {
		return nil, err
	}

	var filtered []*Message
	for _, msg := range all {
		if filter(msg) {
			filtered = append(filtered, msg)
		}
	}
	return filtered, nil
}

// Exists returns true if the storage file exists.
func (s *Storage) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// Path returns the storage file path.
func (s *Storage) Path() string {
	return s.path
}
