package identity

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	casket "github.com/CarriedWorldUniverse/casket-go"
	jose "github.com/go-jose/go-jose/v4"
)

func TestAgentAssertionVerifies(t *testing.T) {
	seed := []byte("owner-seed-32-bytes-padded-xxxxx")
	_, pub, err := casket.DeriveAgentKey(seed, "shadow")
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	assertion, err := AgentAssertion(seed, "shadow", "agent-123", "http://edge:8080/herald/token")
	if err != nil {
		t.Fatalf("AgentAssertion: %v", err)
	}
	// Verify the assertion against the derived public key (what herald does).
	jws, err := jose.ParseSigned(assertion, []jose.SignatureAlgorithm{jose.EdDSA})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	payload, err := jws.Verify(pub)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	claims := DecodeClaimsBytes(payload)
	if claims["iss"] != "agent-123" || claims["sub"] != "agent-123" || claims["aud"] != "http://edge:8080/herald/token" {
		t.Fatalf("claims: %+v", claims)
	}
}

func TestDecodeAccessClaims(t *testing.T) {
	// header.payload.sig where payload = base64url({"sub":"u1","kind":"human","scope":"a b"})
	tok := "x." + b64url(`{"sub":"u1","kind":"human","scope":"a b","exp":111}`) + ".y"
	claims, err := DecodeAccessClaims(tok)
	if err != nil || claims["sub"] != "u1" || claims["kind"] != "human" {
		t.Fatalf("DecodeAccessClaims: %v %+v", err, claims)
	}

	// Non-3-part token -> error.
	if _, err := DecodeAccessClaims("not.a"); err == nil {
		t.Fatal("non-3-part token should error")
	}
	// Valid base64 payload that isn't JSON -> non-nil empty claims (safe to read).
	bad := "x." + b64url(`not json`) + ".y"
	c, err := DecodeAccessClaims(bad)
	if err != nil {
		t.Fatalf("bad-json payload: unexpected err %v", err)
	}
	if c == nil {
		t.Fatal("claims must be non-nil even on bad JSON")
	}
	if _, ok := c["sub"].(string); ok {
		t.Fatalf("bad-json claims should be empty, got %+v", c)
	}
}

func TestAgentAssertionFromKeyMatchesSeedPath(t *testing.T) {
	seed := bytes.Repeat([]byte{7}, 32)
	slug, agentID, tokenURL := "plumb", "agent-uuid-1", "https://edge/herald/token"
	now := time.Unix(1_700_000_000, 0)

	fromSeed, err := AgentAssertionAt(seed, slug, agentID, tokenURL, now)
	if err != nil {
		t.Fatalf("AgentAssertionAt: %v", err)
	}
	priv, _, err := casket.DeriveAgentKey(seed, slug)
	if err != nil {
		t.Fatalf("DeriveAgentKey: %v", err)
	}
	fromKey, err := AgentAssertionFromKeyAt(priv, agentID, tokenURL, now)
	if err != nil {
		t.Fatalf("AgentAssertionFromKeyAt: %v", err)
	}
	if fromSeed != fromKey {
		t.Fatalf("assertions differ:\n seed=%s\n key =%s", fromSeed, fromKey)
	}

	claims, err := DecodeAccessClaims(fromKey)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if claims["aud"] != tokenURL || claims["iss"] != agentID || claims["sub"] != agentID {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestAgentAssertionFromKeyValidates(t *testing.T) {
	priv, _, _ := casket.DeriveAgentKey(bytes.Repeat([]byte{1}, 32), "x")
	if _, err := AgentAssertionFromKey(nil, "a", "u"); err == nil {
		t.Error("nil key should error")
	}
	if _, err := AgentAssertionFromKey(priv, "", "u"); err == nil {
		t.Error("empty agentID should error")
	}
	if _, err := AgentAssertionFromKey(priv, "a", ""); err == nil {
		t.Error("empty tokenURL should error")
	}
}

func b64url(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
