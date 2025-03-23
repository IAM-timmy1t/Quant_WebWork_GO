package discovery

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Client handles service registration and heartbeat
type Client struct {
	service  *Service
	registry *ServiceRegistry
	
	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}
}

// NewClient creates a new discovery client
func NewClient(registry *ServiceRegistry, service *Service) *Client {
	return &Client{
		service:  service,
		registry: registry,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start begins the service registration and heartbeat process
func (c *Client) Start(ctx context.Context) error {
	// Initial registration
	if err := c.registry.Register(c.service); err != nil {
		return fmt.Errorf("initial registration failed: %v", err)
	}

	// Start heartbeat in background
	go c.heartbeat(ctx)

	return nil
}

// Stop ends the service registration and heartbeat process
func (c *Client) Stop() {
	close(c.stopChan)
	<-c.doneChan
	
	// Deregister service
	if err := c.registry.Deregister(c.service.ID); err != nil {
		log.Printf("Error deregistering service: %v", err)
	}
}

// UpdateMetadata updates the service metadata
func (c *Client) UpdateMetadata(metadata ServiceMetadata) error {
	c.service.Metadata = metadata
	return c.registry.Update(c.service)
}

// UpdateStatus updates the service status
func (c *Client) UpdateStatus(status string) error {
	c.service.Status = status
	return c.registry.Update(c.service)
}

// heartbeat periodically updates the service's last updated time
func (c *Client) heartbeat(ctx context.Context) {
	defer close(c.doneChan)

	ticker := time.NewTicker(c.service.TTL / 3)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			if err := c.registry.Update(c.service); err != nil {
				log.Printf("Error updating service heartbeat: %v", err)
			}
		}
	}
}
