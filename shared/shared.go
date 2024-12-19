package shared

import (
	"crypto/rand"
	"log"
	"os"

	"github.com/dghubble/sessions"
)

var SessionCookieConfig *sessions.CookieConfig

const (
	SessionName     = "example-google-app"
	SessionUserKey  = "googleID"
	SessionUsername = "googleName"
	SessionEmail    = "googleEmail"
)

func NewSessionStore() sessions.Store[string] {
	randKey, err := GenerateRandomKey(32)
	if err != nil {
		log.Fatal("Error generating random key")
	}
	if os.Getenv("PROD") == "true" {
		SessionCookieConfig = sessions.DefaultCookieConfig
	} else {
		SessionCookieConfig = sessions.DebugCookieConfig
	}
	return sessions.NewCookieStore[string](SessionCookieConfig, randKey, nil)
}

// GenerateRandomKey generates a random key of specified length in bytes
func GenerateRandomKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}
