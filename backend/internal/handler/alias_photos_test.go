package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inboxes/backend/internal/store"
	"github.com/jackc/pgx/v5"
)

func photoUploadRequest(t *testing.T, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("photo", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	file.Write(data)
	form.Close()
	r := httptest.NewRequest("POST", "/api/aliases/a1/photo", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	r = r.WithContext(newChiRouteContext("id", "a1"))
	return withClaims(r, "u1", "o1", "member")
}

func TestAliasPhotoUploadNormalizesAndScopes(t *testing.T) {
	var raw bytes.Buffer
	png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 512, 512)))
	var savedVersion string
	h := AliasHandler{Store: &store.MockStore{
		CanManageAliasPhotoFn: func(ctx context.Context, id, org, user, role string) (bool, error) {
			if id != "a1" || org != "o1" || user != "u1" || role != "member" {
				t.Fatal("wrong access scope")
			}
			return true, nil
		},
		SetAliasPhotoFn: func(ctx context.Context, id, org, user, role string, data []byte, version *string) (int64, error) {
			if id != "a1" || org != "o1" || user != "u1" || role != "member" {
				t.Fatal("wrong save scope")
			}
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
			if err != nil || cfg.Width != 512 || cfg.Height != 512 {
				t.Fatal("photo was not normalized to JPEG")
			}
			if version == nil {
				t.Fatal("missing version")
			}
			if _, err := uuid.Parse(*version); err != nil {
				t.Fatal(err)
			}
			savedVersion = *version
			return 1, nil
		},
	}}
	w := httptest.NewRecorder()
	h.UploadPhoto(w, photoUploadRequest(t, raw.Bytes()))
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["avatar_url"] != "/api/public/alias-photos/a1/"+savedVersion {
		t.Fatalf("wrong URL: %v", result)
	}
}

func TestAliasPhotoRejectsUnauthorizedAndInvalidUploads(t *testing.T) {
	var rectangle, large bytes.Buffer
	png.Encode(&rectangle, image.NewRGBA(image.Rect(0, 0, 200, 100)))
	png.Encode(&large, image.NewRGBA(image.Rect(0, 0, 1025, 1025)))
	for _, tc := range []struct {
		name    string
		data    []byte
		allowed bool
		code    int
	}{
		{"unauthorized", []byte("image"), false, 404},
		{"SVG", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), true, 400},
		{"rectangle", rectangle.Bytes(), true, 400},
		{"dimensions", large.Bytes(), true, 400},
		{"size", bytes.Repeat([]byte("x"), maxAliasPhotoBytes+1), true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := AliasHandler{Store: &store.MockStore{
				CanManageAliasPhotoFn: func(context.Context, string, string, string, string) (bool, error) { return tc.allowed, nil },
				SetAliasPhotoFn: func(context.Context, string, string, string, string, []byte, *string) (int64, error) {
					t.Fatal("invalid upload reached storage")
					return 0, nil
				},
			}}
			w := httptest.NewRecorder()
			h.UploadPhoto(w, photoUploadRequest(t, tc.data))
			if w.Code != tc.code {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAliasPhotoDeleteClearsBothFields(t *testing.T) {
	h := AliasHandler{Store: &store.MockStore{SetAliasPhotoFn: func(ctx context.Context, id, org, user, role string, data []byte, version *string) (int64, error) {
		if id != "a1" || org != "o1" || user != "u1" || data != nil || version != nil {
			t.Fatal("wrong deletion parameters")
		}
		return 1, nil
	}}}
	r := httptest.NewRequest("DELETE", "/api/aliases/a1/photo", nil)
	r = withClaims(r.WithContext(newChiRouteContext("id", "a1")), "u1", "o1", "member")
	w := httptest.NewRecorder()
	h.DeletePhoto(w, r)
	if w.Code != 204 {
		t.Fatalf("status %d", w.Code)
	}
}

func TestPublicAliasPhotoWithoutAuthenticationAndRevocation(t *testing.T) {
	id, version := uuid.NewString(), uuid.NewString()
	var resultErr error
	h := AliasHandler{Store: &store.MockStore{GetPublicAliasPhotoFn: func(ctx context.Context, aliasID, v string) ([]byte, error) {
		if aliasID != id || v != version {
			t.Fatal("wrong photo requested")
		}
		return []byte("jpeg bytes"), resultErr
	}}}
	for _, tc := range []struct {
		err  error
		etag string
		code int
	}{
		{nil, "", 200}, {nil, `"` + version + `"`, 304}, {pgx.ErrNoRows, `"` + version + `"`, 404}, {errors.New("offline"), "", 500},
	} {
		resultErr = tc.err
		r := httptest.NewRequest("GET", "/photo", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		rctx.URLParams.Add("version", version)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		r.Header.Set("If-None-Match", tc.etag)
		w := httptest.NewRecorder()
		h.PublicPhoto(w, r)
		if w.Code != tc.code {
			t.Fatalf("status %d, want %d", w.Code, tc.code)
		}
		if tc.code == 200 && w.Header().Get("Content-Type") != "image/jpeg" {
			t.Fatal("wrong content type")
		}
	}
}
