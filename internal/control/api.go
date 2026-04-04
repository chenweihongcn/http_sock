package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"http-proxy-platform/internal/config"
	"http-proxy-platform/internal/netutil"
)

const adminSessionCookieName = "admin_session"

type requestActor struct {
	Username  string
	Role      string
	SessionID string
}

type APIServer struct {
	cfg   config.Config
	store *Store
	srv   *http.Server
}

func NewAPIServer(cfg config.Config, store *Store) *APIServer {
	h := &APIServer{cfg: cfg, store: store}
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
	if r.URL.Path == "/" || r.URL.Path == "/admin" || r.URL.Path == "/index.html" {
		h.handleAdminUI(w, r)
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
	if r.URL.Path == "/api/admin/login" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}
		h.handleAdminLogin(w, r)
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

	if r.URL.Path == "/api/admin/logout" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
			return
		}
		h.handleAdminLogout(w, r, actor)
		return
	}
	if r.URL.Path == "/api/admin/me" {
		ipInfo := h.requestIPInfo(r)
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

func (h *APIServer) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if len(strings.TrimSpace(req.Username)) == 0 || len(strings.TrimSpace(req.Password)) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "username_and_password_required"})
		return
	}
	admin, err := h.store.AuthenticateAdminPassword(r.Context(), req.Username, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_credentials"})
		return
	}
	ipInfo := h.requestIPInfo(r)
	session, err := h.store.CreateAdminSession(r.Context(), admin.Username, ipInfo.ClientIP, ipInfo.RemoteIP, r.UserAgent(), h.cfg.AdminSessionTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "session_create_failed"})
		return
	}
	h.setSessionCookie(w, session.SessionID)
	_ = h.store.InsertAudit(r.Context(), admin.Username, "admin_login", admin.Username, "client_ip="+ipInfo.ClientIP+",remote_ip="+ipInfo.RemoteIP)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "username": admin.Username, "role": admin.Role})
}

func (h *APIServer) handleAdminLogout(w http.ResponseWriter, r *http.Request, actor requestActor) {
	if actor.SessionID != "" {
		_ = h.store.DeleteAdminSession(r.Context(), actor.SessionID)
	}
	h.clearSessionCookie(w)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
			MaxDevices int    `json:"max_devices"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if len(strings.TrimSpace(req.Password)) < h.cfg.PasswordMinLength {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password_too_short", "min_length": h.cfg.PasswordMinLength})
			return
		}
		err := h.store.CreateUser(r.Context(), User{
			Username:   strings.TrimSpace(req.Username),
			Tag:        strings.TrimSpace(req.Tag),
			ExpiresAt:  req.ExpiresAt,
			QuotaBytes: req.QuotaBytes,
			MaxDevices: req.MaxDevices,
		}, req.Password)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "create_user", req.Username, "")
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

		writeJSON(w, http.StatusOK, map[string]any{
			"user":                item,
			"devices":             devices,
			"audits":              audits,
			"trust_proxy_headers": h.cfg.TrustProxyHeaders,
			"real_ip_header":      h.cfg.RealIPHeader,
		})
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

func (h *APIServer) setDevices(w http.ResponseWriter, r *http.Request, actor requestActor, username string) {
	var req struct {
		MaxDevices int `json:"max_devices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	_ = h.store.InsertAudit(r.Context(), actor.Username, "set_user_password", username, "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if len(strings.TrimSpace(req.Password)) < h.cfg.PasswordMinLength {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password_too_short", "min_length": h.cfg.PasswordMinLength})
			return
		}
		if err := h.store.CreateAdmin(r.Context(), req.Username, req.Password, req.Role); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "create_admin", req.Username, "role="+strings.ToLower(strings.TrimSpace(req.Role)))
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		SameSite: http.SameSiteLaxMode,
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
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (h *APIServer) requestIPInfo(r *http.Request) netutil.ClientIPInfo {
	return netutil.ResolveClientIP(r, h.cfg.TrustProxyHeaders, h.cfg.RealIPHeader)
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
