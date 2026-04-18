package control

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"http-proxy-platform/internal/config"
	"http-proxy-platform/internal/netutil"

	"golang.org/x/net/webdav"
)

const adminSessionCookieName = "admin_session"
const adminCSRFCookieName = "admin_csrf"
const maxJSONBodyBytes int64 = 1 << 20

type requestActor struct {
	Username  string
	Role      string
	SessionID string
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

type loginAttemptState struct {
	Failures    int
	LastFailure time.Time
	BlockedUntil time.Time
}

type APIServer struct {
	cfg   config.Config
	store *Store
	srv   *http.Server

	mu            sync.Mutex
	rateBuckets   map[string]*tokenBucket
	loginAttempts map[string]loginAttemptState
	lastSweep     time.Time

	adminRateLimitedTotal int64
	adminLoginFailedTotal int64
	adminLoginBlockedTotal int64
}

func NewAPIServer(cfg config.Config, store *Store) *APIServer {
	h := &APIServer{
		cfg:           cfg,
		store:         store,
		rateBuckets:   make(map[string]*tokenBucket),
		loginAttempts: make(map[string]loginAttemptState),
	}
	h.srv = &http.Server{
		Addr:         cfg.AdminListenAddr(),
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	return h
}

func (h *APIServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.srv.Shutdown(shutdownCtx)
	}()

	go func() {
		err := h.srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	return <-errCh
}

func (h *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.setSecurityHeaders(w)
	ipInfo := h.requestIPInfo(r)

	if r.URL.Path == "/" || r.URL.Path == "/admin" || r.URL.Path == "/index.html" {
		h.handleAdminUI(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/webdav") {
		h.handleWebDAV(w, r)
		return
	}
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/api/admin/healthz" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ts": time.Now().Unix()})
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/admin/") {
		if !h.allowAdminAPIRequest(ipInfo.ClientIP, time.Now()) {
			atomic.AddInt64(&h.adminRateLimitedTotal, 1)
			_ = h.store.InsertAudit(r.Context(), "system", "admin_api_rate_limited", strings.TrimSpace(ipInfo.ClientIP), "path="+r.URL.Path)
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "too_many_requests", "reason": "admin_api_rate_limited"})
			return
		}
	}
	if r.URL.Path == "/api/admin/login" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}
		h.handleAdminLogin(w, r, ipInfo)
		return
	}

	actor, ok := h.authenticateAdmin(w, r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	if !h.allowMethod(actor, r.Method) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden", "reason": "readonly_cannot_write"})
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/admin/") && isUnsafeMethod(r.Method) {
		if !h.verifyCSRF(r) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden", "reason": "csrf_check_failed"})
			return
		}
	}

	if r.URL.Path == "/api/admin/logout" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}
		h.handleAdminLogout(w, r, actor)
		return
	}
	if r.URL.Path == "/api/admin/me" {
		writeJSON(w, http.StatusOK, map[string]any{
			"username":            actor.Username,
			"role":                actor.Role,
			"client_ip":           ipInfo.ClientIP,
			"remote_ip":           ipInfo.RemoteIP,
			"forwarded_for":       ipInfo.ForwardedFor,
			"real_ip_header":      ipInfo.RealIPHeader,
			"trust_proxy_headers": h.cfg.TrustProxyHeaders,
		})
		return
	}
	if r.URL.Path == "/api/admin/security-stats" {
		h.handleSecurityStats(w, r, actor)
		return
	}
	if r.URL.Path == "/api/admin/profile/password" {
		h.handleProfilePassword(w, r, actor)
		return
	}
	if r.URL.Path == "/api/admin/sessions" {
		h.handleSessions(w, r, actor)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/admin/sessions/") {
		h.handleSessionActions(w, r, actor)
		return
	}
	if r.URL.Path == "/api/admin/audits" {
		h.handleAudits(w, r)
		return
	}
	if r.URL.Path == "/api/admin/admins" {
		h.handleAdmins(w, r, actor)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/admin/admins/") {
		h.handleAdminActions(w, r, actor)
		return
	}
	if r.URL.Path == "/api/admin/users" {
		h.handleUsers(w, r, actor)
		return
	}
	if r.URL.Path == "/api/admin/users/tags" {
		h.handleUserTags(w, r)
		return
	}
	if r.URL.Path == "/api/admin/users/batch-status" {
		h.handleBatchUserStatus(w, r, actor)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/admin/users/") {
		h.handleUserActions(w, r, actor)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found"})
}

func (h *APIServer) authenticateAdmin(w http.ResponseWriter, r *http.Request) (requestActor, bool) {
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return requestActor{}, false
	}
	admin, err := h.store.ValidateAdminSession(r.Context(), cookie.Value, h.cfg.AdminSessionTTL)
	if err != nil {
		h.clearSessionCookie(w)
		h.clearCSRFCookie(w)
		return requestActor{}, false
	}
	return requestActor{Username: admin.Username, Role: admin.Role, SessionID: strings.TrimSpace(cookie.Value)}, true
}

func (h *APIServer) allowMethod(actor requestActor, method string) bool {
	if actor.Role == "super" {
		return true
	}
	if actor.Role == "readonly" {
		return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	}
	return false
}

func (h *APIServer) requireSuper(actor requestActor, w http.ResponseWriter) bool {
	if actor.Role == "super" {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden", "reason": "super_only"})
	return false
}

func (h *APIServer) handleAdminLogin(w http.ResponseWriter, r *http.Request, ipInfo netutil.ClientIPInfo) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if len(strings.TrimSpace(req.Username)) == 0 || len(strings.TrimSpace(req.Password)) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "username_and_password_required"})
		return
	}
	loginKey := loginAttemptKey(req.Username, ipInfo.ClientIP)
	if retryAfter, blocked := h.loginRetryAfterSeconds(loginKey, time.Now()); blocked {
		atomic.AddInt64(&h.adminLoginBlockedTotal, 1)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "too_many_attempts", "retry_after_seconds": retryAfter})
		return
	}
	admin, err := h.store.AuthenticateAdminPassword(r.Context(), req.Username, req.Password)
	if err != nil {
		atomic.AddInt64(&h.adminLoginFailedTotal, 1)
		_ = h.store.InsertAudit(r.Context(), "system", "admin_login_failed", strings.TrimSpace(req.Username), "client_ip="+strings.TrimSpace(ipInfo.ClientIP))
		h.recordLoginFailure(r.Context(), loginKey, strings.TrimSpace(req.Username), ipInfo.ClientIP, time.Now())
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_credentials"})
		return
	}
	h.clearLoginFailures(loginKey)
	session, err := h.store.CreateAdminSession(r.Context(), admin.Username, ipInfo.ClientIP, ipInfo.RemoteIP, r.UserAgent(), h.cfg.AdminSessionTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "session_create_failed"})
		return
	}
	csrfToken, err := randomToken(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "csrf_token_create_failed"})
		return
	}
	h.setSessionCookie(w, session.SessionID)
	h.setCSRFCookie(w, csrfToken)
	_ = h.store.InsertAudit(r.Context(), admin.Username, "admin_login", admin.Username, "client_ip="+ipInfo.ClientIP+",remote_ip="+ipInfo.RemoteIP)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "username": admin.Username, "role": admin.Role, "csrf_token": csrfToken})
}

func (h *APIServer) handleAdminLogout(w http.ResponseWriter, r *http.Request, actor requestActor) {
	if actor.SessionID != "" {
		_ = h.store.DeleteAdminSession(r.Context(), actor.SessionID)
	}
	h.clearSessionCookie(w)
	h.clearCSRFCookie(w)
	_ = h.store.InsertAudit(r.Context(), actor.Username, "admin_logout", actor.Username, "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) handleProfilePassword(w http.ResponseWriter, r *http.Request, actor requestActor) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if len(strings.TrimSpace(req.NewPassword)) < h.cfg.PasswordMinLength {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password_too_short", "min_length": h.cfg.PasswordMinLength})
		return
	}
	if err := h.store.ChangeAdminPassword(r.Context(), actor.Username, req.OldPassword, req.NewPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "change_admin_password", actor.Username, "self")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) handleSessions(w http.ResponseWriter, r *http.Request, actor requestActor) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	items, err := h.store.ListAdminSessions(r.Context(), actor.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *APIServer) handleSessionActions(w http.ResponseWriter, r *http.Request, actor requestActor) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	sessionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/sessions/"))
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_path"})
		return
	}
	items, err := h.store.ListAdminSessions(r.Context(), actor.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	allowed := false
	for _, item := range items {
		if item.SessionID == sessionID {
			allowed = true
			break
		}
	}
	if !allowed {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "session_not_found"})
		return
	}
	if err := h.store.DeleteAdminSession(r.Context(), sessionID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if actor.SessionID == sessionID {
		h.clearSessionCookie(w)
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "delete_admin_session", actor.Username, sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) handleUsers(w http.ResponseWriter, r *http.Request, actor requestActor) {
	switch r.Method {
	case http.MethodGet:
		query := UserListQuery{
			Search:       strings.TrimSpace(r.URL.Query().Get("q")),
			Tag:          strings.TrimSpace(r.URL.Query().Get("tag")),
			ExpireFilter: strings.TrimSpace(r.URL.Query().Get("expire_filter")),
			Offset:       parseIntDefault(r.URL.Query().Get("offset"), 0),
			Limit:        parseIntDefault(r.URL.Query().Get("limit"), 20),
		}
		if rawStatus := strings.TrimSpace(r.URL.Query().Get("status")); rawStatus != "" {
			status := parseIntDefault(rawStatus, -1)
			if status == 0 || status == 1 {
				query.Status = &status
			}
		}
		result, err := h.store.ListUsers(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var req struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			Tag        string `json:"tag"`
			ExpiresAt  int64  `json:"expires_at"`
			QuotaBytes int64  `json:"quota_bytes"`
			SMBEnabled bool  `json:"smb_enabled"`
			SMBQuotaBytes int64 `json:"smb_quota_bytes"`
			SpeedLimitKbps int64 `json:"speed_limit_kbps"`
			MaxDevices int    `json:"max_devices"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		username, err := normalizeUsername(req.Username)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if len(strings.TrimSpace(req.Password)) < h.cfg.PasswordMinLength {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password_too_short", "min_length": h.cfg.PasswordMinLength})
			return
		}
		err = h.store.CreateUser(r.Context(), User{
			Username:   username,
			Tag:        strings.TrimSpace(req.Tag),
			ExpiresAt:  req.ExpiresAt,
			QuotaBytes: req.QuotaBytes,
			SMBEnabled: func() int {
				if req.SMBEnabled {
					return 1
				}
				return 0
			}(),
			SMBQuotaBytes: req.SMBQuotaBytes,
			SpeedLimitKbps: req.SpeedLimitKbps,
			MaxDevices: req.MaxDevices,
		}, req.Password)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if req.SMBEnabled {
			if err := h.ensureUserSMBDir(username); err != nil {
				_ = h.store.DeleteUser(r.Context(), username)
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_dir_create_failed", "detail": err.Error()})
				return
			}
			if err := h.syncSambaUserPassword(username, req.Password); err != nil {
				_ = h.store.DeleteUser(r.Context(), username)
				_ = h.removeUserSMBDir(username)
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_account_sync_failed", "detail": err.Error()})
				return
			}
			if err := h.applySambaShare(username, true); err != nil {
				_ = h.store.DeleteUser(r.Context(), username)
				_ = h.removeUserSMBDir(username)
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_share_sync_failed", "detail": err.Error()})
				return
			}
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "create_user", username, "")
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
	}
}

func (h *APIServer) handleUserTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	tags, err := h.store.ListUserTags(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tags})
}

func (h *APIServer) handleBatchUserStatus(w http.ResponseWriter, r *http.Request, actor requestActor) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	var req struct {
		Usernames []string `json:"usernames"`
		Enabled   bool     `json:"enabled"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	count, err := h.store.BatchSetUserStatus(r.Context(), req.Usernames, req.Enabled)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	action := "batch_disable_user"
	if req.Enabled {
		action = "batch_enable_user"
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, action, strings.Join(req.Usernames, ","), "count="+strconv.FormatInt(count, 10))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "count": count})
}

func (h *APIServer) handleUserActions(w http.ResponseWriter, r *http.Request, actor requestActor) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_path"})
		return
	}
	username := strings.TrimSpace(parts[0])
	if _, err := normalizeUsername(username); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			item, err := h.store.GetUser(r.Context(), username)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeJSON(w, http.StatusNotFound, map[string]any{"error": "user_not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !h.requireSuper(actor, w) {
				return
			}
			item, err := h.store.GetUser(r.Context(), username)
			if err == nil && item.SMBEnabled == 1 {
				if smbErr := h.applySambaShare(username, false); smbErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_share_sync_failed", "detail": smbErr.Error()})
					return
				}
				if rmErr := h.removeUserSMBDir(username); rmErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_dir_remove_failed", "detail": rmErr.Error()})
					return
				}
			}
			if err := h.store.DeleteUser(r.Context(), username); err != nil {
				statusCode := http.StatusBadRequest
				if errors.Is(err, sql.ErrNoRows) {
					statusCode = http.StatusNotFound
				}
				writeJSON(w, statusCode, map[string]any{"error": err.Error()})
				return
			}
			_ = h.store.InsertAudit(r.Context(), actor.Username, "delete_user", username, "")
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		}
		return
	}

	if len(parts) == 2 && parts[1] == "devices" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}
		items, err := h.store.ListUserDevices(r.Context(), username)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":               items,
			"trust_proxy_headers": h.cfg.TrustProxyHeaders,
			"real_ip_header":      h.cfg.RealIPHeader,
		})
		return
	}

	if len(parts) == 2 && parts[1] == "overview" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}

		item, err := h.store.GetUser(r.Context(), username)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": "user_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		devices, err := h.store.ListUserDevices(r.Context(), username)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		audits, err := h.store.ListAudits(r.Context(), AuditQuery{Target: username, Limit: 20})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		smbPath, _ := h.smbPathForUser(username)

		writeJSON(w, http.StatusOK, map[string]any{
			"user":                item,
			"devices":             devices,
			"audits":              audits,
			"smb_path":            smbPath,
			"trust_proxy_headers": h.cfg.TrustProxyHeaders,
			"real_ip_header":      h.cfg.RealIPHeader,
		})
		return
	}

	if len(parts) == 2 && parts[1] == "smb-path" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}
		smbPath, err := h.smbPathForUser(username)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_, statErr := os.Stat(smbPath)
		writeJSON(w, http.StatusOK, map[string]any{"username": username, "smb_path": smbPath, "exists": statErr == nil})
		return
	}

	if len(parts) == 3 && parts[1] == "devices" {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}
		if err := h.store.DeleteUserDevice(r.Context(), username, parts[2]); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "delete_user_device", username, strings.TrimSpace(parts[2]))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	if len(parts) != 2 || r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}

	action := parts[1]
	switch action {
	case "disable":
		h.setStatus(w, r, actor, username, false)
	case "enable":
		h.setStatus(w, r, actor, username, true)
	case "set-devices":
		h.setDevices(w, r, actor, username)
	case "extend":
		h.extend(w, r, actor, username)
	case "topup":
		h.topup(w, r, actor, username)
	case "smb-topup":
		h.smbTopup(w, r, actor, username)
	case "smb-enable":
		h.smbEnable(w, r, actor, username)
	case "smb-disable":
		h.smbDisable(w, r, actor, username)
	case "set-speed":
		h.setSpeedLimit(w, r, actor, username)
	case "set-quota":
		h.setQuota(w, r, actor, username)
	case "set-smb-quota":
		h.setSMBQuota(w, r, actor, username)
	case "usage":
		h.usage(w, r, actor, username)
	case "usage-reset":
		h.resetUsage(w, r, actor, username)
	case "password":
		h.setUserPassword(w, r, actor, username)
	case "set-tag":
		h.setUserTag(w, r, actor, username)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "unknown_action"})
	}
}

func (h *APIServer) setStatus(w http.ResponseWriter, r *http.Request, actor requestActor, username string, enabled bool) {
	if err := h.store.SetUserStatus(r.Context(), username, enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	action := "disable_user"
	if enabled {
		action = "enable_user"
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, action, username, "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) extend(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Days int `json:"days"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.ExtendUserDays(r.Context(), username, req.Days); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "extend_user", username, "days="+strconv.Itoa(req.Days))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) topup(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Bytes int64 `json:"bytes"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.TopUpQuota(r.Context(), username, req.Bytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "topup_user", username, "bytes="+strconv.FormatInt(req.Bytes, 10))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) smbTopup(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Bytes int64 `json:"bytes"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.TopUpSMBQuota(r.Context(), username, req.Bytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "topup_user_smb", username, "bytes="+strconv.FormatInt(req.Bytes, 10))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) smbEnable(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	if err := h.ensureUserSMBDir(username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_dir_create_failed", "detail": err.Error()})
		return
	}
	if err := h.applySambaShare(username, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_share_sync_failed", "detail": err.Error()})
		return
	}
	if err := h.store.SetUserSMBEnabled(r.Context(), username, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "enable_user_smb", username, "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) smbDisable(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	if err := h.applySambaShare(username, false); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_share_sync_failed", "detail": err.Error()})
		return
	}
	if err := h.removeUserSMBDir(username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_dir_remove_failed", "detail": err.Error()})
		return
	}
	if err := h.store.SetUserSMBEnabled(r.Context(), username, false); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "disable_user_smb", username, "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) setSpeedLimit(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Kbps int64 `json:"kbps"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.SetUserSpeedLimitKbps(r.Context(), username, req.Kbps); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "set_user_speed_limit", username, "kbps="+strconv.FormatInt(req.Kbps, 10))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) setQuota(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Bytes int64 `json:"bytes"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.SetUserQuota(r.Context(), username, req.Bytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "set_user_quota", username, "bytes="+strconv.FormatInt(req.Bytes, 10))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) setSMBQuota(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Bytes int64 `json:"bytes"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.SetUserSMBQuota(r.Context(), username, req.Bytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "set_user_smb_quota", username, "bytes="+strconv.FormatInt(req.Bytes, 10))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) setDevices(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		MaxDevices int `json:"max_devices"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.SetUserMaxDevices(r.Context(), username, req.MaxDevices); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "set_user_max_devices", username, "max_devices="+strconv.Itoa(req.MaxDevices))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) setUserPassword(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if len(strings.TrimSpace(req.Password)) < h.cfg.PasswordMinLength {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password_too_short", "min_length": h.cfg.PasswordMinLength})
		return
	}
	if err := h.store.SetUserPassword(r.Context(), username, req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if item, err := h.store.GetUser(r.Context(), username); err == nil && item.SMBEnabled == 1 {
		if err := h.syncSambaUserPassword(username, req.Password); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "smb_account_sync_failed", "detail": err.Error()})
			return
		}
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "set_user_password", username, "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) smbRootDir() string {
	root := strings.TrimSpace(h.cfg.SMBRootDir)
	if root == "" {
		root = "/mnt/mmc0-4/proxy-platform/smb"
	}
	return filepath.Clean(root)
}

func (h *APIServer) smbPathForUser(username string) (string, error) {
	name, err := normalizeUsername(username)
	if err != nil {
		return "", err
	}
	root := h.smbRootDir()
	userPath := filepath.Join(root, name)
	rel, err := filepath.Rel(root, userPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid_smb_path")
	}
	return userPath, nil
}

func (h *APIServer) ensureUserSMBDir(username string) error {
	path, err := h.smbPathForUser(username)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0o770); err != nil {
		return err
	}
	user, err := normalizeUsername(username)
	if err != nil {
		return err
	}
	quotedUser := shellQuote(user)
	quotedPath := shellQuote(path)
	script := "if id -u " + quotedUser + " >/dev/null 2>&1; then " +
		"chown " + quotedUser + ":" + quotedUser + " " + quotedPath + " >/dev/null 2>&1 || chown " + quotedUser + " " + quotedPath + " >/dev/null 2>&1; " +
		"fi; " +
		"chmod 0770 " + quotedPath + " >/dev/null 2>&1 || true"
	if out, err := exec.Command("/bin/sh", "-c", script).CombinedOutput(); err != nil {
		return fmt.Errorf("smb_dir_permission_fix_failed: %v output=%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (h *APIServer) removeUserSMBDir(username string) error {
	path, err := h.smbPathForUser(username)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func (h *APIServer) syncSambaUserPassword(username, password string) error {
	user, err := normalizeUsername(username)
	pass := strings.TrimSpace(password)
	if err != nil || pass == "" {
		return errors.New("username_or_password_empty")
	}
	quotedUser := shellQuote(user)
	quotedPass := shellQuote(pass)
	script := "if ! command -v smbpasswd >/dev/null 2>&1; then echo smbpasswd_not_found; exit 1; fi; " +
		"if ! id -u " + quotedUser + " >/dev/null 2>&1; then " +
		"(adduser -D -H " + quotedUser + " >/dev/null 2>&1 || adduser -H " + quotedUser + " >/dev/null 2>&1 || useradd -M -s /bin/false " + quotedUser + " >/dev/null 2>&1 || true); fi; " +
		"if smbpasswd --help 2>&1 | grep -q -- ' -s'; then " +
		"printf '%s\\n%s\\n' " + quotedPass + " " + quotedPass + " | smbpasswd -a -s " + quotedUser + " >/dev/null 2>&1; " +
		"else " +
		"printf '%s\\n%s\\n' " + quotedPass + " " + quotedPass + " | smbpasswd -a " + quotedUser + " >/dev/null 2>&1; " +
		"fi; " +
		"smbpasswd -e " + quotedUser + " >/dev/null 2>&1 || true"
	if out, err := exec.Command("/bin/sh", "-c", script).CombinedOutput(); err != nil {
		return fmt.Errorf("samba_password_sync_failed: %v output=%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (h *APIServer) applySambaShare(username string, enabled bool) error {
	user, err := normalizeUsername(username)
	if err != nil {
		return err
	}
	userPath, err := h.smbPathForUser(user)
	if err != nil {
		return err
	}
	quotedUser := shellQuote(user)
	quotedPath := shellQuote(userPath)
	state := "0"
	if enabled {
		state = "1"
	}
	script := "cfg='samba4'; [ -f /etc/config/samba4 ] || cfg='samba'; " +
		"for s in $(uci -q show $cfg | sed -n \"s/^\\($cfg\\.[^.]*\\)=sambashare$/\\1/p\"); do " +
		"n=$(uci -q get $s.name 2>/dev/null); [ \"$n\" = " + quotedUser + " ] && uci -q delete $s; done; " +
		"if [ " + state + " = 1 ]; then " +
		"uci -q add $cfg sambashare >/dev/null; " +
		"sec=$(uci -q show $cfg | sed -n \"s/^\\($cfg\\.[^.]*\\)=sambashare$/\\1/p\" | tail -n1); " +
		"uci -q set $sec.name=" + quotedUser + "; " +
		"uci -q set $sec.path=" + quotedPath + "; " +
		"uci -q set $sec.read_only='no'; " +
		"uci -q set $sec.guest_ok='no'; " +
		"uci -q set $sec.users=" + quotedUser + "; " +
		"uci -q set $sec.create_mask='0660'; " +
		"uci -q set $sec.dir_mask='0770'; " +
		"fi; " +
		"uci -q commit $cfg; /etc/init.d/samba4 restart >/dev/null 2>&1 || /etc/init.d/samba restart >/dev/null 2>&1"
	if out, err := exec.Command("/bin/sh", "-c", script).CombinedOutput(); err != nil {
		return fmt.Errorf("samba_share_sync_failed: %v output=%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func shellQuote(raw string) string {
	if raw == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(raw, "'", "'\\''") + "'"
}

func (h *APIServer) handleWebDAV(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	pathSuffix := strings.TrimPrefix(r.URL.Path, "/webdav")
	decodedPath, err := url.PathUnescape(pathSuffix)
	if err != nil || strings.Contains(decodedPath, "\\") || strings.Contains(decodedPath, "..") {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_webdav_path"))
		return
	}
	cleanPath := path.Clean("/" + pathSuffix)
	if strings.Contains(cleanPath, "..") {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_webdav_path"))
		return
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="webdav"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("missing_basic_auth"))
		return
	}

	user, err := h.store.AuthenticateUserPassword(r.Context(), username, password)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="webdav"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid_credentials"))
		return
	}
	if user.SMBEnabled != 1 {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("smb_not_enabled"))
		return
	}

	userPath, err := h.smbPathForUser(user.Username)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_user_path"))
		return
	}
	if err := h.ensureUserSMBDir(user.Username); err != nil {
		log.Printf("webdav mkdir failed user=%s path=%s err=%v", user.Username, userPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("mkdir_failed"))
		return
	}

	handler := &webdav.Handler{
		Prefix:     "/webdav",
		FileSystem: webdav.Dir(userPath),
		LockSystem: webdav.NewMemLS(),
	}
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/webdav" + cleanPath
	handler.ServeHTTP(w, r2)
}

func (h *APIServer) resetUsage(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	if err := h.store.ResetUserUsage(r.Context(), username); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "reset_user_usage", username, "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) usage(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Bytes int64 `json:"bytes"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.AddUsage(r.Context(), username, req.Bytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "usage_user", username, "bytes="+strconv.FormatInt(req.Bytes, 10))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) setUserTag(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		Tag string `json:"tag"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if err := h.store.SetUserTag(r.Context(), username, req.Tag); err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, sql.ErrNoRows) {
			statusCode = http.StatusNotFound
		}
		writeJSON(w, statusCode, map[string]any{"error": err.Error()})
		return
	}
	_ = h.store.InsertAudit(r.Context(), actor.Username, "set_user_tag", username, strings.TrimSpace(req.Tag))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *APIServer) handleAudits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	query := AuditQuery{Limit: 100}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			query.Limit = value
		}
	}
	query.Actor = strings.TrimSpace(r.URL.Query().Get("actor"))
	query.Action = strings.TrimSpace(r.URL.Query().Get("action"))
	query.Target = strings.TrimSpace(r.URL.Query().Get("target"))
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
			query.CreatedFrom = value
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("to")); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
			query.CreatedTo = value
		}
	}
	items, err := h.store.ListAudits(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *APIServer) handleSecurityStats(w http.ResponseWriter, r *http.Request, actor requestActor) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	h.mu.Lock()
	blockedActive := 0
	now := time.Now()
	for _, state := range h.loginAttempts {
		if state.BlockedUntil.After(now) {
			blockedActive++
		}
	}
	rateBucketEntries := len(h.rateBuckets)
	loginAttemptEntries := len(h.loginAttempts)
	h.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"actor":                     actor.Username,
		"admin_rate_limited_total":  atomic.LoadInt64(&h.adminRateLimitedTotal),
		"admin_login_failed_total":  atomic.LoadInt64(&h.adminLoginFailedTotal),
		"admin_login_blocked_total": atomic.LoadInt64(&h.adminLoginBlockedTotal),
		"blocked_active":            blockedActive,
		"rate_bucket_entries":       rateBucketEntries,
		"login_attempt_entries":     loginAttemptEntries,
		"ui_alert_rate_limited_delta": h.cfg.AdminUIAlertRateLimitDelta,
		"ui_alert_login_failed_delta": h.cfg.AdminUIAlertLoginFailDelta,
		"ui_alert_login_blocked_delta": h.cfg.AdminUIAlertLoginBlockDelta,
		"ui_alert_cooldown_seconds": int64(h.cfg.AdminUIAlertCooldown.Seconds()),
	})
}

func (h *APIServer) handleAdmins(w http.ResponseWriter, r *http.Request, actor requestActor) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.store.ListAdmins(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if !h.requireSuper(actor, w) {
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		username, err := normalizeUsername(req.Username)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if len(strings.TrimSpace(req.Password)) < h.cfg.PasswordMinLength {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password_too_short", "min_length": h.cfg.PasswordMinLength})
			return
		}
		if err := h.store.CreateAdmin(r.Context(), username, req.Password, req.Role); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "create_admin", username, "role="+strings.ToLower(strings.TrimSpace(req.Role)))
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
	}
}

func (h *APIServer) handleAdminActions(w http.ResponseWriter, r *http.Request, actor requestActor) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/admins/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_path"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	if !h.requireSuper(actor, w) {
		return
	}

	username := strings.TrimSpace(parts[0])
	action := strings.TrimSpace(parts[1])
	switch action {
	case "disable":
		if err := h.store.SetAdminStatus(r.Context(), username, false); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "disable_admin", username, "")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "enable":
		if err := h.store.SetAdminStatus(r.Context(), username, true); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "enable_admin", username, "")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "set-role":
		var req struct {
			Role string `json:"role"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if err := h.store.SetAdminRole(r.Context(), username, req.Role); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "set_admin_role", username, "role="+strings.ToLower(strings.TrimSpace(req.Role)))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "password":
		var req struct {
			Password string `json:"password"`
		}
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if len(strings.TrimSpace(req.Password)) < h.cfg.PasswordMinLength {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password_too_short", "min_length": h.cfg.PasswordMinLength})
			return
		}
		if err := h.store.SetAdminPassword(r.Context(), username, req.Password); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "set_admin_password", username, "")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "unknown_action"})
	}
}

func (h *APIServer) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.AdminCookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(h.cfg.AdminSessionTTL.Seconds()),
	})
}

func (h *APIServer) setCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminCSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cfg.AdminCookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(h.cfg.AdminSessionTTL.Seconds()),
	})
}

func (h *APIServer) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.AdminCookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (h *APIServer) clearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminCSRFCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cfg.AdminCookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (h *APIServer) setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	if h.cfg.AdminCookieSecure {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}
}

func (h *APIServer) verifyCSRF(r *http.Request) bool {
	tokenCookie, err := r.Cookie(adminCSRFCookieName)
	if err == nil {
		cookieValue := strings.TrimSpace(tokenCookie.Value)
		headerValue := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
		if cookieValue != "" && headerValue != "" && subtle.ConstantTimeCompare([]byte(cookieValue), []byte(headerValue)) == 1 {
			return true
		}
	}
	return isSameOriginRequest(r)
}

func isSameOriginRequest(r *http.Request) bool {
	expectedHost := strings.ToLower(strings.TrimSpace(r.Host))
	if expectedHost == "" {
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return strings.ToLower(u.Host) == expectedHost
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		return strings.ToLower(u.Host) == expectedHost
	}
	return false
}

func isUnsafeMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func randomToken(size int) (string, error) {
	if size <= 0 {
		size = 32
	}
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected_json_tokens")
	}
	return nil
}

func (h *APIServer) requestIPInfo(r *http.Request) netutil.ClientIPInfo {
	return netutil.ResolveClientIP(r, h.cfg.TrustProxyHeaders, h.cfg.RealIPHeader)
}

func (h *APIServer) allowAdminAPIRequest(clientIP string, now time.Time) bool {
	if h.cfg.AdminRateLimitRPS <= 0 || h.cfg.AdminRateLimitBurst <= 0 {
		return true
	}
	key := strings.TrimSpace(clientIP)
	if key == "" {
		key = "unknown"
	}
	rps := float64(h.cfg.AdminRateLimitRPS)
	burst := float64(h.cfg.AdminRateLimitBurst)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.sweepStateLocked(now)

	bucket, ok := h.rateBuckets[key]
	if !ok {
		h.rateBuckets[key] = &tokenBucket{tokens: burst - 1, last: now}
		return true
	}
	elapsed := now.Sub(bucket.last).Seconds()
	if elapsed > 0 {
		bucket.tokens = math.Min(burst, bucket.tokens+elapsed*rps)
		bucket.last = now
	}
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens -= 1
	return true
}

func (h *APIServer) loginRetryAfterSeconds(key string, now time.Time) (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sweepStateLocked(now)
	state, ok := h.loginAttempts[key]
	if !ok {
		return 0, false
	}
	if state.BlockedUntil.After(now) {
		return int(math.Ceil(state.BlockedUntil.Sub(now).Seconds())), true
	}
	if !state.BlockedUntil.IsZero() {
		state.BlockedUntil = time.Time{}
		state.Failures = 0
		h.loginAttempts[key] = state
	}
	return 0, false
}

func (h *APIServer) recordLoginFailure(ctx context.Context, key, username, clientIP string, now time.Time) {
	if h.cfg.AdminLoginMaxFails <= 0 {
		return
	}
	shouldAudit := false
	h.mu.Lock()
	h.sweepStateLocked(now)

	state := h.loginAttempts[key]
	if !state.LastFailure.IsZero() && now.Sub(state.LastFailure) > h.cfg.AdminLoginFailWindow {
		state.Failures = 0
	}
	state.Failures++
	state.LastFailure = now
	if state.Failures >= h.cfg.AdminLoginMaxFails {
		state.Failures = 0
		state.BlockedUntil = now.Add(h.cfg.AdminLoginBlockFor)
		shouldAudit = true
	}
	h.loginAttempts[key] = state
	h.mu.Unlock()
	if shouldAudit {
		_ = h.store.InsertAudit(ctx, "system", "admin_login_blocked", username, "client_ip="+strings.TrimSpace(clientIP)+",blocked_for="+h.cfg.AdminLoginBlockFor.String())
	}
}

func (h *APIServer) sweepStateLocked(now time.Time) {
	if !h.lastSweep.IsZero() && now.Sub(h.lastSweep) < time.Minute {
		return
	}
	h.lastSweep = now
	rateTTL := 10 * time.Minute
	for key, bucket := range h.rateBuckets {
		if now.Sub(bucket.last) > rateTTL {
			delete(h.rateBuckets, key)
		}
	}
	loginTTL := h.cfg.AdminLoginFailWindow * 2
	if loginTTL <= 0 {
		loginTTL = 30 * time.Minute
	}
	for key, state := range h.loginAttempts {
		if state.BlockedUntil.After(now) {
			continue
		}
		if now.Sub(state.LastFailure) > loginTTL {
			delete(h.loginAttempts, key)
		}
	}
}

func (h *APIServer) clearLoginFailures(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.loginAttempts, key)
}

func loginAttemptKey(username, clientIP string) string {
	return strings.ToLower(strings.TrimSpace(username)) + "|" + strings.TrimSpace(clientIP)
}

func parseIntDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

