package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/axiom-idp/axiom/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type runtimeStateStore interface {
	AppendAudit(ctx context.Context, entry AuditLog) error
	QueryAudit(ctx context.Context, userID string, limit int) ([]AuditLog, error)
	AuditStats(ctx context.Context) (AuditStats, error)
	AllowRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	RateLimitStats(ctx context.Context) (RateLimiterStats, error)
	Close() error
	Backend() string
	Shared() bool
}

type sqlRuntimeStateStore struct {
	db      *sql.DB
	backend string
	driver  string
	shared  bool
}

func newRuntimeStateStore(cfg *config.Config) (runtimeStateStore, error) {
	if cfg == nil {
		return nil, nil
	}

	driver := cfg.NormalizedDBDriver()
	switch driver {
	case "", "memory":
		return nil, nil
	case "sqlite":
		return openSQLRuntimeStateStore("sqlite", "sqlite", cfg.DBURL, false)
	case "postgres":
		return openSQLRuntimeStateStore("postgres", "pgx", cfg.DBURL, true)
	default:
		return nil, nil
	}
}

func openSQLRuntimeStateStore(backend, driver, dsn string, shared bool) (runtimeStateStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("%s runtime state store requires a database URL", driver)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	store := &sqlRuntimeStateStore{
		db:      db,
		backend: backend,
		driver:  driver,
		shared:  shared,
	}

	if driver == "sqlite" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)
	} else {
		db.SetMaxOpenConns(8)
		db.SetMaxIdleConns(8)
		db.SetConnMaxLifetime(30 * time.Minute)
		db.SetConnMaxIdleTime(5 * time.Minute)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := store.initSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *sqlRuntimeStateStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *sqlRuntimeStateStore) Backend() string {
	if s == nil {
		return ""
	}
	return s.backend
}

func (s *sqlRuntimeStateStore) Shared() bool {
	if s == nil {
		return false
	}
	return s.shared
}

func (s *sqlRuntimeStateStore) initSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS axiom_audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			status TEXT NOT NULL,
			details_json TEXT NOT NULL,
			error_text TEXT NOT NULL DEFAULT '',
			created_at_ms BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_axiom_audit_log_user_created ON axiom_audit_log(user_id, created_at_ms)`,
		`CREATE INDEX IF NOT EXISTS idx_axiom_audit_log_created ON axiom_audit_log(created_at_ms)`,
		`CREATE TABLE IF NOT EXISTS axiom_rate_limit_window (
			key TEXT PRIMARY KEY,
			window_start_ms BIGINT NOT NULL,
			request_count INTEGER NOT NULL,
			updated_at_ms BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_axiom_rate_limit_window_updated ON axiom_rate_limit_window(updated_at_ms)`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return nil
}

func (s *sqlRuntimeStateStore) AppendAudit(ctx context.Context, entry AuditLog) error {
	if s == nil || s.db == nil {
		return nil
	}

	detailsJSON, err := json.Marshal(entry.Details)
	if err != nil {
		return err
	}

	_, err = s.execContext(ctx, `
		INSERT INTO axiom_audit_log (
			id, user_id, action, resource, status, details_json, error_text, created_at_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.UserID, entry.Action, entry.Resource, entry.Status, string(detailsJSON), entry.Error, entry.CreatedAt.UTC().UnixMilli())
	return err
}

func (s *sqlRuntimeStateStore) QueryAudit(ctx context.Context, userID string, limit int) ([]AuditLog, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, user_id, action, resource, status, details_json, error_text, created_at_ms
		FROM axiom_audit_log
		WHERE (? = '' OR user_id = ?)
		ORDER BY created_at_ms DESC
		LIMIT ?
	`

	rows, err := s.queryContext(ctx, query, userID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]AuditLog, 0, limit)
	for rows.Next() {
		var (
			entry       AuditLog
			detailsJSON string
			createdAtMs int64
		)
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.Resource, &entry.Status, &detailsJSON, &entry.Error, &createdAtMs); err != nil {
			return nil, err
		}
		entry.CreatedAt = time.UnixMilli(createdAtMs).UTC()
		if err := json.Unmarshal([]byte(detailsJSON), &entry.Details); err != nil {
			entry.Details = map[string]interface{}{}
		}
		logs = append(logs, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	reverseAuditLogs(logs)
	return logs, nil
}

func (s *sqlRuntimeStateStore) AuditStats(ctx context.Context) (AuditStats, error) {
	if s == nil || s.db == nil {
		return AuditStats{}, nil
	}

	var (
		total       sql.NullInt64
		lastAt      sql.NullInt64
		errorsSeen  sql.NullInt64
		deniedSeen  sql.NullInt64
		successSeen sql.NullInt64
	)

	row := s.queryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(MAX(created_at_ms), 0),
			COALESCE(SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'denied' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status NOT IN ('error', 'denied') THEN 1 ELSE 0 END), 0)
		FROM axiom_audit_log
	`)
	if err := row.Scan(&total, &lastAt, &errorsSeen, &deniedSeen, &successSeen); err != nil {
		return AuditStats{}, err
	}

	stats := AuditStats{
		Entries:      int(total.Int64),
		ErrorCount:   int(errorsSeen.Int64),
		DeniedCount:  int(deniedSeen.Int64),
		SuccessCount: int(successSeen.Int64),
	}
	if lastAt.Valid && lastAt.Int64 > 0 {
		stats.LastEntryAt = time.UnixMilli(lastAt.Int64).UTC()
	}
	return stats, nil
}

func (s *sqlRuntimeStateStore) AllowRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if s == nil || s.db == nil {
		return true, nil
	}
	if limit <= 0 || window <= 0 {
		return true, nil
	}

	now := time.Now().UTC()
	windowStart := now.Truncate(window).UnixMilli()
	cutoff := now.Add(-window * 2).UnixMilli()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	if _, err := s.txExecContext(ctx, tx, `DELETE FROM axiom_rate_limit_window WHERE updated_at_ms < ?`, cutoff); err != nil {
		return false, err
	}

	row := s.txQueryRowContext(ctx, tx, `
		INSERT INTO axiom_rate_limit_window (key, window_start_ms, request_count, updated_at_ms)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(key) DO UPDATE SET
			window_start_ms = excluded.window_start_ms,
			request_count = CASE
				WHEN axiom_rate_limit_window.window_start_ms = excluded.window_start_ms THEN axiom_rate_limit_window.request_count + 1
				ELSE 1
			END,
			updated_at_ms = excluded.updated_at_ms
		RETURNING request_count
	`, key, windowStart, now.UnixMilli())

	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}

	return count <= limit, nil
}

func (s *sqlRuntimeStateStore) RateLimitStats(ctx context.Context) (RateLimiterStats, error) {
	if s == nil || s.db == nil {
		return RateLimiterStats{}, nil
	}

	var (
		trackedKeys sql.NullInt64
		requests    sql.NullInt64
		lastUpdated sql.NullInt64
	)

	row := s.queryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(request_count), 0),
			COALESCE(MAX(updated_at_ms), 0)
		FROM axiom_rate_limit_window
	`)
	if err := row.Scan(&trackedKeys, &requests, &lastUpdated); err != nil {
		return RateLimiterStats{}, err
	}

	stats := RateLimiterStats{
		TrackedKeys:   int(trackedKeys.Int64),
		RequestsSeen:  int(requests.Int64),
		LastCleanupAt: time.Time{},
	}
	if lastUpdated.Valid && lastUpdated.Int64 > 0 {
		stats.LastCleanupAt = time.UnixMilli(lastUpdated.Int64).UTC()
	}
	return stats, nil
}

func reverseAuditLogs(logs []AuditLog) {
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}
}

func (s *sqlRuntimeStateStore) execContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	query = s.rebind(query)
	return s.db.ExecContext(ctx, query, args...)
}

func (s *sqlRuntimeStateStore) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	query = s.rebind(query)
	return s.db.QueryContext(ctx, query, args...)
}

func (s *sqlRuntimeStateStore) queryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	query = s.rebind(query)
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *sqlRuntimeStateStore) txExecContext(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	query = s.rebind(query)
	return tx.ExecContext(ctx, query, args...)
}

func (s *sqlRuntimeStateStore) txQueryRowContext(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) *sql.Row {
	query = s.rebind(query)
	return tx.QueryRowContext(ctx, query, args...)
}

func (s *sqlRuntimeStateStore) rebind(query string) string {
	if s == nil || s.driver != "pgx" {
		return query
	}

	var out strings.Builder
	index := 1
	for _, r := range query {
		if r == '?' {
			out.WriteByte('$')
			out.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
