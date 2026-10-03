package handler

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inboxes/backend/internal/middleware"
	"github.com/jackc/pgx/v5"
)

const maxAliasPhotoBytes = 3 * 1024 * 1024

func (h *AliasHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetCurrentUser(r.Context())
	aliasID := chi.URLParam(r, "id")
	allowed, err := h.Store.CanManageAliasPhoto(r.Context(), aliasID, claims.OrgID, claims.UserID, claims.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check alias access")
		return
	}
	if !allowed {
		writeError(w, http.StatusNotFound, "alias not found or photo editing is not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAliasPhotoBytes+64*1024)
	if err := r.ParseMultipartForm(maxAliasPhotoBytes + 64*1024); err != nil {
		writeError(w, http.StatusBadRequest, "photo upload is invalid or too large (max 3 MB)")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "photo is required")
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxAliasPhotoBytes+1))
	if err != nil || len(raw) > maxAliasPhotoBytes {
		writeError(w, http.StatusBadRequest, "photo is too large (max 3 MB)")
		return
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "png" && format != "jpeg") {
		writeError(w, http.StatusBadRequest, "photo must be a valid PNG or JPEG")
		return
	}
	if cfg.Width != cfg.Height || cfg.Width < 1 || cfg.Width > 1024 {
		writeError(w, http.StatusBadRequest, "crop the photo to a square of at most 1024 pixels")
		return
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		writeError(w, http.StatusBadRequest, "photo could not be decoded")
		return
	}
	// Re-encode to strip metadata and flatten transparency onto white.
	flat := image.NewRGBA(img.Bounds())
	draw.Draw(flat, flat.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, img.Bounds().Min, draw.Over)
	var data bytes.Buffer
	if err := jpeg.Encode(&data, flat, &jpeg.Options{Quality: 90}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare photo")
		return
	}
	version := uuid.NewString()
	n, err := h.Store.SetAliasPhoto(r.Context(), aliasID, claims.OrgID, claims.UserID, claims.Role, data.Bytes(), &version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save photo")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "alias not found or photo editing is not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"avatar_url": "/api/public/alias-photos/" + aliasID + "/" + version})
}

func (h *AliasHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetCurrentUser(r.Context())
	n, err := h.Store.SetAliasPhoto(r.Context(), chi.URLParam(r, "id"), claims.OrgID, claims.UserID, claims.Role, nil, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove photo")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "alias not found or photo editing is not allowed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AliasHandler) PublicPhoto(w http.ResponseWriter, r *http.Request) {
	aliasID, version := chi.URLParam(r, "id"), chi.URLParam(r, "version")
	w.Header().Set("Cache-Control", "no-cache")
	if _, err := uuid.Parse(aliasID); err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := uuid.Parse(version); err != nil {
		http.NotFound(w, r)
		return
	}
	data, err := h.Store.GetPublicAliasPhoto(r.Context(), aliasID, version)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && len(data) == 0) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load photo")
		return
	}
	// Check the current record before conditional requests, so removed photos
	// cannot keep returning 304 after their public URL has been revoked.
	etag := `"` + version + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "image/jpeg")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}
