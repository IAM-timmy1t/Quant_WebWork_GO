package graphql

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/monitoring"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/auth"
)

// Resolver is the root resolver for GraphQL queries
type Resolver struct {
	monitor    *monitoring.Monitor
	auth       *auth.Authenticator
	eventBus   *EventBus
}

// NewResolver creates a new GraphQL resolver
func NewResolver(monitor *monitoring.Monitor, auth *auth.Authenticator) *Resolver {
	return &Resolver{
		monitor:  monitor,
		auth:     auth,
		eventBus: NewEventBus(),
	}
}

// SystemMetrics resolves the system metrics query
func (r *Resolver) SystemMetrics(ctx context.Context) (*SystemMetricsResolver, error) {
	metrics, err := r.monitor.GetSystemMetrics()
	if err != nil {
		return nil, err
	}
	return &SystemMetricsResolver{metrics: metrics}, nil
}

// ServiceMetrics resolves metrics for a specific service
func (r *Resolver) ServiceMetrics(ctx context.Context, args struct{ ServiceID graphql.ID }) (*ServiceMetricsResolver, error) {
	metrics, err := r.monitor.GetServiceMetrics(string(args.ServiceID))
	if err != nil {
		return nil, err
	}
	return &ServiceMetricsResolver{metrics: metrics}, nil
}

// AllServices resolves the list of all services
func (r *Resolver) AllServices(ctx context.Context) ([]*ServiceResolver, error) {
	services, err := r.monitor.GetAllServices()
	if err != nil {
		return nil, err
	}

	resolvers := make([]*ServiceResolver, len(services))
	for i, svc := range services {
		resolvers[i] = &ServiceResolver{service: svc}
	}
	return resolvers, nil
}

// Service resolves a specific service by ID
func (r *Resolver) Service(ctx context.Context, args struct{ ID graphql.ID }) (*ServiceResolver, error) {
	service, err := r.monitor.GetService(string(args.ID))
	if err != nil {
		return nil, err
	}
	return &ServiceResolver{service: service}, nil
}

// SecurityEvents resolves security events with optional filtering
func (r *Resolver) SecurityEvents(ctx context.Context, args struct {
	Limit    *int32
	Severity *string
}) ([]*SecurityEventResolver, error) {
	limit := int32(10)
	if args.Limit != nil {
		limit = *args.Limit
	}

	events, err := r.monitor.GetSecurityEvents(int(limit), args.Severity)
	if err != nil {
		return nil, err
	}

	resolvers := make([]*SecurityEventResolver, len(events))
	for i, event := range events {
		resolvers[i] = &SecurityEventResolver{event: event}
	}
	return resolvers, nil
}

// SecurityScore resolves the current security score
func (r *Resolver) SecurityScore(ctx context.Context) (float64, error) {
	return r.monitor.GetSecurityScore()
}

// Users resolves the list of users with optional role filtering
func (r *Resolver) Users(ctx context.Context, args struct{ Role *string }) ([]*UserResolver, error) {
	users, err := r.auth.GetUsers(args.Role)
	if err != nil {
		return nil, err
	}

	resolvers := make([]*UserResolver, len(users))
	for i, user := range users {
		resolvers[i] = &UserResolver{user: user}
	}
	return resolvers, nil
}

// User resolves a specific user by ID
func (r *Resolver) User(ctx context.Context, args struct{ ID graphql.ID }) (*UserResolver, error) {
	user, err := r.auth.GetUser(string(args.ID))
	if err != nil {
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// ResourceUsage resolves the current resource usage metrics
func (r *Resolver) ResourceUsage(ctx context.Context) (*ResourceUsageResolver, error) {
	usage, err := r.monitor.GetResourceUsage()
	if err != nil {
		return nil, err
	}
	return &ResourceUsageResolver{usage: usage}, nil
}

// NetworkTraffic resolves the current network traffic metrics
func (r *Resolver) NetworkTraffic(ctx context.Context) (*NetworkTrafficResolver, error) {
	traffic, err := r.monitor.GetNetworkTraffic()
	if err != nil {
		return nil, err
	}
	return &NetworkTrafficResolver{traffic: traffic}, nil
}

// Mutation resolvers

// RegisterService handles service registration
func (r *Resolver) RegisterService(ctx context.Context, args struct {
	Input RegisterServiceInput
}) (*ServiceResolver, error) {
	service, err := r.monitor.RegisterService(args.Input)
	if err != nil {
		return nil, err
	}
	return &ServiceResolver{service: service}, nil
}

// UpdateService handles service updates
func (r *Resolver) UpdateService(ctx context.Context, args struct {
	ID    graphql.ID
	Input UpdateServiceInput
}) (*ServiceResolver, error) {
	service, err := r.monitor.UpdateService(string(args.ID), args.Input)
	if err != nil {
		return nil, err
	}
	return &ServiceResolver{service: service}, nil
}

// DeleteService handles service deletion
func (r *Resolver) DeleteService(ctx context.Context, args struct{ ID graphql.ID }) (bool, error) {
	return r.monitor.DeleteService(string(args.ID))
}

// CreateUser handles user creation
func (r *Resolver) CreateUser(ctx context.Context, args struct {
	Input CreateUserInput
}) (*UserResolver, error) {
	user, err := r.auth.CreateUser(args.Input)
	if err != nil {
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// UpdateUser handles user updates
func (r *Resolver) UpdateUser(ctx context.Context, args struct {
	ID    graphql.ID
	Input UpdateUserInput
}) (*UserResolver, error) {
	user, err := r.auth.UpdateUser(string(args.ID), args.Input)
	if err != nil {
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// DeleteUser handles user deletion
func (r *Resolver) DeleteUser(ctx context.Context, args struct{ ID graphql.ID }) (bool, error) {
	return r.auth.DeleteUser(string(args.ID))
}

// Subscription resolvers

// SystemMetricsUpdated handles system metrics subscription
func (r *Resolver) SystemMetricsUpdated(ctx context.Context) (<-chan *SystemMetricsResolver, error) {
	updates := make(chan *SystemMetricsResolver)
	r.eventBus.Subscribe("system_metrics", func(data interface{}) {
		if metrics, ok := data.(monitoring.SystemMetrics); ok {
			updates <- &SystemMetricsResolver{metrics: metrics}
		}
	})
	return updates, nil
}

// ServiceStatusChanged handles service status subscription
func (r *Resolver) ServiceStatusChanged(ctx context.Context) (<-chan *ServiceResolver, error) {
	updates := make(chan *ServiceResolver)
	r.eventBus.Subscribe("service_status", func(data interface{}) {
		if service, ok := data.(monitoring.Service); ok {
			updates <- &ServiceResolver{service: service}
		}
	})
	return updates, nil
}

// SecurityEventOccurred handles security event subscription
func (r *Resolver) SecurityEventOccurred(ctx context.Context) (<-chan *SecurityEventResolver, error) {
	updates := make(chan *SecurityEventResolver)
	r.eventBus.Subscribe("security_event", func(data interface{}) {
		if event, ok := data.(monitoring.SecurityEvent); ok {
			updates <- &SecurityEventResolver{event: event}
		}
	})
	return updates, nil
}

// Helper types and resolvers

type SystemMetricsResolver struct {
	metrics monitoring.SystemMetrics
}

type ServiceMetricsResolver struct {
	metrics monitoring.ServiceMetrics
}

type ServiceResolver struct {
	service monitoring.Service
}

type SecurityEventResolver struct {
	event monitoring.SecurityEvent
}

type UserResolver struct {
	user auth.User
}

type ResourceUsageResolver struct {
	usage monitoring.ResourceUsage
}

type NetworkTrafficResolver struct {
	traffic monitoring.NetworkTraffic
}

// EventBus handles real-time event subscriptions
type EventBus struct {
	subscribers map[string][]func(interface{})
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(interface{})),
	}
}

func (eb *EventBus) Subscribe(event string, callback func(interface{})) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[event] = append(eb.subscribers[event], callback)
}

func (eb *EventBus) Publish(event string, data interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	if callbacks, ok := eb.subscribers[event]; ok {
		for _, callback := range callbacks {
			go callback(data)
		}
	}
}

