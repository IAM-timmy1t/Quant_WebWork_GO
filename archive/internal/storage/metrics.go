package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/monitoring"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/tracing"
	"go.etcd.io/bbolt"
)

// MetricsStorage manages persistent storage of metrics data
type MetricsStorage struct {
	mu sync.RWMutex

	db     *bbolt.DB
	tracer *tracing.Tracer
	config MetricsStorageConfig

	// Batch processing
	batchMu     sync.Mutex
	metricsBatch []monitoring.ResourceMetrics
	eventsBatch  []monitoring.SecurityEvent

	// Bucket names
	systemBucket   []byte
	diskBucket     []byte
	networkBucket  []byte
	securityBucket []byte
	
	// Aggregation buckets
	hourlyBucket   []byte
	dailyBucket    []byte
	monthlyBucket  []byte
}

// MetricsStorageConfig defines configuration for metrics storage
type MetricsStorageConfig struct {
	DBPath          string        `json:"dbPath"`
	SyncInterval    time.Duration `json:"syncInterval"`
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	MaxBatchSize    int          `json:"maxBatchSize"`
	
	// Compression settings
	CompressionEnabled bool    `json:"compressionEnabled"`
	CompressionLevel   int     `json:"compressionLevel"`
	
	// Aggregation settings
	AggregationEnabled bool          `json:"aggregationEnabled"`
	HourlyRetention   time.Duration `json:"hourlyRetention"`
	DailyRetention    time.Duration `json:"dailyRetention"`
	MonthlyRetention  time.Duration `json:"monthlyRetention"`
}

// NewMetricsStorage creates a new metrics storage
func NewMetricsStorage(config MetricsStorageConfig, tracer *tracing.Tracer) (*MetricsStorage, error) {
	if config.MaxBatchSize <= 0 {
		config.MaxBatchSize = 100 // Default batch size
	}

	// Open BoltDB database
	db, err := bbolt.Open(config.DBPath, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	ms := &MetricsStorage{
		db:     db,
		tracer: tracer,
		config: config,
		
		systemBucket:   []byte("system_metrics"),
		diskBucket:     []byte("disk_metrics"),
		networkBucket:  []byte("network_metrics"),
		securityBucket: []byte("security_events"),
		hourlyBucket:   []byte("hourly_metrics"),
		dailyBucket:    []byte("daily_metrics"),
		monthlyBucket:  []byte("monthly_metrics"),
		
		metricsBatch: make([]monitoring.ResourceMetrics, 0, config.MaxBatchSize),
		eventsBatch:  make([]monitoring.SecurityEvent, 0, config.MaxBatchSize),
	}

	// Create buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := [][]byte{
			ms.systemBucket,
			ms.diskBucket,
			ms.networkBucket,
			ms.securityBucket,
			ms.hourlyBucket,
			ms.dailyBucket,
			ms.monthlyBucket,
		}
		
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %v", bucket, err)
			}
		}
		return nil
	})
	
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %v", err)
	}

	// Start background tasks
	go ms.periodicFlush(config.SyncInterval)
	go ms.periodicCleanup(config.RetentionPeriod)
	if config.AggregationEnabled {
		go ms.periodicAggregation()
	}

	return ms, nil
}

// StoreMetrics adds metrics to batch or triggers immediate storage
func (ms *MetricsStorage) StoreMetrics(ctx context.Context, metrics monitoring.ResourceMetrics) error {
	ms.batchMu.Lock()
	ms.metricsBatch = append(ms.metricsBatch, metrics)
	shouldFlush := len(ms.metricsBatch) >= ms.config.MaxBatchSize
	ms.batchMu.Unlock()

	if shouldFlush {
		return ms.FlushMetricsBatch(ctx)
	}
	return nil
}

// FlushMetricsBatch stores the current batch of metrics
func (ms *MetricsStorage) FlushMetricsBatch(ctx context.Context) error {
	ms.batchMu.Lock()
	if len(ms.metricsBatch) == 0 {
		ms.batchMu.Unlock()
		return nil
	}
	
	batch := ms.metricsBatch
	ms.metricsBatch = make([]monitoring.ResourceMetrics, 0, ms.config.MaxBatchSize)
	ms.batchMu.Unlock()

	return ms.db.Batch(func(tx *bbolt.Tx) error {
		for _, metrics := range batch {
			if err := ms.storeMetricsInTx(tx, metrics); err != nil {
				return err
			}
		}
		return nil
	})
}

// storeMetricsInTx stores a single metrics record within a transaction
func (ms *MetricsStorage) storeMetricsInTx(tx *bbolt.Tx, metrics monitoring.ResourceMetrics) error {
	// Store system metrics
	if err := ms.storeSystemMetrics(tx, metrics); err != nil {
		return err
	}

	// Store disk metrics
	if err := ms.storeDiskMetrics(tx, metrics.Disk); err != nil {
		return err
	}

	// Store network metrics
	if err := ms.storeNetworkMetrics(tx, metrics.Network); err != nil {
		return err
	}

	return nil
}

// compressData compresses data using gzip
func (ms *MetricsStorage) compressData(data []byte) ([]byte, error) {
	if !ms.config.CompressionEnabled {
		return data, nil
	}

	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, ms.config.CompressionLevel)
	if err != nil {
		return nil, err
	}
	
	if _, err := gw.Write(data); err != nil {
		return nil, err
	}
	
	if err := gw.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompressData decompresses gzipped data
func (ms *MetricsStorage) decompressData(data []byte) ([]byte, error) {
	if !ms.config.CompressionEnabled {
		return data, nil
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gr); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// periodicFlush periodically flushes batched metrics
func (ms *MetricsStorage) periodicFlush(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		if err := ms.FlushMetricsBatch(context.Background()); err != nil {
			ms.tracer.RecordError(context.Background(), fmt.Errorf("periodic flush error: %v", err))
		}
	}
}

// periodicAggregation performs periodic metric aggregation
func (ms *MetricsStorage) periodicAggregation() {
	hourlyTicker := time.NewTicker(time.Hour)
	dailyTicker := time.NewTicker(24 * time.Hour)
	monthlyTicker := time.NewTicker(30 * 24 * time.Hour)
	
	defer hourlyTicker.Stop()
	defer dailyTicker.Stop()
	defer monthlyTicker.Stop()

	for {
		select {
		case <-hourlyTicker.C:
			if err := ms.aggregateMetrics("hourly"); err != nil {
				ms.tracer.RecordError(context.Background(), fmt.Errorf("hourly aggregation error: %v", err))
			}
		case <-dailyTicker.C:
			if err := ms.aggregateMetrics("daily"); err != nil {
				ms.tracer.RecordError(context.Background(), fmt.Errorf("daily aggregation error: %v", err))
			}
		case <-monthlyTicker.C:
			if err := ms.aggregateMetrics("monthly"); err != nil {
				ms.tracer.RecordError(context.Background(), fmt.Errorf("monthly aggregation error: %v", err))
			}
		}
	}
}

// StoreSecurityEvent stores a security event
func (ms *MetricsStorage) StoreSecurityEvent(ctx context.Context, event monitoring.SecurityEvent) error {
	ms.batchMu.Lock()
	ms.eventsBatch = append(ms.eventsBatch, event)
	shouldFlush := len(ms.eventsBatch) >= ms.config.MaxBatchSize
	ms.batchMu.Unlock()

	if shouldFlush {
		return ms.FlushEventsBatch(ctx)
	}
	return nil
}

// FlushEventsBatch stores the current batch of security events
func (ms *MetricsStorage) FlushEventsBatch(ctx context.Context) error {
	ms.batchMu.Lock()
	if len(ms.eventsBatch) == 0 {
		ms.batchMu.Unlock()
		return nil
	}
	
	batch := ms.eventsBatch
	ms.eventsBatch = make([]monitoring.SecurityEvent, 0, ms.config.MaxBatchSize)
	ms.batchMu.Unlock()

	return ms.db.Batch(func(tx *bbolt.Tx) error {
		for _, event := range batch {
			if err := ms.storeSecurityEventInTx(tx, event); err != nil {
				return err
			}
		}
		return nil
	})
}

// storeSecurityEventInTx stores a single security event within a transaction
func (ms *MetricsStorage) storeSecurityEventInTx(tx *bbolt.Tx, event monitoring.SecurityEvent) error {
	b := tx.Bucket(ms.securityBucket)
	key := []byte(fmt.Sprintf("%s_%s", event.Source, event.Timestamp.Format(time.RFC3339Nano)))
	
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}
	
	value, err = ms.compressData(value)
	if err != nil {
		return err
	}
	
	return b.Put(key, value)
}

// GetMetricsRange retrieves metrics for a specific time range
func (ms *MetricsStorage) GetMetricsRange(ctx context.Context, start, end time.Time) (*monitoring.ResourceMetrics, error) {
	var metrics monitoring.ResourceMetrics

	err := ms.db.View(func(tx *bbolt.Tx) error {
		// Get system metrics
		b := tx.Bucket(ms.systemBucket)
		c := b.Cursor()

		min := []byte(start.Format(time.RFC3339Nano))
		max := []byte(end.Format(time.RFC3339Nano))

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			var metric monitoring.ResourceMetrics
			v, err := ms.decompressData(v)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(v, &metric); err != nil {
				return err
			}
			metrics = metric // We'll take the latest one in the range
		}

		return nil
	})

	if err != nil {
		ms.tracer.RecordError(ctx, err)
		return nil, err
	}

	return &metrics, nil
}

// GetSecurityEvents retrieves security events for a time range
func (ms *MetricsStorage) GetSecurityEvents(ctx context.Context, start, end time.Time, limit int) ([]monitoring.SecurityEvent, error) {
	var events []monitoring.SecurityEvent

	err := ms.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(ms.securityBucket)
		c := b.Cursor()

		for k, v := c.Last(); k != nil && len(events) < limit; k, v = c.Prev() {
			var event monitoring.SecurityEvent
			v, err := ms.decompressData(v)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(v, &event); err != nil {
				return err
			}

			if event.Timestamp.Before(start) || event.Timestamp.After(end) {
				continue
			}

			events = append(events, event)
		}

		return nil
	})

	if err != nil {
		ms.tracer.RecordError(ctx, err)
		return nil, err
	}

	return events, nil
}

// Close closes the database
func (ms *MetricsStorage) Close() error {
	return ms.db.Close()
}

// periodicCleanup removes old metrics periodically
func (ms *MetricsStorage) periodicCleanup(retentionPeriod time.Duration) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := ms.cleanup(retentionPeriod); err != nil {
			// Log error but continue
			fmt.Printf("Error during metrics cleanup: %v\n", err)
		}
	}
}

// cleanup removes metrics older than the retention period
func (ms *MetricsStorage) cleanup(retentionPeriod time.Duration) error {
	cutoff := time.Now().Add(-retentionPeriod)

	return ms.db.Update(func(tx *bbolt.Tx) error {
		buckets := [][]byte{ms.systemBucket, ms.diskBucket, ms.networkBucket, ms.securityBucket, ms.hourlyBucket, ms.dailyBucket, ms.monthlyBucket}
		
		for _, bucket := range buckets {
			b := tx.Bucket(bucket)
			c := b.Cursor()

			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				ts := bytes.Split(k, []byte("_"))
				if len(ts) < 2 {
					continue
				}

				timestamp, err := time.Parse(time.RFC3339Nano, string(ts[len(ts)-1]))
				if err != nil {
					continue
				}

				if timestamp.Before(cutoff) {
					if err := c.Delete(); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
}

func (ms *MetricsStorage) storeSystemMetrics(tx *bbolt.Tx, metrics monitoring.ResourceMetrics) error {
	b := tx.Bucket(ms.systemBucket)
	key := []byte(metrics.Timestamp.Format(time.RFC3339Nano))
	
	value, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	
	value, err = ms.compressData(value)
	if err != nil {
		return err
	}
	
	return b.Put(key, value)
}

func (ms *MetricsStorage) storeDiskMetrics(tx *bbolt.Tx, metrics []monitoring.DiskMetrics) error {
	b := tx.Bucket(ms.diskBucket)
	
	for _, metric := range metrics {
		key := []byte(fmt.Sprintf("%s_%s", metric.Path, metric.Timestamp.Format(time.RFC3339Nano)))
		value, err := json.Marshal(metric)
		if err != nil {
			return err
		}
		
		value, err = ms.compressData(value)
		if err != nil {
			return err
		}
		
		if err := b.Put(key, value); err != nil {
			return err
		}
	}
	
	return nil
}

func (ms *MetricsStorage) storeNetworkMetrics(tx *bbolt.Tx, metrics []monitoring.NetworkMetrics) error {
	b := tx.Bucket(ms.networkBucket)
	
	for _, metric := range metrics {
		key := []byte(fmt.Sprintf("%s_%s", metric.Interface, metric.Timestamp.Format(time.RFC3339Nano)))
		value, err := json.Marshal(metric)
		if err != nil {
			return err
		}
		
		value, err = ms.compressData(value)
		if err != nil {
			return err
		}
		
		if err := b.Put(key, value); err != nil {
			return err
		}
	}
	
	return nil
}

func (ms *MetricsStorage) aggregateMetrics(aggregationType string) error {
	var bucket []byte
	switch aggregationType {
	case "hourly":
		bucket = ms.hourlyBucket
	case "daily":
		bucket = ms.dailyBucket
	case "monthly":
		bucket = ms.monthlyBucket
	default:
		return fmt.Errorf("unknown aggregation type: %s", aggregationType)
	}

	return ms.db.Update(func(tx *bbolt.Tx) error {
		// Aggregate metrics
		b := tx.Bucket(bucket)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			// Aggregate metrics here
		}

		return nil
	})
}

