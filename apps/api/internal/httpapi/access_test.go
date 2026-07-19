package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/openclaw/clickclack/apps/api/internal/realtime"
	"github.com/openclaw/clickclack/apps/api/internal/store"
	sqlitestore "github.com/openclaw/clickclack/apps/api/internal/store/sqlite"
)

func TestAccessAssertionProvisionsSessionAndMembership(t *testing.T) {
	key := newAccessTestKey(t)
	jwks, requests := newAccessTestJWKSServer(t, func() map[string]*rsa.PublicKey {
		return map[string]*rsa.PublicKey{"key-1": &key.PublicKey}
	})
	st := newAccessTestStore(t)
	handler := New(st, realtime.NewHub(), Options{
		DisableDevAuth: true,
		Access: AccessConfig{
			TeamDomain: jwks.URL,
			Audience:   "test-access-aud",
			HTTPClient: jwks.Client(),
		},
	}).Handler()
	token := signAccessTestToken(t, key, "key-1", jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   jwks.URL,
		"aud":   []string{"test-other-aud", "test-access-aud"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Add(-time.Minute).Unix(),
		"email": "Captain@Example.com",
		"name":  "Access Captain",
	})
	blocked := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces", strings.NewReader(`{"name":"Cross Site","slug":"cross-site"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(accessAssertionHeader, token)
	handler.ServeHTTP(blocked, request)
	if blocked.Code != http.StatusForbidden || len(blocked.Result().Cookies()) != 0 {
		t.Fatalf("first Access mutation bypassed CSRF checks: status=%d cookies=%#v body=%s", blocked.Code, blocked.Result().Cookies(), blocked.Body.String())
	}
	if requests.Load() != 0 {
		t.Fatalf("CSRF-rejected Access request fetched JWKS: requests=%d", requests.Load())
	}

	first := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set(accessAssertionHeader, token)
	handler.ServeHTTP(first, request)
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, body = %s", first.Code, first.Body.String())
	}
	if requests.Load() != 1 {
		t.Fatalf("JWKS requests = %d, want 1", requests.Load())
	}
	var response struct {
		User store.User `json:"user"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.User.DisplayName != "Access Captain" {
		t.Fatalf("unexpected access user: %#v", response.User)
	}
	workspaces, err := st.ListWorkspaces(context.Background(), response.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0].Role != store.WorkspaceRoleOwner {
		t.Fatalf("expected first access user to own the default workspace, got %#v", workspaces)
	}
	cookies := first.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "cc_session" || cookies[0].Value == "" || !cookies[0].HttpOnly {
		t.Fatalf("unexpected session cookies: %#v", cookies)
	}

	second := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.AddCookie(cookies[0])
	handler.ServeHTTP(second, request)
	if second.Code != http.StatusOK {
		t.Fatalf("cookie request status = %d, body = %s", second.Code, second.Body.String())
	}
	if len(second.Result().Cookies()) != 0 {
		t.Fatalf("cookie-authenticated request unexpectedly replaced its session: %#v", second.Result().Cookies())
	}
	if requests.Load() != 1 {
		t.Fatalf("cookie request fetched JWKS; requests = %d", requests.Load())
	}

	unsafe := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/workspaces", strings.NewReader(`{"name":"Cross Site","slug":"cross-site"}`))
	request.AddCookie(cookies[0])
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(unsafe, request)
	if unsafe.Code != http.StatusForbidden {
		t.Fatalf("Access session bypassed cookie CSRF checks: status=%d body=%s", unsafe.Code, unsafe.Body.String())
	}

	memberToken := signAccessTestToken(t, key, "key-1", jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": jwks.URL, "aud": "test-access-aud",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(),
		"email": "member@example.com", "common_name": "Access Member",
	})
	memberResponse := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.AddCookie(&http.Cookie{Name: "cc_session", Value: "expired-session"})
	request.Header.Set(accessAssertionHeader, memberToken)
	handler.ServeHTTP(memberResponse, request)
	if memberResponse.Code != http.StatusOK || len(memberResponse.Result().Cookies()) != 1 {
		t.Fatalf("Access did not replace an invalid session: status=%d cookies=%#v body=%s", memberResponse.Code, memberResponse.Result().Cookies(), memberResponse.Body.String())
	}
	if err := json.Unmarshal(memberResponse.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.User.DisplayName != "Access Member" {
		t.Fatalf("common name was not used for the member profile: %#v", response.User)
	}
	workspaces, err = st.ListWorkspaces(context.Background(), response.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0].Role != store.WorkspaceRoleMember {
		t.Fatalf("expected later access user to join as a member, got %#v", workspaces)
	}
	if requests.Load() != 1 {
		t.Fatalf("cached Access key was unexpectedly refetched: requests=%d", requests.Load())
	}
}

func TestAccessAssertionRejectsInvalidTokens(t *testing.T) {
	key := newAccessTestKey(t)
	jwks, _ := newAccessTestJWKSServer(t, func() map[string]*rsa.PublicKey {
		return map[string]*rsa.PublicKey{"key-1": &key.PublicKey}
	})
	st := newAccessTestStore(t)
	handler := New(st, realtime.NewHub(), Options{
		DisableDevAuth: true,
		Access:         AccessConfig{TeamDomain: jwks.URL, Audience: "test-expected-aud", HTTPClient: jwks.Client()},
	}).Handler()
	now := time.Now()
	validClaims := func() jwt.MapClaims {
		return jwt.MapClaims{
			"iss": jwks.URL, "aud": "test-expected-aud",
			"exp": now.Add(time.Hour).Unix(), "iat": now.Add(-time.Minute).Unix(),
			"email": "access@example.com",
		}
	}
	tests := []struct {
		name   string
		method jwt.SigningMethod
		key    any
		claims func() jwt.MapClaims
	}{
		{"wrong audience", jwt.SigningMethodRS256, key, func() jwt.MapClaims { claims := validClaims(); claims["aud"] = "wrong"; return claims }},
		{"wrong issuer", jwt.SigningMethodRS256, key, func() jwt.MapClaims {
			claims := validClaims()
			claims["iss"] = "https://wrong.example.com"
			return claims
		}},
		{"expired", jwt.SigningMethodRS256, key, func() jwt.MapClaims {
			claims := validClaims()
			claims["exp"] = now.Add(-time.Hour).Unix()
			return claims
		}},
		{"future issued at", jwt.SigningMethodRS256, key, func() jwt.MapClaims {
			claims := validClaims()
			claims["iat"] = now.Add(time.Hour).Unix()
			return claims
		}},
		{"missing issued at", jwt.SigningMethodRS256, key, func() jwt.MapClaims { claims := validClaims(); delete(claims, "iat"); return claims }},
		{"missing email", jwt.SigningMethodRS256, key, func() jwt.MapClaims { claims := validClaims(); delete(claims, "email"); return claims }},
		{"wrong algorithm", jwt.SigningMethodHS256, []byte("test-signing-key"), validClaims},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := signAccessTestToken(t, test.key, "key-1", test.method, test.claims())
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			request.Header.Set(accessAssertionHeader, token)
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if body := recorder.Body.String(); strings.Contains(body, token) {
				t.Fatal("authentication error echoed the assertion")
			}
		})
	}
}

func TestAccessAssertionIgnoredWhenUnconfigured(t *testing.T) {
	st := newAccessTestStore(t)
	handler := New(st, realtime.NewHub(), Options{DisableDevAuth: true}).Handler()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set(accessAssertionHeader, "untrusted")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized || len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("unconfigured access header was trusted: status=%d cookies=%#v", recorder.Code, recorder.Result().Cookies())
	}
}

func TestAccessFailureFallsThroughToLoopbackDevAuth(t *testing.T) {
	key := newAccessTestKey(t)
	jwks, _ := newAccessTestJWKSServer(t, func() map[string]*rsa.PublicKey {
		return map[string]*rsa.PublicKey{"key-1": &key.PublicKey}
	})
	st := newAccessTestStore(t)
	owner, err := st.EnsureBootstrap(context.Background(), "Local Owner", "local@example.com")
	if err != nil {
		t.Fatal(err)
	}
	handler := New(st, realtime.NewHub(), Options{
		Access: AccessConfig{TeamDomain: jwks.URL, Audience: "test-expected-aud", HTTPClient: jwks.Client()},
	}).Handler()
	token := signAccessTestToken(t, key, "key-1", jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": jwks.URL, "aud": "test-wrong-aud",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(),
		"email": "access@example.com",
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost/api/me", nil)
	request.Host = "localhost"
	request.RemoteAddr = "127.0.0.1:43210"
	request.Header.Set(accessAssertionHeader, token)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		User store.User `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.User.ID != owner.ID || len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("unexpected dev fallback response: user=%#v cookies=%#v", response.User, recorder.Result().Cookies())
	}
}

func TestAccessJWKSCacheRefetchesUnknownAndExpiredKeys(t *testing.T) {
	firstKey := newAccessTestKey(t)
	secondKey := newAccessTestKey(t)
	thirdKey := newAccessTestKey(t)
	var mu sync.RWMutex
	keys := map[string]*rsa.PublicKey{"first": &firstKey.PublicKey}
	jwks, requests := newAccessTestJWKSServer(t, func() map[string]*rsa.PublicKey {
		mu.RLock()
		defer mu.RUnlock()
		copy := make(map[string]*rsa.PublicKey, len(keys))
		for kid, key := range keys {
			copy[kid] = key
		}
		return copy
	})
	now := time.Now().UTC().Truncate(time.Second)
	verifier := newAccessVerifier(AccessConfig{TeamDomain: jwks.URL, Audience: "test-aud", HTTPClient: jwks.Client(), Now: func() time.Time { return now }})
	claims := func() jwt.MapClaims {
		return jwt.MapClaims{"iss": jwks.URL, "aud": "test-aud", "exp": now.Add(time.Hour).Unix(), "iat": now.Add(-time.Minute).Unix(), "email": "cache@example.com"}
	}
	if _, err := verifier.verify(context.Background(), signAccessTestToken(t, firstKey, "first", jwt.SigningMethodRS256, claims())); err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.verify(context.Background(), signAccessTestToken(t, firstKey, "first", jwt.SigningMethodRS256, claims())); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("cached key made %d JWKS requests, want 1", requests.Load())
	}
	now = now.Add(accessUnknownKidFloor)
	mu.Lock()
	keys = map[string]*rsa.PublicKey{"second": &secondKey.PublicKey}
	mu.Unlock()
	if _, err := verifier.verify(context.Background(), signAccessTestToken(t, secondKey, "second", jwt.SigningMethodRS256, claims())); err != nil {
		t.Fatalf("unknown kid did not trigger a successful refetch: %v", err)
	}
	if requests.Load() != 2 {
		t.Fatalf("unknown kid requests = %d, want 2", requests.Load())
	}
	mu.Lock()
	keys = map[string]*rsa.PublicKey{"second": &thirdKey.PublicKey}
	mu.Unlock()
	now = now.Add(2 * time.Minute)
	if _, err := verifier.verify(context.Background(), signAccessTestToken(t, thirdKey, "second", jwt.SigningMethodRS256, claims())); err != nil {
		t.Fatalf("expired cache did not refresh: %v", err)
	}
	if requests.Load() != 3 {
		t.Fatalf("expired cache requests = %d, want 3", requests.Load())
	}
}

func TestAccessJWKSUnknownKidRefreshFloor(t *testing.T) {
	key := newAccessTestKey(t)
	jwks, requests := newAccessTestJWKSServer(t, func() map[string]*rsa.PublicKey {
		return map[string]*rsa.PublicKey{"known": &key.PublicKey}
	})
	now := time.Now().UTC().Truncate(time.Second)
	verifier := newAccessVerifier(AccessConfig{TeamDomain: jwks.URL, Audience: "test-aud", HTTPClient: jwks.Client(), Now: func() time.Time { return now }})
	claims := jwt.MapClaims{
		"iss": jwks.URL, "aud": "test-aud",
		"exp": now.Add(time.Hour).Unix(), "iat": now.Add(-time.Minute).Unix(),
		"email": "cache@example.com",
	}
	assertion := signAccessTestToken(t, key, "unknown", jwt.SigningMethodRS256, claims)
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := verifier.verify(context.Background(), assertion); err == nil {
			t.Fatalf("unknown kid verification %d unexpectedly succeeded", attempt)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("two unknown-kid verifications made %d JWKS requests, want 1", requests.Load())
	}
}

func TestAccessJWKSFetchDoesNotFollowRedirects(t *testing.T) {
	var redirectedRequests atomic.Int64
	redirectTarget := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectedRequests.Add(1)
		_ = json.NewEncoder(w).Encode(accessJWKS{})
	}))
	t.Cleanup(redirectTarget.Close)
	teamDomain := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", redirectTarget.URL+accessJWKSPath)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(teamDomain.Close)
	verifier := newAccessVerifier(AccessConfig{TeamDomain: teamDomain.URL, Audience: "test-aud", HTTPClient: teamDomain.Client()})
	if _, _, err := verifier.fetchKeys(context.Background(), time.Now()); err == nil {
		t.Fatal("expected redirected JWKS fetch to fail")
	}
	if redirectedRequests.Load() != 0 {
		t.Fatalf("JWKS client followed redirect outside the configured team domain: requests=%d", redirectedRequests.Load())
	}
}

func newAccessTestStore(t *testing.T) *sqlitestore.Store {
	t.Helper()
	st, err := sqlitestore.Open("sqlite://" + filepath.Join(t.TempDir(), "clickclack.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return st
}

func newAccessTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func newAccessTestJWKSServer(t *testing.T, keys func() map[string]*rsa.PublicKey) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	requests := &atomic.Int64{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != accessJWKSPath {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		w.Header().Set("Cache-Control", "public, max-age=60")
		set := accessJWKS{}
		for kid, key := range keys() {
			set.Keys = append(set.Keys, accessJWK{
				Kid: kid, Kty: "RSA", Alg: "RS256", Use: "sig",
				N: base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				E: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
			})
		}
		_ = json.NewEncoder(w).Encode(set)
	}))
	t.Cleanup(server.Close)
	return server, requests
}

func signAccessTestToken(t *testing.T, key any, kid string, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
