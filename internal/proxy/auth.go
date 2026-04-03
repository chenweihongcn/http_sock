package proxy

import (
	"crypto/subtle"
)

type Authenticator interface {
	Validate(username, password, sourceIP string) bool
}

type UsageRecorder interface {
	RecordUsage(username string, bytes int64)
}

type StaticAuthenticator struct {
	users map[string]string
}

func NewStaticAuthenticator(users map[string]string) *StaticAuthenticator {
	copyUsers := make(map[string]string, len(users))
	for u, p := range users {
		copyUsers[u] = p
	}
	return &StaticAuthenticator{users: copyUsers}
}

func (a *StaticAuthenticator) Validate(username, password, sourceIP string) bool {
	expected, ok := a.users[username]
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(password)) == 1
}

func (a *StaticAuthenticator) RecordUsage(username string, bytes int64) {
	_ = username
	_ = bytes
}

func (a *StaticAuthenticator) Enabled() bool {
	return len(a.users) > 0
}
