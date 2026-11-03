package push

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push/webpush"
)

// Environment variables holding the VAPID identity.
const (
	EnvPublicKey  = "LANKAWEATHER_VAPID_PUBLIC_KEY"
	EnvPrivateKey = "LANKAWEATHER_VAPID_PRIVATE_KEY"
	EnvSubject    = "LANKAWEATHER_VAPID_SUBJECT"
)

// ErrNotConfigured means the VAPID variables are not all set.
var ErrNotConfigured = errors.New("LANKAWEATHER_VAPID_PUBLIC_KEY, LANKAWEATHER_VAPID_PRIVATE_KEY and LANKAWEATHER_VAPID_SUBJECT are not all set")

// VAPIDFromEnv reads the identity from the environment. With none of the
// variables set it returns ErrNotConfigured: push is off. A partial or
// invalid set is an error of its own, so a typo is not mistaken for "off".
func VAPIDFromEnv() (*webpush.VAPID, error) {
	public, private, subject := os.Getenv(EnvPublicKey), os.Getenv(EnvPrivateKey), os.Getenv(EnvSubject)
	if public == "" && private == "" && subject == "" {
		return nil, ErrNotConfigured
	}
	if public == "" || private == "" || subject == "" {
		return nil, errors.New("the VAPID variables are only partly set: set all three or none")
	}
	return webpush.NewVAPID(public, private, subject)
}

// NewClient is the push client used by the API and the worker.
func NewClient(vapid *webpush.VAPID) *webpush.Client {
	return &webpush.Client{VAPID: vapid, HTTP: &http.Client{Timeout: 15 * time.Second}}
}
