package metrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// DatabaseMetrics holds the metrics related to database performance
type DatabaseMetrics struct {
	dbConnections    *prometheus.GaugeVec
	queryDurations   *prometheus.HistogramVec
	activeQueries    prometheus.Gauge
	errorQueries     *prometheus.CounterVec
	queryCount       *prometheus.CounterVec
	deadlocks        prometheus.Counter
	replicationLag   prometheus.Gauge
	dbStatsCollector *prometheus.GaugeVec
	db               *sql.DB
	logger           *zap.SugaredLogger
}

// NewDatabaseMetrics creates a new database metrics collector
func NewDatabaseMetrics(db *sql.DB, logger *zap.SugaredLogger) *DatabaseMetrics {
	dm := &DatabaseMetrics{
		db:     db,
		logger: logger,
		dbConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connections",
				Help: "Number of database connections by state",
			},
			[]string{"state"},
		),
		queryDurations: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Duration of database queries",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // from 1ms to ~1s
			},
			[]string{"query_type", "table"},
		),
		activeQueries: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_active_queries",
				Help: "Number of active database queries",
			},
		),
		errorQueries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_error_queries_total",
				Help: "Total number of database queries that resulted in an error",
			},
			[]string{"query_type", "error_class"},
		),
		queryCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_query_total",
				Help: "Total number of database queries by type",
			},
			[]string{"query_type", "table"},
		),
		deadlocks: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_deadlocks_total",
				Help: "Total number of deadlocks detected",
			},
		),
		replicationLag: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_replication_lag_seconds",
				Help: "Replication lag in seconds",
			},
		),
		dbStatsCollector: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_stats",
				Help: "Database connection pool statistics",
			},
			[]string{"stat"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		dm.dbConnections,
		dm.queryDurations,
		dm.activeQueries,
		dm.errorQueries,
		dm.queryCount,
		dm.deadlocks,
		dm.replicationLag,
		dm.dbStatsCollector,
	)

	return dm
}

// StartCollecting starts collecting database metrics
func (dm *DatabaseMetrics) StartCollecting(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				dm.collectMetrics()
			}
		}
	}()
}

// collectMetrics collects all database metrics
func (dm *DatabaseMetrics) collectMetrics() {
	dm.collectConnectionStats()
	dm.collectActiveQueries()
	dm.collectReplicationStats()
}

// collectConnectionStats collects database connection statistics
func (dm *DatabaseMetrics) collectConnectionStats() {
	stats := dm.db.Stats()

	dm.dbStatsCollector.WithLabelValues("max_open_connections").Set(float64(stats.MaxOpenConnections))
	dm.dbStatsCollector.WithLabelValues("open_connections").Set(float64(stats.OpenConnections))
	dm.dbStatsCollector.WithLabelValues("in_use").Set(float64(stats.InUse))
	dm.dbStatsCollector.WithLabelValues("idle").Set(float64(stats.Idle))
	dm.dbStatsCollector.WithLabelValues("wait_count").Set(float64(stats.WaitCount))
	dm.dbStatsCollector.WithLabelValues("wait_duration_ms").Set(float64(stats.WaitDuration.Milliseconds()))
	dm.dbStatsCollector.WithLabelValues("max_idle_closed").Set(float64(stats.MaxIdleClosed))
	dm.dbStatsCollector.WithLabelValues("max_lifetime_closed").Set(float64(stats.MaxLifetimeClosed))

	dm.dbConnections.WithLabelValues("open").Set(float64(stats.OpenConnections))
	dm.dbConnections.WithLabelValues("in_use").Set(float64(stats.InUse))
	dm.dbConnections.WithLabelValues("idle").Set(float64(stats.Idle))
}

// collectActiveQueries collects the number of active queries
func (dm *DatabaseMetrics) collectActiveQueries() {
	var activeQueries int

	// This query works for PostgreSQL
	row := dm.db.QueryRowContext(context.Background(), 
		"SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active' AND pid <> pg_backend_pid()")
	
	if err := row.Scan(&activeQueries); err != nil {
		dm.logger.Warnw("Failed to collect active queries", "error", err)
		return
	}

	dm.activeQueries.Set(float64(activeQueries))
}

// collectReplicationStats collects replication statistics
func (dm *DatabaseMetrics) collectReplicationStats() {
	var replicationLag float64

	// This query works for PostgreSQL replicas
	row := dm.db.QueryRowContext(context.Background(), 
		"SELECT EXTRACT(EPOCH FROM (NOW() - pg_last_xact_replay_timestamp())) AS lag")
	
	if err := row.Scan(&replicationLag); err != nil {
		// This might fail on non-replica databases, which is expected
		dm.logger.Debugw("Failed to collect replication lag", "error", err)
		return
	}

	dm.replicationLag.Set(replicationLag)
}

// RecordQuery records a database query execution and duration
func (dm *DatabaseMetrics) RecordQuery(queryType, table string, duration time.Duration) {
	dm.queryDurations.WithLabelValues(queryType, table).Observe(duration.Seconds())
	dm.queryCount.WithLabelValues(queryType, table).Inc()
}

// RecordQueryError records a database query error
func (dm *DatabaseMetrics) RecordQueryError(queryType, errorClass string) {
	dm.errorQueries.WithLabelValues(queryType, errorClass).Inc()
}

// RecordDeadlock records a database deadlock
func (dm *DatabaseMetrics) RecordDeadlock() {
	dm.deadlocks.Inc()
}

// WrapDB wraps database calls to automatically record metrics
func (dm *DatabaseMetrics) WrapDB(db *sql.DB) *InstrumentedDB {
	return &InstrumentedDB{
		DB:     db,
		metrics: dm,
	}
}

// InstrumentedDB is a wrapper around sql.DB that automatically records metrics
type InstrumentedDB struct {
	*sql.DB
	metrics *DatabaseMetrics
}

// QueryContext wraps sql.DB.QueryContext with metrics
func (idb *InstrumentedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := idb.DB.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	// Simplified query type detection - in production this would be more sophisticated
	queryType := detectQueryType(query)
	table := detectTable(query)

	idb.metrics.RecordQuery(queryType, table, duration)
	
	if err != nil {
		errorClass := classifyError(err)
		idb.metrics.RecordQueryError(queryType, errorClass)
	}
	
	return rows, err
}

// ExecContext wraps sql.DB.ExecContext with metrics
func (idb *InstrumentedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := idb.DB.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	queryType := detectQueryType(query)
	table := detectTable(query)

	idb.metrics.RecordQuery(queryType, table, duration)
	
	if err != nil {
		errorClass := classifyError(err)
		idb.metrics.RecordQueryError(queryType, errorClass)
	}
	
	return result, err
}

// Helper functions to classify queries
func detectQueryType(query string) string {
	// This is a simplified version - in production would use a proper SQL parser
	lowercaseQuery := query[:min(len(query), 20)]
	switch {
	case containsAny(lowercaseQuery, "select"):
		return "select"
	case containsAny(lowercaseQuery, "insert"):
		return "insert"
	case containsAny(lowercaseQuery, "update"):
		return "update"
	case containsAny(lowercaseQuery, "delete"):
		return "delete"
	default:
		return "other"
	}
}

func detectTable(query string) string {
	// This is a simplified version - in production would use a proper SQL parser
	// In reality, this would need to be more sophisticated to handle JOINs, subqueries, etc.
	return "unknown" // Placeholder for a real implementation
}

func classifyError(err error) string {
	// In a real implementation, we would classify errors more precisely
	return "general_error"
}

func containsAny(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
