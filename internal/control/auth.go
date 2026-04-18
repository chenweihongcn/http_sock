package control

import (
	"context"
	"log"
)

type Authenticator struct {
	store *Store
}

func NewAuthenticator(store *Store) *Authenticator {
	return &Authenticator{store: store}
}

func (a *Authenticator) Validate(username, password, sourceIP, clientAgent string) bool {
	ok, reason, err := a.store.Authorize(username, password, sourceIP, clientAgent)
	if err != nil {
		log.Printf("authorize error user=%s ip=%s err=%v", username, sourceIP, err)
		return false
	}
	if !ok {
		log.Printf("authorize denied user=%s ip=%s reason=%s", username, sourceIP, reason)
	}
	return ok
}

func (a *Authenticator) RecordUsage(username string, bytes int64) {
	if bytes <= 0 {
		return
	}
	if err := a.store.AddUsage(context.Background(), username, bytes); err != nil {
		log.Printf("record usage error user=%s bytes=%d err=%v", username, bytes, err)
	}
}

func (a *Authenticator) SpeedLimitBytesPerSec(username string) int64 {
	return a.store.GetUserSpeedLimitBytesPerSec(context.Background(), username)
}
