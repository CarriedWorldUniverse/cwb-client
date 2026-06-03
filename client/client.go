// Package client is the edge-anchored HTTP seam. It presents a bearer from a
// TokenSource to product routes under the edge (<edge>/<pillar>/<path>), and on
// a 401 asks the source to refresh and retries once.
package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrReauth means re-auth is required (the refresh path is exhausted).
var ErrReauth = errors.New("session expired: run 'cw auth login'")

// ErrNoRefresh is returned by a TokenSource that cannot refresh (e.g. a static
// token); on a 401 the client surfaces the original response instead of retrying.
var ErrNoRefresh = errors.New("token source cannot refresh")

// TokenSource supplies (and refreshes) the bearer the client presents.
type TokenSource interface {
	Token(ctx context.Context) (string, error)   // current token, refreshing if stale
	Refresh(ctx context.Context) (string, error) // force-refresh after a 401; ErrReauth/ErrNoRefresh
}

// Doer executes an authenticated CWB request and returns the raw response.
// *Client is the HTTP implementation; other transports (e.g. a WS relay in
// nexus) implement it so the pillar wrappers are transport-agnostic.
type Doer interface {
	Do(ctx context.Context, method, pillar, path string, body []byte) (*http.Response, []byte, error)
}

// Client targets one edge as one identity (its TokenSource).
type Client struct {
	edge string
	src  TokenSource
	hc   *http.Client
}

// New builds a Client from an edge + a TokenSource.
func New(edge string, src TokenSource) *Client {
	return &Client{edge: strings.TrimRight(edge, "/"), src: src, hc: &http.Client{Timeout: 30 * time.Second}}
}

// staticSource always returns a fixed token and cannot refresh.
type staticSource struct{ token string }

func (s staticSource) Token(context.Context) (string, error)   { return s.token, nil }
func (s staticSource) Refresh(context.Context) (string, error) { return "", ErrNoRefresh }

// WithStaticToken returns a Client that always uses the given bearer (stateless
// per-invocation use, e.g. --token / ToolRunner agents).
func WithStaticToken(edge, token string) *Client { return New(edge, staticSource{token}) }

// AccessToken returns a currently-valid access token (refreshing if stale).
func (c *Client) AccessToken(ctx context.Context) (string, error) { return c.src.Token(ctx) }

// URL builds a product URL: <edge>/<pillar><path> (path begins with "/").
func (c *Client) URL(pillar, path string) string {
	return c.edge + "/" + strings.Trim(pillar, "/") + path
}

// Do executes an authenticated request; on a 401 it asks the source to refresh
// and retries once (ErrNoRefresh → surface the original response). body is []byte
// so the retry can resend it.
func (c *Client) Do(ctx context.Context, method, pillar, path string, body []byte) (*http.Response, []byte, error) {
	tok, err := c.src.Token(ctx)
	if err != nil {
		return nil, nil, err
	}
	resp, raw, err := c.do(ctx, method, c.URL(pillar, path), tok, body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		fresh, rerr := c.src.Refresh(ctx)
		if errors.Is(rerr, ErrNoRefresh) {
			return resp, raw, nil
		}
		if rerr != nil {
			return nil, nil, rerr
		}
		return c.do(ctx, method, c.URL(pillar, path), fresh, body)
	}
	return resp, raw, nil
}

// Get is a convenience wrapper.
func (c *Client) Get(ctx context.Context, pillar, path string) (*http.Response, []byte, error) {
	return c.Do(ctx, http.MethodGet, pillar, path, nil)
}

func (c *Client) do(ctx context.Context, method, url, bearer string, body []byte) (*http.Response, []byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err == nil && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("client: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("client: %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("client: %s %s: read body: %w", method, url, err)
	}
	return resp, raw, nil
}
