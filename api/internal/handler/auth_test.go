package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaludineth/pingwatch/api/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// mockUserStore is an in-memory implementation of the userStore interface.
type mockUserStore struct {
	users map[string]*store.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*store.User)}
}

func (m *mockUserStore) CreateUser(_ context.Context, email, hash string) (string, error) {
	if _, exists := m.users[email]; exists {
		return "", store.ErrDuplicate
	}
	id := "user-" + email
	m.users[email] = &store.User{ID: id, Email: email, PasswordHash: hash}
	return id, nil
}

func (m *mockUserStore) GetUserByEmail(_ context.Context, email string) (*store.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, store.ErrNotFound
	}
	return u, nil
}

// seedUser inserts a user with a pre-computed bcrypt hash so login tests don't
// have to go through the (intentionally slow) registration path.
func (m *mockUserStore) seedUser(email, password string) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	m.users[email] = &store.User{ID: "user-" + email, Email: email, PasswordHash: string(hash)}
}

const testSecret = "test-secret-32-chars-long-enough"

func newAuthHandler(s *mockUserStore) *AuthHandler {
	return NewAuthHandler(s, testSecret)
}

func postJSON(handler http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// --- Register tests ---

func TestRegister_Success(t *testing.T) {
	h := newAuthHandler(newMockUserStore())
	rr := postJSON(h.Register, "/api/auth/register", map[string]string{
		"email":    "alice@example.com",
		"password": "supersecret",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body)
	}

	// JWT cookie must be set and httpOnly
	cookies := rr.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c
		}
	}
	if tokenCookie == nil {
		t.Fatal("expected 'token' cookie to be set")
	}
	if !tokenCookie.HttpOnly {
		t.Error("expected cookie to be HttpOnly")
	}
	if tokenCookie.Value == "" {
		t.Error("expected non-empty token value")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	ms := newMockUserStore()
	h := newAuthHandler(ms)
	payload := map[string]string{"email": "bob@example.com", "password": "password123"}

	postJSON(h.Register, "/api/auth/register", payload) // first call — OK
	rr := postJSON(h.Register, "/api/auth/register", payload) // second call — conflict

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d — body: %s", rr.Code, rr.Body)
	}
	assertErrorBody(t, rr, "email already registered")
}

func TestRegister_MissingFields(t *testing.T) {
	h := newAuthHandler(newMockUserStore())

	cases := []map[string]string{
		{"email": "", "password": "secret"},
		{"email": "x@x.com", "password": ""},
		{},
	}
	for _, c := range cases {
		rr := postJSON(h.Register, "/api/auth/register", c)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("payload %v: expected 400, got %d", c, rr.Code)
		}
	}
}

// --- Login tests ---

func TestLogin_Success(t *testing.T) {
	ms := newMockUserStore()
	ms.seedUser("carol@example.com", "mypassword")
	h := newAuthHandler(ms)

	rr := postJSON(h.Login, "/api/auth/login", map[string]string{
		"email":    "carol@example.com",
		"password": "mypassword",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body)
	}

	var tokenCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "token" {
			tokenCookie = c
		}
	}
	if tokenCookie == nil {
		t.Fatal("expected 'token' cookie")
	}
	if tokenCookie.Value == "" {
		t.Error("expected non-empty token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	ms := newMockUserStore()
	ms.seedUser("dave@example.com", "correcthorse")
	h := newAuthHandler(ms)

	rr := postJSON(h.Login, "/api/auth/login", map[string]string{
		"email":    "dave@example.com",
		"password": "wrongpassword",
	})

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d — body: %s", rr.Code, rr.Body)
	}
	assertErrorBody(t, rr, "invalid credentials")
}

func TestLogin_UnknownEmail(t *testing.T) {
	h := newAuthHandler(newMockUserStore())

	rr := postJSON(h.Login, "/api/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "doesn'tmatter",
	})

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d — body: %s", rr.Code, rr.Body)
	}
	assertErrorBody(t, rr, "invalid credentials")
}

func TestLogin_MissingFields(t *testing.T) {
	h := newAuthHandler(newMockUserStore())

	rr := postJSON(h.Login, "/api/auth/login", map[string]string{"email": "x@x.com"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// assertErrorBody checks the response has {"error": wantMsg}.
func assertErrorBody(t *testing.T, rr *httptest.ResponseRecorder, wantMsg string) {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode error body: %v", err)
	}
	if body["error"] != wantMsg {
		t.Errorf("error body: got %q, want %q", body["error"], wantMsg)
	}
}
