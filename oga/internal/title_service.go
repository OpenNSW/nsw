package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// TitleService handles human-readable titles for tasks/applications
type TitleService interface {
	GetTitle(lookupKey string) string
	Reload() error
}

type titleService struct {
	titlesPath string
	titles     map[string]string
	mu         sync.RWMutex
}

// NewTitleService creates a new title service that loads mappings from a JSON file
func NewTitleService(titlesPath string) (TitleService, error) {
	s := &titleService{
		titlesPath: titlesPath,
		titles:     make(map[string]string),
	}

	if err := s.Reload(); err != nil {
		// If file doesn't exist, we just start with empty mapping
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}

	return s, nil
}

func (s *titleService) GetTitle(lookupKey string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if title, ok := s.titles[lookupKey]; ok {
		return title
	}

	return ""
}

func (s *titleService) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.titlesPath == "" {
		return nil
	}

	data, err := os.ReadFile(s.titlesPath)
	if err != nil {
		return err
	}

	var titles map[string]string
	if err := json.Unmarshal(data, &titles); err != nil {
		return fmt.Errorf("failed to unmarshal titles: %w", err)
	}

	s.titles = titles
	return nil
}
