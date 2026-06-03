package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeSource is a TokenSource whose Token/Refresh behavior is set per-test.
type fakeSource struct {
	token    string
	refresh  func(ctx context.Context) (string, error)
	refreshN int
}

func (f *fakeSource) Token(context.Context) (string, error) { return f.token, nil }
func (f *fakeSource) Refresh(ctx context.Context) (string, error) {
	f.refreshN++
	return f.refresh(ctx)
}

func TestStaticHappyPath(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	c := WithStaticToken(srv.URL, "static-tok")
	resp, raw, err := c.Get(context.Background(), "ledger", "/things")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(raw) != "ok" {
		t.Fatalf("body = %q, want ok", raw)
	}
	if gotAuth != "Bearer static-tok" {
		t.Fatalf("auth = %q, want Bearer static-tok", gotAuth)
	}
}

func TestStatic401ReturnsBareResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`nope`))
	}))
	defer srv.Close()

	c := WithStaticToken(srv.URL, "static-tok")
	resp, raw, err := c.Get(context.Background(), "ledger", "/things")
	if errors.Is(err, ErrReauth) {
		t.Fatal("static-token 401 must not surface ErrReauth")
	}
	if err != nil {
		t.Fatalf("Do returned unexpected err %v, want bare 401 response", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if string(raw) != "nope" {
		t.Fatalf("body = %q, want nope", raw)
	}
}

func TestRefreshingSourceRetriesOn401(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		seen = append(seen, auth)
		if auth == "Bearer fresh" {
			_, _ = w.Write([]byte(`ok`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	src := &fakeSource{token: "stale", refresh: func(context.Context) (string, error) { return "fresh", nil }}
	c := New(srv.URL, src)
	resp, raw, err := c.Get(context.Background(), "ledger", "/things")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.StatusCode != http.StatusOK || string(raw) != "ok" {
		t.Fatalf("status=%d body=%q, want 200 ok", resp.StatusCode, raw)
	}
	if src.refreshN != 1 {
		t.Fatalf("refresh called %d times, want 1", src.refreshN)
	}
	if len(seen) != 2 || seen[0] != "Bearer stale" || seen[1] != "Bearer fresh" {
		t.Fatalf("requests = %v, want [Bearer stale, Bearer fresh]", seen)
	}
}

func TestRefreshErrReauthPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	src := &fakeSource{token: "stale", refresh: func(context.Context) (string, error) { return "", ErrReauth }}
	c := New(srv.URL, src)
	_, _, err := c.Get(context.Background(), "ledger", "/things")
	if !errors.Is(err, ErrReauth) {
		t.Fatalf("err = %v, want ErrReauth", err)
	}
}
