package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-gateway/internal/api"
	"api-gateway/internal/app"
	"api-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestGatewayIdentity(t *testing.T) {
	const secret = "test-key"
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "account-id", "role": "user", "type": "access", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, method, path, upstreamPath, bearer, wantUser string
	}{
		{"cabinet", "GET", "/ai/assistants/list", "/assistants/list", token, "account-id"},
		{"analytics", "GET", "/api/analytics", "/analytics", token, "account-id"},
		{"assistant credential", "POST", "/webhooks/telegram", "/webhooks/telegram", "assistant-key", ""},
		{"public callback", "POST", "/twilio/webhook/assistant", "/twilio/webhook/assistant", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.upstreamPath || r.Header.Get("X-User-Id") != tc.wantUser || r.Header.Get("X-Assistant-Id") != "" {
					t.Errorf("forwarded path or identity mismatch on %s", r.URL.Path)
				}
				if tc.bearer != "" && r.Header.Get("Authorization") != "Bearer "+tc.bearer {
					t.Error("credential lost in transit")
				}
				w.WriteHeader(http.StatusNoContent)
			})
			router := api.NewRouter(&app.App{Cfg: &config.Settings{AccessSecret: secret}, AI: upstream, Auth: upstream})
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("X-User-Id", "untrusted")
			req.Header.Set("X-Assistant-Id", "untrusted")
			if tc.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tc.bearer)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d", response.Code)
			}
		})
	}
}
