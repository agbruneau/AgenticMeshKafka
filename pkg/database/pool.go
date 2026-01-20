package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds the configuration for the database pool
type PoolConfig struct {
	Host         string
	Port         int
	Database     string
	User         string
	Password     string
	MaxConns     int32
	MinConns     int32
	MaxConnIdleTime string
}

// Validate checks if the pool configuration is valid
func (c *PoolConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.Database == "" {
		return fmt.Errorf("database is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	return nil
}

// ConnectionString returns the PostgreSQL connection string
func (c *PoolConfig) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

// Pool defines the interface for database operations
type Pool interface {
	HealthCheck(ctx context.Context) error
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
	Close()
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (int64, error)
}

// DBPool wraps pgxpool.Pool and implements the Pool interface
type DBPool struct {
	config PoolConfig
	pool   *pgxpool.Pool
}

// NewDBPool creates a new database pool
func NewDBPool(cfg PoolConfig) (*DBPool, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolConfig.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolConfig.MinConns = cfg.MinConns
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	return &DBPool{
		config: cfg,
		pool:   pool,
	}, nil
}

// HealthCheck verifies the database connection
func (p *DBPool) HealthCheck(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// WithTransaction executes a function within a database transaction
func (p *DBPool) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the database pool
func (p *DBPool) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Query executes a query that returns rows
func (p *DBPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row
func (p *DBPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a query without returning rows
func (p *DBPool) Exec(ctx context.Context, sql string, args ...interface{}) (int64, error) {
	result, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetPool returns the underlying pgxpool.Pool
func (p *DBPool) GetPool() *pgxpool.Pool {
	return p.pool
}
