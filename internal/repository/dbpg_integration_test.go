//go:build integration

package repository

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mersikovs/korob.git/internal/config/db"
	"github.com/mersikovs/korob.git/internal/database"
	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgresContainer(tb testing.TB) (*postgres.PostgresContainer, string) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		tb.Fatalf("failed to start postgres container: %v", err)
	}

	tb.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			tb.Logf("failed to terminate container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		tb.Fatalf("failed to get connection string: %v", err)
	}

	if err := database.MigrateUp(dsn, "file://../../migrations"); err != nil {
		tb.Fatalf("Ошибка миграции базы данных: %v", err)
	}

	return container, dsn
}

func TestNewPgStorage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	logger, err := logger.NewZap("info")
	if err != nil {
		t.Fatalf("failed to create pg logger: %v", err)
	}

	_, dsn := setupPostgresContainer(t)

	tests := []struct {
		name      string
		dsn       string
		wantError bool
	}{
		{
			name:      "successful connection",
			dsn:       dsn,
			wantError: false,
		},
		{
			name:      "invalid host",
			dsn:       "postgres://user:pass@invalid-host:5432/db?sslmode=disable",
			wantError: true,
		},
		{
			name:      "invalid credentials",
			dsn:       "postgres://invalid:invalid@localhost:5432/db?sslmode=disable",
			wantError: true,
		},
		{
			name:      "invalid port",
			dsn:       "postgres://user:pass@localhost:9999/db?sslmode=disable",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewPgStorage(context.Background(), tt.dsn, logger)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if storage != nil {
					t.Fatal("expected nil storage on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			defer storage.Close()

			pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := storage.Ping(pingCtx); err != nil {
				t.Fatalf("failed to ping storage: %v", err)
			}
		})
	}
}

func TestPgStorage_GetNamesList_TableDriven(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	_, dsn := setupPostgresContainer(t)
	ctx := context.Background()

	logger, err := logger.NewZap("info")
	if err != nil {
		t.Fatalf("failed to create pg logger: %v", err)
	}

	storage, err := NewPgStorage(ctx, dsn, logger)
	if err != nil {
		t.Fatalf("failed to create pg storage: %v", err)
	}
	defer storage.Close()

	bigTable := make([]string, 0)
	for i := range 1000 {
		bigTable = append(bigTable, "v"+strconv.Itoa(i))
	}

	tests := []struct {
		name       string
		ctxTimeout time.Duration
		setupData  []string
		wantIDs    []string
		wantErr    bool
	}{
		{
			name:       "empty table",
			ctxTimeout: 1 * time.Second,
			setupData:  []string{},
			wantIDs:    []string{},
			wantErr:    false,
		},
		{
			name:       "single metric",
			ctxTimeout: 1 * time.Second,
			setupData:  []string{"metric1"},
			wantIDs:    []string{"metric1"},
			wantErr:    false,
		},
		{
			name:       "multiple metrics",
			ctxTimeout: 1 * time.Second,
			setupData:  []string{"metric-a", "metric-b", "metric-c"},
			wantIDs:    []string{"metric-a", "metric-b", "metric-c"},
			wantErr:    false,
		},
		{
			name:       "many metrics",
			ctxTimeout: 1 * time.Second,
			setupData:  []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9", "m10"},
			wantIDs:    []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9", "m10"},
			wantErr:    false,
		},
		{
			name:       "short",
			ctxTimeout: 1 * time.Nanosecond,
			setupData:  bigTable,
			wantIDs:    bigTable,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := storage.pool.Exec(ctx, `TRUNCATE TABLE metrics`)
			if err != nil {
				t.Fatalf("failed to truncate: %v", err)
			}

			for _, id := range tt.setupData {
				_, err := storage.pool.Exec(ctx, `
					INSERT INTO metrics (id, type) VALUES ($1, $2)
				`, id, models.Counter)
				if err != nil {
					t.Fatalf("failed to insert: %v", err)
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			result, err := storage.GetNamesList(ctx)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil. result: %v", result)
				}
				t.Logf("got expected error: %v", err)
				cancel()
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assert.ElementsMatch(t, result, tt.wantIDs)

			cancel()
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestPgStorage_Get_TableDriven(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	_, dsn := setupPostgresContainer(t)
	ctx := context.Background()

	logger, err := logger.NewZap("info")
	if err != nil {
		t.Fatalf("failed to create pg logger: %v", err)
	}

	storage, err := NewPgStorage(ctx, dsn, logger)
	if err != nil {
		t.Fatalf("failed to create pg storage: %v", err)
	}
	defer storage.Close()

	bigTable := make([]models.Metrics, 0)
	for i := range 1000 {
		bigTable = append(bigTable, models.Metrics{
			ID:    "v" + strconv.Itoa(i),
			MType: models.Counter,
			Value: nil,
			Delta: ptr(int64(1)),
			Hash:  "",
		})
	}

	tests := []struct {
		name       string
		ctxTimeout time.Duration
		setupData  []models.Metrics
		varType    string
		varName    string
		wantMetric models.Metrics
		wantErr    bool
	}{
		{
			name:       "get counter",
			ctxTimeout: 1 * time.Second,
			setupData: []models.Metrics{
				{
					ID:    "PollCount",
					MType: models.Counter,
					Delta: ptr(int64(1)),
					Value: nil,
					Hash:  "",
				},
			},
			varType: models.Counter,
			varName: "PollCount",
			wantMetric: models.Metrics{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: ptr(int64(1)),
				Value: nil,
				Hash:  "",
			},
			wantErr: false,
		},
		{
			name:       "get Gauge",
			ctxTimeout: 1 * time.Second,
			setupData: []models.Metrics{
				{
					ID:    "GaugeMetric",
					MType: models.Gauge,
					Delta: nil,
					Value: ptr(float64(0.1)),
					Hash:  "",
				},
			},
			varType: models.Gauge,
			varName: "GaugeMetric",
			wantMetric: models.Metrics{
				ID:    "GaugeMetric",
				MType: models.Gauge,
				Delta: nil,
				Value: ptr(float64(0.1)),
				Hash:  "",
			},
			wantErr: false,
		},
		{
			name:       "not found name",
			ctxTimeout: 1 * time.Second,
			setupData: []models.Metrics{
				{
					ID:    "GaugeMetric",
					MType: models.Gauge,
					Delta: nil,
					Value: ptr(float64(0.1)),
					Hash:  "",
				},
			},
			varType: models.Gauge,
			varName: "PollCount",
			wantMetric: models.Metrics{
				ID:    "",
				MType: "",
				Delta: nil,
				Value: nil,
				Hash:  "",
			},
			wantErr: false,
		},
		{
			name:       "not found type",
			ctxTimeout: 1 * time.Second,
			setupData: []models.Metrics{
				{
					ID:    "GaugeMetric",
					MType: models.Gauge,
					Delta: nil,
					Value: ptr(float64(0.1)),
					Hash:  "",
				},
			},
			varType: models.Counter,
			varName: "GaugeMetric",
			wantMetric: models.Metrics{
				ID:    "",
				MType: "",
				Delta: nil,
				Value: nil,
				Hash:  "",
			},
			wantErr: false,
		},
		{
			name:       "error",
			ctxTimeout: 1 * time.Nanosecond,
			setupData:  bigTable,
			varType:    models.Counter,
			varName:    "PollCount",
			wantMetric: models.Metrics{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: ptr(int64(1)),
				Value: nil,
				Hash:  "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := storage.pool.Exec(ctx, `TRUNCATE TABLE metrics`)
			if err != nil {
				t.Fatalf("failed to truncate: %v", err)
			}

			for _, m := range tt.setupData {
				_, err := storage.pool.Exec(ctx, `
					INSERT INTO metrics (id, type, delta, value, hash) VALUES ($1, $2, $3, $4, $5)
				`, m.ID, m.MType, m.Delta, m.Value, m.Hash)
				if err != nil {
					t.Fatalf("failed to insert: %v", err)
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			result, err := storage.Get(ctx, tt.varType, tt.varName)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil. result: %v", result)
				}
				t.Logf("got expected error: %v", err)
				cancel()
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assert.Equal(t, result, tt.wantMetric)

			cancel()
		})
	}
}

type noopLogger struct{}

func (n noopLogger) Info(msg string, args ...any)  {}
func (n noopLogger) Error(msg string, args ...any) {}
func (n noopLogger) Debug(msg string, args ...any) {}
func (n noopLogger) Fatal(msg string, args ...any) {}

func BenchmarkPgStorageGet(b *testing.B) {
	_, dsn := setupPostgresContainer(b)

	ctx := context.Background()
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		b.Fatalf("failed to create pool via db.NewPool: %v", err)
	}
	b.Cleanup(pool.Close)

	store, err := NewPgStorage(ctx, dsn, noopLogger{})
	if err != nil {
		b.Fatalf("failed to create pool via db.NewPool: %v", err)
	}

	if _, err := pool.Exec(ctx, `TRUNCATE TABLE metrics`); err != nil {
		b.Fatalf("failed to truncate: %v", err)
	}

	testMetric := models.Metrics{
		ID:    "bench_get_counter",
		MType: models.Counter,
		Delta: new(int64(42)),
		Value: nil,
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO metrics (id, type, delta, value) VALUES ($1, $2, $3, $4)`,
		testMetric.ID, testMetric.MType, testMetric.Delta, testMetric.Value,
	); err != nil {
		b.Fatalf("failed to insert test data: %v", err)
	}

	b.Run("ExistingMetric", func(b *testing.B) {
		b.ReportAllocs()

		var sink models.Metrics

		for b.Loop() {
			m, err := store.Get(ctx, models.Counter, "bench_get_counter")
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			sink = m
		}
		_ = sink
	})

	b.Run("NonExistingMetric", func(b *testing.B) {
		b.ReportAllocs()

		var counter atomic.Int64

		for b.Loop() {
			name := "non_existent_1metric_" + strconv.FormatInt(counter.Add(1), 10)

			_, err := store.Get(ctx, models.Counter, name)
			if err == nil {
				b.Fatal("expected error, got nil")
			}
		}
	})
}

func BenchmarkPgStorageSave(b *testing.B) {
	_, dsn := setupPostgresContainer(b)
	ctx := context.Background()

	store, err := NewPgStorage(ctx, dsn, noopLogger{})
	if err != nil {
		b.Fatalf("failed to create storage: %v", err)
	}
	b.Cleanup(func() {
		if err := store.Close(); err != nil {
			b.Logf("failed to close storage: %v", err)
		}
	})

	// Очистка таблицы перед каждым запуском бенча
	if _, err := store.pool.Exec(ctx, `TRUNCATE TABLE metrics`); err != nil {
		b.Fatalf("failed to truncate: %v", err)
	}

	// Сценарий 1: INSERT новой метрики Counter (delta)
	b.Run("InsertNewCounter", func(b *testing.B) {
		b.ReportAllocs()
		var counter atomic.Int64

		for b.Loop() {
			id := "new_counter_" + strconv.FormatInt(counter.Add(1), 10)
			m := models.Metrics{
				ID:    id,
				MType: models.Counter,
				Delta: ptr(int64(42)),
				Value: nil,
			}
			if err := store.Save(ctx, m); err != nil {
				b.Fatalf("save failed: %v", err)
			}
		}
	})

	// Сценарий 2: INSERT новой метрики Gauge (value)
	b.Run("InsertNewGauge", func(b *testing.B) {
		b.ReportAllocs()
		var counter atomic.Int64

		for b.Loop() {
			id := "new_gauge_" + strconv.FormatInt(counter.Add(1), 10)
			m := models.Metrics{
				ID:    id,
				MType: models.Gauge,
				Delta: nil,
				Value: ptr(float64(3.14)),
			}
			if err := store.Save(ctx, m); err != nil {
				b.Fatalf("save failed: %v", err)
			}
		}
	})

	// Сценарий 3: UPDATE существующей метрики Counter (ON CONFLICT)
	b.Run("UpdateExistingCounter", func(b *testing.B) {
		b.ReportAllocs()

		// Предварительно создаем метрику, которую будем обновлять
		existingID := "existing_counter"
		if err := store.Save(ctx, models.Metrics{
			ID:    existingID,
			MType: models.Counter,
			Delta: ptr(int64(100)),
		}); err != nil {
			b.Fatalf("failed to create existing metric: %v", err)
		}

		for b.Loop() {
			m := models.Metrics{
				ID:    existingID,
				MType: models.Counter,
				Delta: ptr(int64(42)), // Новое значение delta
			}
			if err := store.Save(ctx, m); err != nil {
				b.Fatalf("save failed: %v", err)
			}
		}
	})

	// Сценарий 4: UPDATE существующей метрики Gauge (ON CONFLICT)
	b.Run("UpdateExistingGauge", func(b *testing.B) {
		b.ReportAllocs()

		// Предварительно создаем метрику
		existingID := "existing_gauge"
		if err := store.Save(ctx, models.Metrics{
			ID:    existingID,
			MType: models.Gauge,
			Value: ptr(float64(100.0)),
		}); err != nil {
			b.Fatalf("failed to create existing metric: %v", err)
		}

		for b.Loop() {
			m := models.Metrics{
				ID:    existingID,
				MType: models.Gauge,
				Value: ptr(float64(42.5)), // Новое значение value
			}
			if err := store.Save(ctx, m); err != nil {
				b.Fatalf("save failed: %v", err)
			}
		}
	})
}

func BenchmarkPgStorageBatchSave(b *testing.B) {
	_, dsn := setupPostgresContainer(b)
	ctx := context.Background()

	store, err := NewPgStorage(ctx, dsn, noopLogger{})
	if err != nil {
		b.Fatalf("failed to create storage: %v", err)
	}
	b.Cleanup(func() {
		if err := store.Close(); err != nil {
			b.Logf("failed to close storage: %v", err)
		}
	})

	if _, err := store.pool.Exec(ctx, `TRUNCATE TABLE metrics`); err != nil {
		b.Fatalf("failed to truncate: %v", err)
	}

	b.Run("BatchSize10_Insert", func(b *testing.B) {
		b.ReportAllocs()
		var counter atomic.Int64

		for b.Loop() {
			start := counter.Add(10)
			batch := make([]models.Metrics, 10)
			for i := 0; i < 10; i++ {
				idx := start - 10 + int64(i)
				if i%2 == 0 {
					batch[i] = models.Metrics{
						ID:    "b10_c_" + strconv.FormatInt(idx, 10),
						MType: models.Counter,
						Delta: ptr(int64(idx)),
					}
				} else {
					batch[i] = models.Metrics{
						ID:    "b10_g_" + strconv.FormatInt(idx, 10),
						MType: models.Gauge,
						Value: ptr(float64(idx)),
					}
				}
			}

			if err := store.BatchSave(ctx, batch); err != nil {
				b.Fatalf("batch save failed: %v", err)
			}
		}
	})

	b.Run("BatchSize100_Insert", func(b *testing.B) {
		b.ReportAllocs()
		var counter atomic.Int64

		for b.Loop() {
			start := counter.Add(100)
			batch := make([]models.Metrics, 100)
			for i := 0; i < 100; i++ {
				idx := start - 100 + int64(i)
				if i%2 == 0 {
					batch[i] = models.Metrics{
						ID:    "b100_c_" + strconv.FormatInt(idx, 10),
						MType: models.Counter,
						Delta: ptr(int64(idx)),
					}
				} else {
					batch[i] = models.Metrics{
						ID:    "b100_g_" + strconv.FormatInt(idx, 10),
						MType: models.Gauge,
						Value: ptr(float64(idx)),
					}
				}
			}

			if err := store.BatchSave(ctx, batch); err != nil {
				b.Fatalf("batch save failed: %v", err)
			}
		}
	})

	b.Run("BatchSize10_Update", func(b *testing.B) {
		b.ReportAllocs()

		baseBatch := make([]models.Metrics, 10)
		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				baseBatch[i] = models.Metrics{ID: "update_c_" + strconv.Itoa(i), MType: models.Counter, Delta: ptr(int64(100))}
			} else {
				baseBatch[i] = models.Metrics{ID: "update_g_" + strconv.Itoa(i), MType: models.Gauge, Value: ptr(float64(100.0))}
			}
		}

		if err := store.BatchSave(ctx, baseBatch); err != nil {
			b.Fatalf("failed to seed data: %v", err)
		}

		var counter atomic.Int64
		for b.Loop() {
			batch := make([]models.Metrics, 10)
			for i := range 10 {
				idx := counter.Add(1)
				if i%2 == 0 {
					batch[i] = models.Metrics{ID: "update_c_" + strconv.Itoa(i), MType: models.Counter, Delta: ptr(int64(idx))}
				} else {
					batch[i] = models.Metrics{ID: "update_g_" + strconv.Itoa(i), MType: models.Gauge, Value: ptr(float64(float64(idx)))}
				}
			}

			if err := store.BatchSave(ctx, batch); err != nil {
				b.Fatalf("batch save failed: %v", err)
			}
		}
	})
}
