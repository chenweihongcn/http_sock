package control

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type User struct {
	Username   string `json:"username"`
	Status     int    `json:"status"`
	ExpiresAt  int64  `json:"expires_at"`
	QuotaBytes int64  `json:"quota_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
	MaxDevices int    `json:"max_devices"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	ActiveIPs  int    `json:"active_ips"`
}

type Admin struct {
	Username  string `json:"username"`
	Role      string `json:"role"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type AuditQuery struct {
	Actor     string
	Action    string
	Target    string
	CreatedTo int64
	CreatedFrom int64
	Limit     int
}

type Store struct {
	db           *sql.DB
	deviceWindow time.Duration
}

func NewStore(dbPath string, deviceWindow time.Duration) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &Store{db: db, deviceWindow: deviceWindow}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
  username TEXT PRIMARY KEY,
  password TEXT NOT NULL,
  status INTEGER NOT NULL DEFAULT 1,
  expires_at INTEGER NOT NULL DEFAULT 0,
  quota_bytes INTEGER NOT NULL DEFAULT 0,
  used_bytes INTEGER NOT NULL DEFAULT 0,
  max_devices INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS active_ips (
  username TEXT NOT NULL,
  ip TEXT NOT NULL,
  last_seen INTEGER NOT NULL,
  PRIMARY KEY(username, ip)
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  target TEXT NOT NULL,
  detail TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS admins (
  username TEXT PRIMARY KEY,
  token TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL,
  status INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_active_ips_last_seen ON active_ips(last_seen);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_admins_token ON admins(token);
`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) EnsureBootstrapAdmins(superUser, superToken, readonlyUser, readonlyToken string) error {
	now := time.Now().Unix()
	if strings.TrimSpace(superUser) != "" && strings.TrimSpace(superToken) != "" {
		if _, err := s.db.Exec(`
INSERT INTO admins(username, token, role, status, created_at, updated_at)
VALUES (?, ?, 'super', 1, ?, ?)
ON CONFLICT(username) DO UPDATE SET token = excluded.token, role = excluded.role, status = 1, updated_at = excluded.updated_at
`, strings.TrimSpace(superUser), strings.TrimSpace(superToken), now, now); err != nil {
			return err
		}
	}

	if strings.TrimSpace(readonlyUser) != "" && strings.TrimSpace(readonlyToken) != "" {
		if _, err := s.db.Exec(`
INSERT INTO admins(username, token, role, status, created_at, updated_at)
VALUES (?, ?, 'readonly', 1, ?, ?)
ON CONFLICT(username) DO UPDATE SET token = excluded.token, role = excluded.role, status = 1, updated_at = excluded.updated_at
`, strings.TrimSpace(readonlyUser), strings.TrimSpace(readonlyToken), now, now); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) AuthenticateAdminToken(ctx context.Context, token string) (Admin, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Admin{}, sql.ErrNoRows
	}

	var a Admin
	err := s.db.QueryRowContext(ctx, `
SELECT username, role, status, created_at, updated_at
FROM admins
WHERE token = ?
`, token).Scan(&a.Username, &a.Role, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Admin{}, err
	}
	if a.Status != 1 {
		return Admin{}, errors.New("admin_disabled")
	}
	return a, nil
}

func (s *Store) ListAdmins(ctx context.Context) ([]Admin, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT username, role, status, created_at, updated_at
FROM admins
ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Admin, 0)
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.Username, &a.Role, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func normalizeAdminRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "super" || role == "readonly" {
		return role, nil
	}
	return "", errors.New("invalid role")
}

func (s *Store) CreateAdmin(ctx context.Context, username, token, role string) error {
	username = strings.TrimSpace(username)
	token = strings.TrimSpace(token)
	if username == "" || token == "" {
		return errors.New("username and token are required")
	}
	normalizedRole, err := normalizeAdminRole(role)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO admins(username, token, role, status, created_at, updated_at)
VALUES (?, ?, ?, 1, ?, ?)
`, username, token, normalizedRole, now, now)
	return err
}

func (s *Store) SetAdminStatus(ctx context.Context, username string, enabled bool) error {
	status := 0
	if enabled {
		status = 1
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE admins
SET status = ?, updated_at = ?
WHERE username = ?
`, status, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) SetAdminRole(ctx context.Context, username, role string) error {
	normalizedRole, err := normalizeAdminRole(role)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE admins
SET role = ?, updated_at = ?
WHERE username = ?
`, normalizedRole, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) RotateAdminToken(ctx context.Context, username, newToken string) error {
	username = strings.TrimSpace(username)
	newToken = strings.TrimSpace(newToken)
	if username == "" || newToken == "" {
		return errors.New("username and new_token are required")
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE admins
SET token = ?, updated_at = ?
WHERE username = ?
`, newToken, time.Now().Unix(), username)
	return err
}

func (s *Store) ListAudits(ctx context.Context, q AuditQuery) ([]map[string]any, error) {
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 100
	}

	clauses := []string{"1=1"}
	args := make([]any, 0, 6)

	if strings.TrimSpace(q.Actor) != "" {
		clauses = append(clauses, "actor = ?")
		args = append(args, strings.TrimSpace(q.Actor))
	}
	if strings.TrimSpace(q.Action) != "" {
		clauses = append(clauses, "action = ?")
		args = append(args, strings.TrimSpace(q.Action))
	}
	if strings.TrimSpace(q.Target) != "" {
		clauses = append(clauses, "target = ?")
		args = append(args, strings.TrimSpace(q.Target))
	}
	if q.CreatedFrom > 0 {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, q.CreatedFrom)
	}
	if q.CreatedTo > 0 {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, q.CreatedTo)
	}

	query := `
SELECT id, actor, action, target, detail, created_at
FROM audit_logs
WHERE ` + strings.Join(clauses, " AND ") + `
ORDER BY id DESC
LIMIT ?
`
	args = append(args, q.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var actor, action, target, detail string
		var createdAt int64
		if err := rows.Scan(&id, &actor, &action, &target, &detail, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":         id,
			"actor":      actor,
			"action":     action,
			"target":     target,
			"detail":     detail,
			"created_at": createdAt,
		})
	}
	return items, rows.Err()
}

func (s *Store) EnsureBootstrapUser(username, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return nil
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(`
INSERT INTO users (username, password, status, expires_at, quota_bytes, used_bytes, max_devices, created_at, updated_at)
VALUES (?, ?, 1, 0, 0, 0, 1, ?, ?)
`, username, password, now, now)
	return err
}

func (s *Store) CreateUser(ctx context.Context, user User, password string) error {
	if strings.TrimSpace(user.Username) == "" || strings.TrimSpace(password) == "" {
		return errors.New("username and password are required")
	}
	if user.MaxDevices <= 0 {
		user.MaxDevices = 1
	}
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO users (username, password, status, expires_at, quota_bytes, used_bytes, max_devices, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, user.Username, password, 1, user.ExpiresAt, user.QuotaBytes, 0, user.MaxDevices, now, now)
	return err
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT u.username, u.status, u.expires_at, u.quota_bytes, u.used_bytes, u.max_devices, u.created_at, u.updated_at,
COALESCE((SELECT COUNT(*) FROM active_ips a WHERE a.username = u.username AND a.last_seen >= ?), 0) AS active_ips
FROM users u
ORDER BY u.created_at DESC
`, time.Now().Add(-s.deviceWindow).Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.Username, &u.Status, &u.ExpiresAt, &u.QuotaBytes, &u.UsedBytes, &u.MaxDevices, &u.CreatedAt, &u.UpdatedAt, &u.ActiveIPs); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) GetUser(ctx context.Context, username string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
SELECT u.username, u.status, u.expires_at, u.quota_bytes, u.used_bytes, u.max_devices, u.created_at, u.updated_at,
COALESCE((SELECT COUNT(*) FROM active_ips a WHERE a.username = u.username AND a.last_seen >= ?), 0) AS active_ips
FROM users u
WHERE u.username = ?
`, time.Now().Add(-s.deviceWindow).Unix(), username).
		Scan(&u.Username, &u.Status, &u.ExpiresAt, &u.QuotaBytes, &u.UsedBytes, &u.MaxDevices, &u.CreatedAt, &u.UpdatedAt, &u.ActiveIPs)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) SetUserStatus(ctx context.Context, username string, enabled bool) error {
	status := 0
	if enabled {
		status = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET status = ?, updated_at = ? WHERE username = ?`, status, time.Now().Unix(), username)
	return err
}

func (s *Store) ExtendUserDays(ctx context.Context, username string, days int) error {
	if days <= 0 {
		return errors.New("days must be > 0")
	}
	now := time.Now().Unix()
	var expires int64
	if err := s.db.QueryRowContext(ctx, `SELECT expires_at FROM users WHERE username = ?`, username).Scan(&expires); err != nil {
		return err
	}
	base := expires
	if base < now {
		base = now
	}
	next := base + int64(days)*24*3600
	_, err := s.db.ExecContext(ctx, `UPDATE users SET expires_at = ?, updated_at = ? WHERE username = ?`, next, now, username)
	return err
}

func (s *Store) TopUpQuota(ctx context.Context, username string, bytes int64) error {
	if bytes <= 0 {
		return errors.New("bytes must be > 0")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET quota_bytes = quota_bytes + ?, updated_at = ? WHERE username = ?`, bytes, time.Now().Unix(), username)
	return err
}

func (s *Store) AddUsage(ctx context.Context, username string, bytes int64) error {
	if bytes <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET used_bytes = used_bytes + ?, updated_at = ? WHERE username = ?`, bytes, time.Now().Unix(), username)
	return err
}

func (s *Store) InsertAudit(ctx context.Context, actor, action, target, detail string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO audit_logs(actor, action, target, detail, created_at)
VALUES (?, ?, ?, ?, ?)
`, actor, action, target, detail, time.Now().Unix())
	return err
}

func (s *Store) Authorize(username, password, sourceIP string) (bool, string, error) {
	now := time.Now().Unix()
	cutoff := time.Now().Add(-s.deviceWindow).Unix()

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false, "internal_error", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`DELETE FROM active_ips WHERE last_seen < ?`, cutoff); err != nil {
		return false, "internal_error", err
	}

	var storedPassword string
	var status int
	var expiresAt int64
	var quotaBytes int64
	var usedBytes int64
	var maxDevices int
	if err := tx.QueryRow(`
SELECT password, status, expires_at, quota_bytes, used_bytes, max_devices
FROM users
WHERE username = ?
`, username).Scan(&storedPassword, &status, &expiresAt, &quotaBytes, &usedBytes, &maxDevices); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "user_not_found", nil
		}
		return false, "internal_error", err
	}

	if subtle.ConstantTimeCompare([]byte(storedPassword), []byte(password)) != 1 {
		return false, "invalid_password", nil
	}
	if status != 1 {
		return false, "user_disabled", nil
	}
	if expiresAt > 0 && now > expiresAt {
		return false, "expired", nil
	}
	if quotaBytes > 0 && usedBytes >= quotaBytes {
		return false, "quota_exceeded", nil
	}
	if maxDevices <= 0 {
		maxDevices = 1
	}

	var existing int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM active_ips WHERE username = ? AND ip = ?`, username, sourceIP).Scan(&existing); err != nil {
		return false, "internal_error", err
	}
	if existing == 0 {
		var cnt int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM active_ips WHERE username = ?`, username).Scan(&cnt); err != nil {
			return false, "internal_error", err
		}
		if cnt >= maxDevices {
			return false, "device_limit", nil
		}
	}

	if _, err := tx.Exec(`
INSERT INTO active_ips(username, ip, last_seen)
VALUES (?, ?, ?)
ON CONFLICT(username, ip) DO UPDATE SET last_seen = excluded.last_seen
`, username, sourceIP, now); err != nil {
		return false, "internal_error", err
	}

	if err := tx.Commit(); err != nil {
		return false, "internal_error", err
	}
	return true, "ok", nil
}

func (s *Store) MustGetUser(ctx context.Context, username string) (User, error) {
	u, err := s.GetUser(ctx, username)
	if err != nil {
		return User{}, fmt.Errorf("get user %s: %w", username, err)
	}
	return u, nil
}
