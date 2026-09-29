package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/inboxes/backend/internal/service"
	"github.com/inboxes/backend/internal/store"
)

func TestOrgKeyRotationRefreshesDomains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer re_new_key" {
			t.Errorf("key was not trimmed")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"provider-new","name":"new.test","status":"verified"}]}`))
	}))
	defer server.Close()
	defer service.SetResendBaseURLForTest(server.URL)()
	enc, _ := service.NewEncryptionService(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	stored, synced := false, false
	mock := &store.MockStore{
		UpdateOrgAPIKeyFn: func(ctx context.Context, org, ct, iv, tag string) error {
			key, err := enc.Decrypt(ct, iv, tag)
			if err != nil || key != "re_new_key" || org != "org-a" {
				t.Fatalf("wrong stored key/workspace")
			}
			stored = true
			return nil
		},
		SyncDomainsFn: func(ctx context.Context, org string, domains []store.ResendDomainInfo) error {
			if org != "org-a" || len(domains) != 1 || domains[0].Status != "active" {
				t.Fatalf("incorrect domains: %v", domains)
			}
			synced = true
			return nil
		},
	}
	h := &OrgHandler{Store: mock, EncSvc: enc, ResendSvc: service.NewResendService(enc, nil, "", "")}
	req := withClaims(httptest.NewRequest("PATCH", "/orgs/settings", strings.NewReader(`{"api_key":"  re_new_key  "}`)), "user-a", "org-a", "admin")
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	if w.Code != 204 || !stored || !synced {
		t.Fatalf("rotation did not refresh domains: status=%d stored=%v synced=%v", w.Code, stored, synced)
	}
}

func TestOrgKeyRotationRejectsProviderFailureBeforeWrites(t *testing.T) {
	for _, status := range []int{403, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				w.Write([]byte(`{"message":"provider failure"}`))
			}))
			defer server.Close()
			defer service.SetResendBaseURLForTest(server.URL)()
			h := &OrgHandler{Store: &store.MockStore{
				UpdateOrgNameFn: func(context.Context, string, string) error {
					t.Fatal("settings were changed before key validation")
					return nil
				},
				UpdateOrgAPIKeyFn: func(context.Context, string, string, string, string) error {
					t.Fatal("invalid key was stored")
					return nil
				},
			}}
			req := withClaims(httptest.NewRequest("PATCH", "/orgs/settings", strings.NewReader(`{"name":"changed","api_key":"re_bad"}`)), "user-a", "org-a", "admin")
			w := httptest.NewRecorder()
			h.UpdateSettings(w, req)
			expected := http.StatusBadGateway
			if status == 429 {
				expected = 429
			}
			if w.Code != expected {
				t.Fatalf("status=%d want %d", w.Code, expected)
			}
		})
	}
}

func TestOrgKeyRotationDoesNotReportSuccessOnSyncFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"data":[]}`)) }))
	defer server.Close()
	defer service.SetResendBaseURLForTest(server.URL)()
	enc, _ := service.NewEncryptionService(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	h := &OrgHandler{Store: &store.MockStore{SyncDomainsFn: func(context.Context, string, []store.ResendDomainInfo) error { return errors.New("sync failed") }}, EncSvc: enc}
	req := withClaims(httptest.NewRequest("PATCH", "/orgs/settings", strings.NewReader(`{"api_key":"re_key"}`)), "user-a", "org-a", "admin")
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	if w.Code != 500 {
		t.Fatalf("sync failure reported as success: %d", w.Code)
	}
}
