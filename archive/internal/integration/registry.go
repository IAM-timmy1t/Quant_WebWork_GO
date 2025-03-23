package integration

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/timot/Quant_WebWork_GO/pkg/models"
)

// Registry manages registered applications and services
type Registry struct {
	apps    map[string]*models.App
	mu      sync.RWMutex
	eventBus EventBus
}

// NewRegistry creates a new application registry
func NewRegistry(eventBus EventBus) *Registry {
	return &Registry{
		apps:     make(map[string]*models.App),
		eventBus: eventBus,
	}
}

// RegisterApp adds a new application to the registry
func (r *Registry) RegisterApp(ctx context.Context, app *models.App) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate UUID if not provided
	if app.ID == "" {
		app.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	app.CreatedAt = now
	app.UpdatedAt = now

	// Store app
	r.apps[app.ID] = app

	// Publish event
	r.eventBus.Publish("app.registered", app)

	return nil
}

// GetApp retrieves an application by ID
func (r *Registry) GetApp(ctx context.Context, id string) (*models.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	app, exists := r.apps[id]
	if !exists {
		return nil, models.ErrAppNotFound
	}

	return app, nil
}

// ListApps returns all registered applications
func (r *Registry) ListApps(ctx context.Context) ([]*models.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apps := make([]*models.App, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}

	return apps, nil
}

// UpdateApp updates an existing application
func (r *Registry) UpdateApp(ctx context.Context, app *models.App) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.apps[app.ID]; !exists {
		return models.ErrAppNotFound
	}

	app.UpdatedAt = time.Now()
	r.apps[app.ID] = app

	// Publish event
	r.eventBus.Publish("app.updated", app)

	return nil
}

// DeleteApp removes an application from the registry
func (r *Registry) DeleteApp(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, exists := r.apps[id]
	if !exists {
		return models.ErrAppNotFound
	}

	delete(r.apps, id)

	// Publish event
	r.eventBus.Publish("app.deleted", app)

	return nil
}

// GetAppByName finds an application by its name
func (r *Registry) GetAppByName(ctx context.Context, name string) (*models.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, app := range r.apps {
		if app.Name == name {
			return app, nil
		}
	}

	return nil, models.ErrAppNotFound
}

// CheckHealth verifies the health status of all registered applications
func (r *Registry) CheckHealth(ctx context.Context) map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := make(map[string]string)
	for id, app := range r.apps {
		// TODO: Implement actual health checking logic
		health[id] = app.Status
	}

	return health
}
