package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const loginCodeTTL = 10 * time.Minute

// Compare and optionally consume the same challenge in one operation. Workspace
// selection checks the code first and consumes it when the user picks a workspace.
var checkLoginCode = redis.NewScript(`
if redis.call("GET", KEYS[1]) ~= ARGV[1] then return 0 end
if ARGV[2] == "consume" then redis.call("DEL", KEYS[1]) end
return 1
`)

func (h *AuthHandler) loginCodeMAC(email, code string) string {
	mac := hmac.New(sha256.New, []byte(h.Secret))
	fmt.Fprintf(mac, "login-code\x00%s\x00%s", email, code)
	return hex.EncodeToString(mac.Sum(nil))
}

func (h *AuthHandler) RequestLoginCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	email := normalizeEmail(req.Email)
	if err := validateEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	if h.ResendSvc == nil || !h.ResendSvc.HasSystemKey(ctx) {
		writeError(w, http.StatusServiceUnavailable, "Sign-in email is temporarily unavailable.")
		return
	}
	from := h.ResendSvc.GetSystemFrom(ctx)
	if from == "" || h.RDB == nil {
		writeError(w, http.StatusServiceUnavailable, "Sign-in email is temporarily unavailable.")
		return
	}

	memberships, err := h.Store.GetLoginMemberships(ctx, email)
	if err == nil && len(activeLoginMemberships(memberships, "")) > 0 && (h.StripeKey == "" || memberships[0].EmailVerified) {
		if err := h.sendLoginCode(r, email, from); err != nil {
			slog.Error("auth: could not send sign-in code", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "If that email has access, a sign-in code was sent."})
}

func (h *AuthHandler) sendLoginCode(r *http.Request, email, from string) error {
	number, err := rand.Int(rand.Reader, big.NewInt(100000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%08d", number.Int64())
	key := h.loginCodeKey(email)
	digest := h.loginCodeMAC(email, code)
	ctx := r.Context()
	if err := h.RDB.Set(ctx, key, digest, loginCodeTTL).Err(); err != nil {
		return err
	}
	_, err = h.ResendSvc.SystemFetch(ctx, "POST", "/emails", map[string]interface{}{
		"from":    from,
		"to":      []string{email},
		"subject": "Your Inboxes sign-in code",
		"text":    "Your Inboxes sign-in code is " + code + ". It expires in 10 minutes and can be used once. If you did not request it, you can ignore this email.",
	})
	if err != nil {
		checkLoginCode.Run(ctx, h.RDB, []string{key}, digest, "consume")
	}
	return err
}

func (h *AuthHandler) loginCodeKey(email string) string {
	return "auth:login-code:" + h.loginCodeMAC(email, "")

}

func (h *AuthHandler) VerifyLoginCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
		OrgID string `json:"org_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	email := normalizeEmail(req.Email)
	if err := validateEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := strconv.ParseUint(req.Code, 10, 32); len(req.Code) != 8 || err != nil {
		writeError(w, http.StatusBadRequest, "Enter the 8-digit sign-in code.")
		return
	}
	ctx := r.Context()
	memberships, err := h.Store.GetLoginMemberships(ctx, email)
	active := activeLoginMemberships(memberships, req.OrgID)
	if err != nil || len(active) == 0 || (h.StripeKey != "" && !memberships[0].EmailVerified) {
		writeError(w, http.StatusUnauthorized, "Invalid or expired sign-in code.")
		return
	}
	mode := "check"
	if len(active) == 1 {
		mode = "consume"
	}
	key := h.loginCodeKey(email)
	valid, err := checkLoginCode.Run(ctx, h.RDB, []string{key}, h.loginCodeMAC(email, req.Code), mode).Int()
	if err != nil {
		slog.Error("auth: could not verify sign-in code", "error", err)
		writeError(w, http.StatusServiceUnavailable, "Code verification is temporarily unavailable.")
		return
	}
	if valid != 1 {
		writeError(w, http.StatusUnauthorized, "Invalid or expired sign-in code.")
		return
	}
	h.completeLogin(w, r, email, active)
}
