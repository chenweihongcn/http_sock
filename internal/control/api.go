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
)

type requestActor struct {
	Username string
	Role     string
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
	if r.URL.Path == "/api/admin/healthz" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ts": time.Now().Unix()})
		return
	}

	actor, ok := h.authenticateAdmin(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	if !h.allowMethod(actor, r.Method) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden", "reason": "readonly_cannot_write"})
		return
	}

	if r.URL.Path == "/api/admin/me" {
		writeJSON(w, http.StatusOK, map[string]any{"username": actor.Username, "role": actor.Role})
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
	if strings.HasPrefix(r.URL.Path, "/api/admin/users/") {
		h.handleUserActions(w, r, actor)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found"})
}

func (h *APIServer) authenticateAdmin(r *http.Request) (requestActor, bool) {
	token := strings.TrimSpace(r.Header.Get("Authorization"))
	if token == "" {
		return requestActor{}, false
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	a, err := h.store.AuthenticateAdminToken(r.Context(), token)
	if err != nil {
		return requestActor{}, false
	}
	return requestActor{Username: a.Username, Role: a.Role}, true
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

func (h *APIServer) handleUsers(w http.ResponseWriter, r *http.Request, actor requestActor) {
	switch r.Method {
	case http.MethodGet:
		users, err := h.store.ListUsers(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": users})
	case http.MethodPost:
		var req struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			ExpiresAt  int64  `json:"expires_at"`
			QuotaBytes int64  `json:"quota_bytes"`
			MaxDevices int    `json:"max_devices"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		err := h.store.CreateUser(r.Context(), User{
			Username:   strings.TrimSpace(req.Username),
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

func (h *APIServer) handleUserActions(w http.ResponseWriter, r *http.Request, actor requestActor) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_path"})
		return
	}
	username := strings.TrimSpace(parts[0])

	if len(parts) == 1 && r.Method == http.MethodGet {
		u, err := h.store.GetUser(r.Context(), username)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": "user_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, u)
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
	case "extend":
		h.extend(w, r, actor, username)
	case "topup":
		h.topup(w, r, actor, username)
	case "usage":
		h.usage(w, r, actor, username)
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

func (h *APIServer) handleAudits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	q := AuditQuery{Limit: 100}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			q.Limit = n
		}
	}
	q.Actor = strings.TrimSpace(r.URL.Query().Get("actor"))
	q.Action = strings.TrimSpace(r.URL.Query().Get("action"))
	q.Target = strings.TrimSpace(r.URL.Query().Get("target"))
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			q.CreatedFrom = n
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("to")); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			q.CreatedTo = n
		}
	}

	items, err := h.store.ListAudits(r.Context(), q)
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
			Token    string `json:"token"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if err := h.store.CreateAdmin(r.Context(), req.Username, req.Token, req.Role); err != nil {
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
	case "rotate-token":
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if err := h.store.RotateAdminToken(r.Context(), username, req.Token); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		_ = h.store.InsertAudit(r.Context(), actor.Username, "rotate_admin_token", username, "")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "unknown_action"})
	}
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
