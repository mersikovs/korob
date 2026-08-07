//go:build integration

package repository

import (
	"context"
	"strconv"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mersikovs/korob.git/internal/database"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgresContainer(t *testing.T) (*postgres.PostgresContainer, string) {
	t.Helper()

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
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	if err := database.MigrateUp(dsn, "file://../../migrations"); err != nil {
		t.Fatalf("Ошибка миграции базы данных: %v", err)
	}

	return container, dsn
}

func TestNewPgStorage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
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
			storage, err := NewPgStorage(context.Background(), tt.dsn)

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

	storage, err := NewPgStorage(ctx, dsn)
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

func int64Ptr(v int64) *int64 {
	return &v
}

// TODO добить кейсы
func TestPgStorage_Get_TableDriven(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	_, dsn := setupPostgresContainer(t)
	ctx := context.Background()

	storage, err := NewPgStorage(ctx, dsn)
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
			Delta: int64Ptr(1),
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
			name:       "one value",
			ctxTimeout: 1 * time.Second,
			setupData: []models.Metrics{
				{
					ID:    "PollCount",
					MType: models.Counter,
					Delta: int64Ptr(1),
					Value: nil,
					Hash:  "",
				},
			},
			varType: models.Counter,
			varName: "PollCount",
			wantMetric: models.Metrics{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: int64Ptr(1),
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
				Delta: int64Ptr(1),
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
