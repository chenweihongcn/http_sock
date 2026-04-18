package control

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username   string `json:"username"`
	Tag        string `json:"tag"`
	Status     int    `json:"status"`
	ExpiresAt  int64  `json:"expires_at"`
	QuotaBytes int64  `json:"quota_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
	SMBEnabled   int   `json:"smb_enabled"`
	SMBQuotaBytes int64 `json:"smb_quota_bytes"`
	SMBUsedBytes  int64 `json:"smb_used_bytes"`
	SpeedLimitKbps int64 `json:"speed_limit_kbps"`
	MaxDevices int     `json:"max_devices"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	ActiveIPs  int    `json:"active_ips"`
}

type UserDevice struct {
	IP        string `json:"ip"`
	FirstSeen int64  `json:"first_seen"`
	LastSeen  int64  `json:"last_seen"`
	UserAgent string `json:"user_agent"`
}

type UserListQuery struct {
	Search       string
	Status       *int
	Tag          string
	ExpireFilter string
	Offset       int
	Limit        int
}

type UserListResult struct {
	Items  []User `json:"items"`
	Total  int    `json:"total"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type Admin struct {
	Username    string `json:"username"`
	Role        string `json:"role"`
	Status      int    `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	PasswordSet bool   `json:"password_set"`
}

type AdminSession struct {
	SessionID    string `json:"session_id"`
	Username     string `json:"username"`
	CreatedAt    int64  `json:"created_at"`
	LastActivity int64  `json:"last_activity"`
	ExpiresAt    int64  `json:"expires_at"`
	IPAddress    string `json:"ip_address"`
	OriginalIP   string `json:"original_ip"`
	UserAgent    string `json:"user_agent"`
}

type AuditQuery struct {
	Actor       string
	Action      string
	Target      string
	CreatedTo   int64
	CreatedFrom int64
	Limit       int
}

type adminCredentials struct {
	Admin
	PasswordHash string
}

type Store struct {
	db           *sql.DB
	deviceWindow time.Duration
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`)

func normalizeUsername(raw string) (string, error) {
	username := strings.TrimSpace(raw)
	if username == "" {
		return "", errors.New("username_required")
	}
	if !usernamePattern.MatchString(username) {
		return "", errors.New("invalid_username_format")
	}
	return username, nil
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
	smb_enabled INTEGER NOT NULL DEFAULT 0,
	smb_quota_bytes INTEGER NOT NULL DEFAULT 0,
	smb_used_bytes INTEGER NOT NULL DEFAULT 0,
	speed_limit_kbps INTEGER NOT NULL DEFAULT 0,
  max_devices INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS active_ips (
  username TEXT NOT NULL,
  ip TEXT NOT NULL,
	user_agent TEXT NOT NULL DEFAULT '',
	first_seen INTEGER NOT NULL DEFAULT 0,
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
  password_hash TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL,
  status INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS admin_sessions (
  session_id TEXT PRIMARY KEY,
  admin_username TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  last_activity INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  ip_address TEXT NOT NULL DEFAULT '',
	original_ip_address TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_active_ips_last_seen ON active_ips(last_seen);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_admins_token ON admins(token);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_username ON admin_sessions(admin_username);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires_at ON admin_sessions(expires_at);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	if err := s.ensureColumn("admins", "password_hash", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("users", "tag", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("users", "smb_enabled", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("users", "smb_quota_bytes", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("users", "smb_used_bytes", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("users", "speed_limit_kbps", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("active_ips", "user_agent", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("active_ips", "first_seen", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if _, err := s.db.Exec(`UPDATE active_ips SET first_seen = last_seen WHERE first_seen = 0`); err != nil {
		return err
	}
	if err := s.ensureColumn("admin_sessions", "original_ip_address", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureColumn(tableName, columnName, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if strings.EqualFold(name, columnName) {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE ` + tableName + ` ADD COLUMN ` + columnName + ` ` + definition)
	return err
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") || strings.HasPrefix(value, "$2b$") || strings.HasPrefix(value, "$2y$")
}

func verifyStoredPassword(stored, raw string) (bool, bool) {
	if stored == "" || raw == "" {
		return false, false
	}
	if isBcryptHash(stored) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(raw)) == nil, false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(raw)) == 1, true
}

func randomSecret(size int) string {
	if size <= 0 {
		size = 32
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func disabledToken() string {
	return "disabled-" + randomSecret(18)
}

func normalizeAdminRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "super" || role == "readonly" {
		return role, nil
	}
	return "", errors.New("invalid role")
}

func (s *Store) EnsureBootstrapAdmins(superUser, superPassword, readonlyUser, readonlyPassword string) error {
	now := time.Now().Unix()
	if strings.TrimSpace(superUser) != "" && strings.TrimSpace(superPassword) != "" {
		normalizedSuperUser, err := normalizeUsername(superUser)
		if err != nil {
			return fmt.Errorf("invalid_bootstrap_admin_user: %w", err)
		}
		hash, err := hashPassword(strings.TrimSpace(superPassword))
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(`
INSERT INTO admins(username, token, password_hash, role, status, created_at, updated_at)
VALUES (?, ?, ?, 'super', 1, ?, ?)
ON CONFLICT(username) DO UPDATE SET password_hash = excluded.password_hash, role = excluded.role, status = 1, updated_at = excluded.updated_at
`, normalizedSuperUser, disabledToken(), hash, now, now); err != nil {
			return err
		}
	}

	if strings.TrimSpace(readonlyUser) != "" && strings.TrimSpace(readonlyPassword) != "" {
		normalizedReadonlyUser, err := normalizeUsername(readonlyUser)
		if err != nil {
			return fmt.Errorf("invalid_bootstrap_readonly_user: %w", err)
		}
		hash, err := hashPassword(strings.TrimSpace(readonlyPassword))
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(`
INSERT INTO admins(username, token, password_hash, role, status, created_at, updated_at)
VALUES (?, ?, ?, 'readonly', 1, ?, ?)
ON CONFLICT(username) DO UPDATE SET password_hash = excluded.password_hash, role = excluded.role, status = 1, updated_at = excluded.updated_at
`, normalizedReadonlyUser, disabledToken(), hash, now, now); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) getAdminCredentials(ctx context.Context, username string) (adminCredentials, error) {
	var item adminCredentials
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `
SELECT username, role, status, created_at, updated_at, password_hash
FROM admins
WHERE username = ?
`, strings.TrimSpace(username)).Scan(&item.Username, &item.Role, &item.Status, &item.CreatedAt, &item.UpdatedAt, &passwordHash)
	if err != nil {
		return adminCredentials{}, err
	}
	item.PasswordHash = passwordHash
	item.PasswordSet = passwordHash != ""
	return item, nil
}

func (s *Store) AuthenticateAdminPassword(ctx context.Context, username, password string) (Admin, error) {
	item, err := s.getAdminCredentials(ctx, username)
	if err != nil {
		return Admin{}, err
	}
	if item.Status != 1 {
		return Admin{}, errors.New("admin_disabled")
	}
	ok, _ := verifyStoredPassword(item.PasswordHash, password)
	if !ok {
		return Admin{}, errors.New("invalid_credentials")
	}
	return item.Admin, nil
}

func (s *Store) ListAdmins(ctx context.Context) ([]Admin, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT username, role, status, created_at, updated_at,
CASE WHEN password_hash <> '' THEN 1 ELSE 0 END AS password_set
FROM admins
ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Admin, 0)
	for rows.Next() {
		var item Admin
		var passwordSet int
		if err := rows.Scan(&item.Username, &item.Role, &item.Status, &item.CreatedAt, &item.UpdatedAt, &passwordSet); err != nil {
			return nil, err
		}
		item.PasswordSet = passwordSet == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateAdmin(ctx context.Context, username, password, role string) error {
	var err error
	username, err = normalizeUsername(username)
	if err != nil {
		return err
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return errors.New("username and password are required")
	}
	normalizedRole, err := normalizeAdminRole(role)
	if err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO admins(username, token, password_hash, role, status, created_at, updated_at)
VALUES (?, ?, ?, ?, 1, ?, ?)
`, username, disabledToken(), hash, normalizedRole, now, now)
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

func (s *Store) SetAdminPassword(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE admins
SET password_hash = ?, updated_at = ?
WHERE username = ?
`, hash, time.Now().Unix(), username)
	return err
}

func (s *Store) ChangeAdminPassword(ctx context.Context, username, oldPassword, newPassword string) error {
	item, err := s.getAdminCredentials(ctx, username)
	if err != nil {
		return err
	}
	ok, _ := verifyStoredPassword(item.PasswordHash, strings.TrimSpace(oldPassword))
	if !ok {
		return errors.New("invalid_old_password")
	}
	return s.SetAdminPassword(ctx, username, newPassword)
}

func (s *Store) cleanupExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE expires_at < ?`, time.Now().Unix())
	return err
}

func (s *Store) CreateAdminSession(ctx context.Context, username, ipAddress, originalIP, userAgent string, ttl time.Duration) (AdminSession, error) {
	if err := s.cleanupExpiredSessions(ctx); err != nil {
		return AdminSession{}, err
	}
	now := time.Now().Unix()
	expiresAt := now + int64(ttl.Seconds())
	item := AdminSession{
		SessionID:    randomSecret(32),
		Username:     strings.TrimSpace(username),
		CreatedAt:    now,
		LastActivity: now,
		ExpiresAt:    expiresAt,
		IPAddress:    strings.TrimSpace(ipAddress),
		OriginalIP:   strings.TrimSpace(originalIP),
		UserAgent:    strings.TrimSpace(userAgent),
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO admin_sessions(session_id, admin_username, created_at, last_activity, expires_at, ip_address, original_ip_address, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, item.SessionID, item.Username, item.CreatedAt, item.LastActivity, item.ExpiresAt, item.IPAddress, item.OriginalIP, item.UserAgent)
	if err != nil {
		return AdminSession{}, err
	}
	return item, nil
}

func (s *Store) ValidateAdminSession(ctx context.Context, sessionID string, ttl time.Duration) (Admin, error) {
	if err := s.cleanupExpiredSessions(ctx); err != nil {
		return Admin{}, err
	}
	var item Admin
	var passwordSet int
	err := s.db.QueryRowContext(ctx, `
SELECT a.username, a.role, a.status, a.created_at, a.updated_at,
CASE WHEN a.password_hash <> '' THEN 1 ELSE 0 END AS password_set
FROM admin_sessions s
JOIN admins a ON a.username = s.admin_username
WHERE s.session_id = ? AND s.expires_at >= ?
`, strings.TrimSpace(sessionID), time.Now().Unix()).Scan(&item.Username, &item.Role, &item.Status, &item.CreatedAt, &item.UpdatedAt, &passwordSet)
	if err != nil {
		return Admin{}, err
	}
	if item.Status != 1 {
		return Admin{}, errors.New("admin_disabled")
	}
	item.PasswordSet = passwordSet == 1
	now := time.Now().Unix()
	_, err = s.db.ExecContext(ctx, `
UPDATE admin_sessions
SET last_activity = ?, expires_at = ?
WHERE session_id = ?
`, now, now+int64(ttl.Seconds()), strings.TrimSpace(sessionID))
	if err != nil {
		return Admin{}, err
	}
	return item, nil
}

func (s *Store) DeleteAdminSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE session_id = ?`, strings.TrimSpace(sessionID))
	return err
}

func (s *Store) ListAdminSessions(ctx context.Context, username string) ([]AdminSession, error) {
	if err := s.cleanupExpiredSessions(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT session_id, admin_username, created_at, last_activity, expires_at, ip_address, original_ip_address, user_agent
FROM admin_sessions
WHERE admin_username = ?
ORDER BY last_activity DESC
`, strings.TrimSpace(username))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminSession, 0)
	for rows.Next() {
		var item AdminSession
		if err := rows.Scan(&item.SessionID, &item.Username, &item.CreatedAt, &item.LastActivity, &item.ExpiresAt, &item.IPAddress, &item.OriginalIP, &item.UserAgent); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
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
	var err error
	username, err = normalizeUsername(username)
	if err != nil {
		return fmt.Errorf("invalid_bootstrap_user: %w", err)
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return nil
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = s.db.Exec(`
INSERT INTO users (username, password, status, expires_at, quota_bytes, used_bytes, smb_enabled, smb_quota_bytes, smb_used_bytes, speed_limit_kbps, max_devices, created_at, updated_at)
VALUES (?, ?, 1, 0, 0, 0, 0, 0, 0, 0, 1, ?, ?)
`, username, hash, now, now)
	return err
}

func (s *Store) CreateUser(ctx context.Context, user User, password string) error {
	username, err := normalizeUsername(user.Username)
	if err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("username and password are required")
	}
	if user.MaxDevices <= 0 {
		user.MaxDevices = 1
	}
	hash, err := hashPassword(strings.TrimSpace(password))
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO users (username, password, status, expires_at, quota_bytes, used_bytes, smb_enabled, smb_quota_bytes, smb_used_bytes, speed_limit_kbps, max_devices, tag, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, username, hash, 1, user.ExpiresAt, user.QuotaBytes, 0, user.SMBEnabled, user.SMBQuotaBytes, 0, user.SpeedLimitKbps, user.MaxDevices, strings.TrimSpace(user.Tag), now, now)
	return err
}

func (s *Store) ListUsers(ctx context.Context, q UserListQuery) (UserListResult, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	clauses := []string{"1=1"}
	args := make([]any, 0, 4)
	if search := strings.TrimSpace(q.Search); search != "" {
		clauses = append(clauses, "u.username LIKE ?")
		args = append(args, "%"+search+"%")
	}
	if q.Status != nil {
		clauses = append(clauses, "u.status = ?")
		args = append(args, *q.Status)
	}
	if tag := strings.TrimSpace(q.Tag); tag != "" {
		clauses = append(clauses, "u.tag = ?")
		args = append(args, tag)
	}
	now := time.Now().Unix()
	switch strings.TrimSpace(q.ExpireFilter) {
	case "expired":
		clauses = append(clauses, "u.expires_at > 0 AND u.expires_at < ?")
		args = append(args, now)
	case "expiring7":
		clauses = append(clauses, "u.expires_at > 0 AND u.expires_at >= ? AND u.expires_at <= ?")
		args = append(args, now, now+7*24*3600)
	case "permanent":
		clauses = append(clauses, "u.expires_at = 0")
	}

	where := strings.Join(clauses, " AND ")
	var total int
	countQuery := `SELECT COUNT(*) FROM users u WHERE ` + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return UserListResult{}, err
	}

	cutoff := time.Now().Add(-s.deviceWindow).Unix()
	listArgs := make([]any, 0, len(args)+3)
	listArgs = append(listArgs, cutoff)
	listArgs = append(listArgs, args...)
	listArgs = append(listArgs, q.Limit, q.Offset)
	rows, err := s.db.QueryContext(ctx, `
SELECT u.username, u.status, u.expires_at, u.quota_bytes, u.used_bytes, u.smb_enabled, u.smb_quota_bytes, u.smb_used_bytes, u.speed_limit_kbps, u.max_devices, u.created_at, u.updated_at,
u.tag,
COALESCE((SELECT COUNT(*) FROM active_ips a WHERE a.username = u.username AND a.last_seen >= ?), 0) AS active_ips
FROM users u
WHERE `+where+`
ORDER BY u.created_at DESC
LIMIT ? OFFSET ?
`, listArgs...)
	if err != nil {
		return UserListResult{}, err
	}
	defer rows.Close()

	items := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.Username, &user.Status, &user.ExpiresAt, &user.QuotaBytes, &user.UsedBytes, &user.SMBEnabled, &user.SMBQuotaBytes, &user.SMBUsedBytes, &user.SpeedLimitKbps, &user.MaxDevices, &user.CreatedAt, &user.UpdatedAt, &user.Tag, &user.ActiveIPs); err != nil {
			return UserListResult{}, err
		}
		items = append(items, user)
	}
	if err := rows.Err(); err != nil {
		return UserListResult{}, err
	}

	return UserListResult{Items: items, Total: total, Offset: q.Offset, Limit: q.Limit}, nil
}

func (s *Store) GetUser(ctx context.Context, username string) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, `
SELECT u.username, u.status, u.expires_at, u.quota_bytes, u.used_bytes, u.smb_enabled, u.smb_quota_bytes, u.smb_used_bytes, u.speed_limit_kbps, u.max_devices, u.created_at, u.updated_at,
u.tag,
COALESCE((SELECT COUNT(*) FROM active_ips a WHERE a.username = u.username AND a.last_seen >= ?), 0) AS active_ips
FROM users u
WHERE u.username = ?
`, time.Now().Add(-s.deviceWindow).Unix(), strings.TrimSpace(username)).
		Scan(&user.Username, &user.Status, &user.ExpiresAt, &user.QuotaBytes, &user.UsedBytes, &user.SMBEnabled, &user.SMBQuotaBytes, &user.SMBUsedBytes, &user.SpeedLimitKbps, &user.MaxDevices, &user.CreatedAt, &user.UpdatedAt, &user.Tag, &user.ActiveIPs)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) DeleteUser(ctx context.Context, username string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	username = strings.TrimSpace(username)
	if _, err := tx.ExecContext(ctx, `DELETE FROM active_ips WHERE username = ?`, username); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (s *Store) SetUserTag(ctx context.Context, username, tag string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username_required")
	}
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx, `UPDATE users SET tag = ?, updated_at = ? WHERE username = ?`, strings.TrimSpace(tag), now, username)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ListUserTags(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT tag FROM users WHERE TRIM(tag) <> '' ORDER BY tag ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (s *Store) SetUserStatus(ctx context.Context, username string, enabled bool) error {
	status := 0
	if enabled {
		status = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET status = ?, updated_at = ? WHERE username = ?`, status, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) BatchSetUserStatus(ctx context.Context, usernames []string, enabled bool) (int64, error) {
	if len(usernames) == 0 {
		return 0, nil
	}
	status := 0
	if enabled {
		status = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `UPDATE users SET status = ?, updated_at = ? WHERE username = ?`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var total int64
	now := time.Now().Unix()
	for _, username := range usernames {
		result, err := stmt.ExecContext(ctx, status, now, strings.TrimSpace(username))
		if err != nil {
			return 0, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		total += affected
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Store) SetUserMaxDevices(ctx context.Context, username string, maxDevices int) error {
	if maxDevices <= 0 {
		return errors.New("max_devices must be > 0")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET max_devices = ?, updated_at = ? WHERE username = ?`, maxDevices, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) SetUserPassword(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE users SET password = ?, updated_at = ? WHERE username = ?`, hash, time.Now().Unix(), username)
	return err
}

func (s *Store) ResetUserUsage(ctx context.Context, username string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET used_bytes = 0, updated_at = ? WHERE username = ?`, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) ExtendUserDays(ctx context.Context, username string, days int) error {
	if days <= 0 {
		return errors.New("days must be > 0")
	}
	now := time.Now().Unix()
	var expires int64
	if err := s.db.QueryRowContext(ctx, `SELECT expires_at FROM users WHERE username = ?`, strings.TrimSpace(username)).Scan(&expires); err != nil {
		return err
	}
	base := expires
	if base < now {
		base = now
	}
	next := base + int64(days)*24*3600
	_, err := s.db.ExecContext(ctx, `UPDATE users SET expires_at = ?, updated_at = ? WHERE username = ?`, next, now, strings.TrimSpace(username))
	return err
}

func (s *Store) TopUpQuota(ctx context.Context, username string, bytes int64) error {
	if bytes <= 0 {
		return errors.New("bytes must be > 0")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET quota_bytes = quota_bytes + ?, updated_at = ? WHERE username = ?`, bytes, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) TopUpSMBQuota(ctx context.Context, username string, bytes int64) error {
	if bytes <= 0 {
		return errors.New("bytes must be > 0")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE users SET smb_quota_bytes = smb_quota_bytes + ?, updated_at = ? WHERE username = ? AND smb_enabled = 1`, bytes, time.Now().Unix(), strings.TrimSpace(username))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("smb_not_enabled_or_user_not_found")
	}
	return nil
}

func (s *Store) SetUserQuota(ctx context.Context, username string, bytes int64) error {
	if bytes < 0 {
		return errors.New("bytes must be >= 0")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET quota_bytes = ?, updated_at = ? WHERE username = ?`, bytes, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) SetUserSMBQuota(ctx context.Context, username string, bytes int64) error {
	if bytes < 0 {
		return errors.New("bytes must be >= 0")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET smb_quota_bytes = ?, updated_at = ? WHERE username = ?`, bytes, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) SetUserSpeedLimitKbps(ctx context.Context, username string, kbps int64) error {
	if kbps < 0 {
		return errors.New("kbps must be >= 0")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET speed_limit_kbps = ?, updated_at = ? WHERE username = ?`, kbps, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) GetUserSpeedLimitBytesPerSec(ctx context.Context, username string) int64 {
	var kbps int64
	err := s.db.QueryRowContext(ctx, `SELECT speed_limit_kbps FROM users WHERE username = ?`, strings.TrimSpace(username)).Scan(&kbps)
	if err != nil || kbps <= 0 {
		return 0
	}
	return kbps * 1024
}

func (s *Store) AuthenticateUserPassword(ctx context.Context, username, password string) (User, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return User{}, errors.New("invalid_credentials")
	}

	var storedPassword string
	var user User
	err := s.db.QueryRowContext(ctx, `
SELECT password, username, tag, status, expires_at, quota_bytes, used_bytes, smb_enabled, smb_quota_bytes, smb_used_bytes, max_devices, created_at, updated_at
FROM users
WHERE username = ?
`, username).Scan(&storedPassword, &user.Username, &user.Tag, &user.Status, &user.ExpiresAt, &user.QuotaBytes, &user.UsedBytes, &user.SMBEnabled, &user.SMBQuotaBytes, &user.SMBUsedBytes, &user.MaxDevices, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, errors.New("invalid_credentials")
		}
		return User{}, err
	}

	ok, needsUpgrade := verifyStoredPassword(storedPassword, password)
	if !ok {
		return User{}, errors.New("invalid_credentials")
	}
	if needsUpgrade {
		hash, err := hashPassword(password)
		if err == nil {
			_, _ = s.db.ExecContext(ctx, `UPDATE users SET password = ?, updated_at = ? WHERE username = ?`, hash, time.Now().Unix(), username)
		}
	}

	now := time.Now().Unix()
	if user.Status != 1 {
		return User{}, errors.New("user_disabled")
	}
	if user.ExpiresAt > 0 && now > user.ExpiresAt {
		return User{}, errors.New("expired")
	}
	return user, nil
}

func (s *Store) SetUserSMBEnabled(ctx context.Context, username string, enabled bool) error {
	value := 0
	if enabled {
		value = 1
	}
	result, err := s.db.ExecContext(ctx, `UPDATE users SET smb_enabled = ?, updated_at = ? WHERE username = ?`, value, time.Now().Unix(), strings.TrimSpace(username))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) AddUsage(ctx context.Context, username string, bytes int64) error {
	if bytes <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET used_bytes = used_bytes + ?, updated_at = ? WHERE username = ?`, bytes, time.Now().Unix(), strings.TrimSpace(username))
	return err
}

func (s *Store) ListUserDevices(ctx context.Context, username string) ([]UserDevice, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT ip, first_seen, last_seen, user_agent
FROM active_ips
WHERE username = ? AND last_seen >= ?
ORDER BY last_seen DESC
`, strings.TrimSpace(username), time.Now().Add(-s.deviceWindow).Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]UserDevice, 0)
	for rows.Next() {
		var item UserDevice
		if err := rows.Scan(&item.IP, &item.FirstSeen, &item.LastSeen, &item.UserAgent); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteUserDevice(ctx context.Context, username, ip string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM active_ips WHERE username = ? AND ip = ?`, strings.TrimSpace(username), strings.TrimSpace(ip))
	return err
}

func (s *Store) InsertAudit(ctx context.Context, actor, action, target, detail string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO audit_logs(actor, action, target, detail, created_at)
VALUES (?, ?, ?, ?, ?)
`, actor, action, target, detail, time.Now().Unix())
	return err
}

func (s *Store) Authorize(username, password, sourceIP, clientAgent string) (bool, string, error) {
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
`, strings.TrimSpace(username)).Scan(&storedPassword, &status, &expiresAt, &quotaBytes, &usedBytes, &maxDevices); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "user_not_found", nil
		}
		return false, "internal_error", err
	}

	ok, needsUpgrade := verifyStoredPassword(storedPassword, password)
	if !ok {
		return false, "invalid_password", nil
	}
	if needsUpgrade {
		hash, err := hashPassword(password)
		if err != nil {
			return false, "internal_error", err
		}
		if _, err := tx.Exec(`UPDATE users SET password = ?, updated_at = ? WHERE username = ?`, hash, now, strings.TrimSpace(username)); err != nil {
			return false, "internal_error", err
		}
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
	if err := tx.QueryRow(`SELECT COUNT(*) FROM active_ips WHERE username = ? AND ip = ?`, strings.TrimSpace(username), strings.TrimSpace(sourceIP)).Scan(&existing); err != nil {
		return false, "internal_error", err
	}
	if existing == 0 {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM active_ips WHERE username = ?`, strings.TrimSpace(username)).Scan(&count); err != nil {
			return false, "internal_error", err
		}
		if count >= maxDevices {
			return false, "device_limit", nil
		}
	}

	if _, err := tx.Exec(`
INSERT INTO active_ips(username, ip, user_agent, first_seen, last_seen)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(username, ip) DO UPDATE SET
	user_agent = excluded.user_agent,
	first_seen = CASE WHEN active_ips.first_seen > 0 THEN active_ips.first_seen ELSE excluded.first_seen END,
	last_seen = excluded.last_seen
`, strings.TrimSpace(username), strings.TrimSpace(sourceIP), strings.TrimSpace(clientAgent), now, now); err != nil {
		return false, "internal_error", err
	}

	if err := tx.Commit(); err != nil {
		return false, "internal_error", err
	}
	return true, "ok", nil
}

func (s *Store) MustGetUser(ctx context.Context, username string) (User, error) {
	user, err := s.GetUser(ctx, username)
	if err != nil {
		return User{}, fmt.Errorf("get user %s: %w", username, err)
	}
	return user, nil
}
