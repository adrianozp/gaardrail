package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type Config struct {
	BaseURL string
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(cfg Config) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		http:    &http.Client{},
	}
}

// FIXME: both functions dont return body and other info
func (c *Client) Post(ctx context.Context, path string, body []byte) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return nil
}

func (c *Client) Get(path string, body string) error {
	url := c.baseURL + path
	resp, err := c.http.Get(path)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return nil
}
