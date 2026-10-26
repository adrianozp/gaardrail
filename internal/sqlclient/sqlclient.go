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
	Driver string
	TLS    TLSConfig
}

type TLSConfig struct {
	Enabled    bool
	CAFile     string
	SkipVerify bool
	ServerName string
}

type Client struct {
	db *sql.DB
}

func New(cfg Config) (*Client, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "mysql"
	}
	dsn, err := secureDSN(driver, cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlclient: open: %w", err)
	}
	configurePool(db)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlclient: ping: %w", err)
	}
	return &Client{db: db}, nil
}

func secureDSN(driver string, cfg Config) (string, error) {
	if !cfg.TLS.Enabled {
		return cfg.DSN, nil
	}
	if driver != "mysql" {
		return "", fmt.Errorf("sqlclient: tls is only supported for the mysql driver, got %q", driver)
	}
	tlsName, err := registerMySQLTLS(cfg.TLS)
	if err != nil {
		return "", fmt.Errorf("sqlclient: tls: %w", err)
	}
	return withDSNParam(cfg.DSN, "tls", tlsName), nil
}

func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
}

func registerMySQLTLS(cfg TLSConfig) (string, error) {
	if cfg.SkipVerify {
		return "skip-verify", nil
	}
	if cfg.CAFile == "" && cfg.ServerName == "" {
		return "true", nil
	}
	tlsCfg := &tls.Config{ServerName: cfg.ServerName, MinVersion: tls.VersionTLS12}
	if err := loadRootCAs(tlsCfg, cfg.CAFile); err != nil {
		return "", err
	}
	const name = "gaardrail-custom"
	if err := mysql.RegisterTLSConfig(name, tlsCfg); err != nil {
		return "", err
	}
	return name, nil
}

func loadRootCAs(tlsCfg *tls.Config, caFile string) error {
	if caFile == "" {
		return nil
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("read ca file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return fmt.Errorf("no valid certificate found in ca file %q", caFile)
	}
	tlsCfg.RootCAs = pool
	return nil
}

func withDSNParam(dsn, key, value string) string {
	sep := "?"
	if strings.ContainsRune(dsn, '?') {
		sep = "&"
	}
	return dsn + sep + key + "=" + value
}

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
