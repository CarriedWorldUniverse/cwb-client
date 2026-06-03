package client

import (
	"context"
	"net/http"
	"testing"
)

// fakeDoer satisfies Doer without being a *Client — proves the seam is an
// interface, so pillar wrappers can run over a non-HTTP transport.
type fakeDoer struct {
	gotMethod, gotPillar, gotPath string
	status                        int
	body                          []byte
}

func (f *fakeDoer) Do(_ context.Context, method, pillar, path string, _ []byte) (*http.Response, []byte, error) {
	f.gotMethod, f.gotPillar, f.gotPath = method, pillar, path
	return &http.Response{StatusCode: f.status}, f.body, nil
}

func TestClientSatisfiesDoer(t *testing.T) {
	var _ Doer = (*Client)(nil)   // *Client implements Doer
	var _ Doer = (*fakeDoer)(nil) // and so can a fake
}
