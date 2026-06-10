# cwb-client

Reusable Go client library for the **CarriedWorld Backbone (CWB)** platform — typed wrappers over the herald, cairn, ledger, and commonplace pillars, plus the auth machinery (OIDC token grants, agent JWT assertions, key/fingerprint identity).

Extracted from the `cw` CLI so the same client code can be reused anywhere — the CLI, nexus, aspects — against any CWB edge. It is **transport-agnostic**: the pillar wrappers talk to a `Doer` interface, so the default edge-anchored HTTP client and an alternative transport (e.g. a WS relay inside nexus) are interchangeable.

```
module github.com/CarriedWorldUniverse/cwb-client
```

## Layout

| Package | What it wraps |
|---|---|
| `client` | The edge-anchored HTTP `Doer` + `TokenSource` (bearer presentation, 401-refresh-retry). |
| `oidc` | Herald's OIDC token endpoint — discovery + password / JWT-bearer / refresh grants, revoke. |
| `identity` | Agent JWT assertions (from a seed via `casket.DeriveAgentKey`, or from an ed25519 key), access-claim decoding, key fingerprints. |
| `herald` | Orgs, humans, agents, products, `Me` (whoami), agent-by-fingerprint lookup. |
| `cairn` | Repos and pull requests (create/list repos, open/list/get/merge pulls). |
| `ledger` | Issues — create, get, claim, transition, comment, list-mine / list-ready / search-by-project. |
| `commonplace` | Knowledge entries — store, search, list, update, delete. |

## Auth model

A `Client` targets one **edge** as one **identity**, defined by its `TokenSource`:

```go
type TokenSource interface {
    Token(ctx context.Context) (string, error)   // current token, refreshing if stale
    Refresh(ctx context.Context) (string, error) // force-refresh after a 401
}
```

The client presents that bearer to product routes under the edge (`<edge>/<pillar>/<path>`). On a `401` it asks the source to `Refresh` and retries once; a source that can't refresh (e.g. a static token) returns `ErrNoRefresh` and the original response is surfaced. When the refresh path is exhausted it returns `ErrReauth` (run `cw auth login`).

Tokens themselves come from `oidc`: a **human** uses the password grant; an **agent** mints a JWT assertion (`identity.AgentAssertion` from an owner seed + slug, or `AgentAssertionFromKey` from an ed25519 key) and exchanges it via the JWT-bearer grant. Refresh-token grants keep a session alive without re-auth.

## Usage

```go
import (
    "github.com/CarriedWorldUniverse/cwb-client/client"
    "github.com/CarriedWorldUniverse/cwb-client/ledger"
)

// Static-token client (stateless per-invocation, e.g. an agent with a short-lived bearer):
c := client.WithStaticToken("https://edge.example", token)

// Or a refreshing client backed by your own TokenSource:
c = client.New("https://edge.example", mySource)

// Pillar wrappers take any client.Doer:
issue, err := ledger.GetIssue(ctx, c, "NEX-123")
```

Each pillar package is a set of free functions taking a `client.Doer` as the first argument, so any transport implementing `Doer` works unchanged:

```go
type Doer interface {
    Do(ctx context.Context, method, pillar, path string, body []byte) (*http.Response, []byte, error)
}
```

## Status

In use by the `cw` CLI and nexus. Requires `casket-go` (agent-key derivation) and `go-jose/v4` (assertion signing).
