package sqlclient

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Config struct {
	DSN string
}

type Client struct {
	db *sql.DB
}

func New(cfg Config) (*Client, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sqlclient: open: %w", err)
	}
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

func (c *Client) Exec(query string) error {
	_, err := c.db.Exec(query)
	return err
}

func (c *Client) Close() error {
	return c.db.Close()
}
