package sqlclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type Config struct {
	DSN    string
	Driver string // "mysql" or "postgres"; defaults to "mysql" when empty
	TLS    TLSConfig
}

// TLSConfig enables encrypted communication with the database (mysql only).
type TLSConfig struct {
	Enabled    bool
	CAFile     string // optional PEM CA bundle for server verification
	SkipVerify bool   // encrypt but do not verify the server certificate (testing)
	ServerName string // optional, used for certificate verification
}

type Client struct {
	db *sql.DB
}

func New(cfg Config) (*Client, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "mysql"
	}

	dsn := cfg.DSN
	if cfg.TLS.Enabled {
		if driver != "mysql" {
			return nil, fmt.Errorf("sqlclient: tls is only supported for the mysql driver, got %q", driver)
		}
		tlsName, err := registerMySQLTLS(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("sqlclient: tls: %w", err)
		}
		dsn = withDSNParam(dsn, "tls", tlsName)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlclient: open: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlclient: ping: %w", err)
	}
	return &Client{db: db}, nil
}

// registerMySQLTLS resolves the value to use for the DSN "tls" parameter.
// For the simple cases it returns a built-in name; when a CA bundle or a
// server name is given it registers a custom *tls.Config and returns its name.
func registerMySQLTLS(cfg TLSConfig) (string, error) {
	if cfg.SkipVerify {
		return "skip-verify", nil
	}
	if cfg.CAFile == "" && cfg.ServerName == "" {
		return "true", nil
	}

	tlsCfg := &tls.Config{
		ServerName: cfg.ServerName,
		MinVersion: tls.VersionTLS12,
	}
	if cfg.CAFile != "" {
		pem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return "", fmt.Errorf("read ca file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return "", fmt.Errorf("no valid certificate found in ca file %q", cfg.CAFile)
		}
		tlsCfg.RootCAs = pool
	}

	const name = "gaardrail-custom"
	if err := mysql.RegisterTLSConfig(name, tlsCfg); err != nil {
		return "", err
	}
	return name, nil
}

// withDSNParam appends key=value to a DSN query string, choosing "?" or "&".
func withDSNParam(dsn, key, value string) string {
	sep := "?"
	if strings.ContainsRune(dsn, '?') {
		sep = "&"
	}
	return dsn + sep + key + "=" + value
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
