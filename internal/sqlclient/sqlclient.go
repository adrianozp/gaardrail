package sqlclient

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type Config struct {
	DSN    string
	Driver string // "mysql" or "postgres"; defaults to "mysql" when empty
}

type Client struct {
	db *sql.DB
}

func New(cfg Config) (*Client, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "mysql"
	}
	db, err := sql.Open(driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sqlclient: open: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlclient: ping: %w", err)
	}
	return &Client{db: db}, nil
}

// NewFromDB creates a Client from an existing *sql.DB. Useful for testing.
func NewFromDB(db *sql.DB) *Client {
	return &Client{db: db}
}

func (c *Client) ExecContext(ctx context.Context, query string) (int64, error) {
	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}
