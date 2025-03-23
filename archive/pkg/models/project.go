package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Project represents a deployed project
type Project struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Config      map[string]string `json:"config"`
	Path        string            `json:"path"`
	Status      string            `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Endpoints   []string         `json:"endpoints"`
	Health      *HealthStatus    `json:"health"`
}

// HealthStatus represents the health status of a project
type HealthStatus struct {
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
	Message   string    `json:"message"`
}

// ProjectStore manages project persistence
type ProjectStore struct {
	storePath string
	mu        sync.RWMutex
	projects  map[string]*Project
}

// NewProjectStore creates a new project store
func NewProjectStore(storePath string) (*ProjectStore, error) {
	store := &ProjectStore{
		storePath: storePath,
		projects:  make(map[string]*Project),
	}

	// Create store directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}

	// Load existing projects
	if err := store.load(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %v", err)
	}

	return store, nil
}

// Create adds a new project
func (s *ProjectStore) Create(project *Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.projects[project.Name]; exists {
		return fmt.Errorf("project %s already exists", project.Name)
	}

	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	project.Health = &HealthStatus{
		Status:    "unknown",
		LastCheck: now,
	}

	s.projects[project.Name] = project
	return s.save()
}

// Get retrieves a project by name
func (s *ProjectStore) Get(name string) (*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, exists := s.projects[name]
	if !exists {
		return nil, fmt.Errorf("project %s not found", name)
	}

	return project, nil
}

// List returns all projects
func (s *ProjectStore) List() ([]*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]*Project, 0, len(s.projects))
	for _, project := range s.projects {
		projects = append(projects, project)
	}

	return projects, nil
}

// Update updates an existing project
func (s *ProjectStore) Update(project *Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.projects[project.Name]; !exists {
		return fmt.Errorf("project %s not found", project.Name)
	}

	project.UpdatedAt = time.Now()
	s.projects[project.Name] = project
	return s.save()
}

// Delete removes a project
func (s *ProjectStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.projects[name]; !exists {
		return fmt.Errorf("project %s not found", name)
	}

	delete(s.projects, name)
	return s.save()
}

// UpdateHealth updates a project's health status
func (s *ProjectStore) UpdateHealth(name string, status string, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, exists := s.projects[name]
	if !exists {
		return fmt.Errorf("project %s not found", name)
	}

	project.Health = &HealthStatus{
		Status:    status,
		LastCheck: time.Now(),
		Message:   message,
	}

	return s.save()
}

// Internal methods

func (s *ProjectStore) load() error {
	data, err := os.ReadFile(s.storePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.projects)
}

func (s *ProjectStore) save() error {
	data, err := json.MarshalIndent(s.projects, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.storePath, data, 0644)
}
