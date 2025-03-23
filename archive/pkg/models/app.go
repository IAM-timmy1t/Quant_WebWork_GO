package models

import (
	"errors"
	"time"
)

var (
	ErrAppNotFound = errors.New("app not found")
)

// App represents a registered application
type App struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Config      map[string]string `json:"config"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Validate checks if the app data is valid
func (a *App) Validate() error {
	if a.Name == "" {
		return errors.New("name is required")
	}
	if a.URL == "" {
		return errors.New("URL is required")
	}
	if a.Type == "" {
		return errors.New("type is required")
	}
	return nil
}

// IsActive checks if the app is currently active
func (a *App) IsActive() bool {
	return a.Status == "active"
}

// SetStatus updates the app's status
func (a *App) SetStatus(status string) {
	a.Status = status
	a.UpdatedAt = time.Now()
}

// UpdateConfig updates the app's configuration
func (a *App) UpdateConfig(config map[string]string) {
	a.Config = config
	a.UpdatedAt = time.Now()
}
