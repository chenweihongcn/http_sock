package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenHost          string
	HTTPPort            int
	SOCKS5Port          int
	AdminPort           int
	TrustProxyHeaders   bool
	RealIPHeader        string
	DialTimeout         time.Duration
	Users               map[string]string
	ControlPlaneEnabled bool
	DBPath              string
	DeviceWindow        time.Duration
	BootstrapUser       string
	BootstrapPass       string
	BootstrapReadOnly   string
	BootstrapAdminUser  string
	BootstrapAdminPass  string
	ReadOnlyAdminPass   string
	AdminSessionTTL     time.Duration
	AdminCookieSecure   bool
	PasswordMinLength   int
}

func LoadFromEnv() Config {
	cfg := Config{
		ListenHost:          getEnv("LISTEN_HOST", "0.0.0.0"),
		HTTPPort:            getEnvInt("HTTP_PORT", 8899),
		SOCKS5Port:          getEnvInt("SOCKS5_PORT", 1080),
		AdminPort:           getEnvInt("ADMIN_PORT", 8088),
		TrustProxyHeaders:   getEnvBool("TRUST_PROXY_HEADERS", false),
		RealIPHeader:        getEnv("REAL_IP_HEADER", "X-Forwarded-For"),
		DialTimeout:         getEnvDuration("DIAL_TIMEOUT", 15*time.Second),
		Users:               parseUsers(getEnv("PROXY_USERS", "admin:admin123")),
		ControlPlaneEnabled: getEnvBool("CONTROL_PLANE_ENABLED", true),
		DBPath:              getEnv("DB_PATH", "./data/proxy.db"),
		DeviceWindow:        getEnvDuration("DEVICE_WINDOW", 10*time.Minute),
		BootstrapUser:       getEnv("BOOTSTRAP_USER", "admin"),
		BootstrapPass:       getEnv("BOOTSTRAP_PASS", "admin123"),
		BootstrapReadOnly:   getEnv("BOOTSTRAP_READONLY", "ops"),
		BootstrapAdminUser:  getEnv("BOOTSTRAP_ADMIN_USER", getEnv("BOOTSTRAP_USER", "admin")),
		BootstrapAdminPass:  getEnv("BOOTSTRAP_ADMIN_PASS", getEnv("BOOTSTRAP_PASS", "admin123")),
		ReadOnlyAdminPass:   getEnv("BOOTSTRAP_READONLY_PASS", "ops123456"),
		AdminSessionTTL:     getEnvDuration("ADMIN_SESSION_TTL", 12*time.Hour),
		AdminCookieSecure:   getEnvBool("ADMIN_COOKIE_SECURE", false),
		PasswordMinLength:   getEnvInt("PASSWORD_MIN_LENGTH", 8),
	}
	return cfg
}

func (c Config) HTTPListenAddr() string {
	return c.ListenHost + ":" + strconv.Itoa(c.HTTPPort)
}

func (c Config) SOCKS5ListenAddr() string {
	return c.ListenHost + ":" + strconv.Itoa(c.SOCKS5Port)
}

func (c Config) AdminListenAddr() string {
	return c.ListenHost + ":" + strconv.Itoa(c.AdminPort)
}

func getEnv(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func getEnvInt(key string, fallback int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 || n > 65535 {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func getEnvBool(key string, fallback bool) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if val == "" {
		return fallback
	}
	if val == "1" || val == "true" || val == "yes" || val == "on" {
		return true
	}
	if val == "0" || val == "false" || val == "no" || val == "off" {
		return false
	}
	return fallback
}

func parseUsers(raw string) map[string]string {
	users := make(map[string]string)
	entries := strings.Split(raw, ",")
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		parts := strings.SplitN(e, ":", 2)
		if len(parts) != 2 {
			continue
		}
		u := strings.TrimSpace(parts[0])
		p := strings.TrimSpace(parts[1])
		if u == "" || p == "" {
			continue
		}
		users[u] = p
	}
	return users
}
