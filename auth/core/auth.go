package core

import (
	"net/http"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/stats"
)

type AuthData struct {
	ExpectedAAD string
	SessionKey  string
}

type AuthRequest struct {
	API     string
	Address string
	AAD     []byte
	PK      []byte
	Data    string
}

type Auth interface {
	Name() string
	Stats() stats.Metrics

	// Authenticate the user from the http request. This should return:
	// - the expected AAD
	// - the authentication error
	Authenticate(req *http.Request) (string, error)

	// Verify that a specific payload complies with
	// the expected format and has the authentication data required
	Verify(req AuthRequest, expected string) error

	// Sets the logger for the authentication plugin.
	SetLogger(log.Logger)
}

type NilAuth struct{}

func (NilAuth) Name() string {
	return "auth.nil"
}
func (NilAuth) Stats() stats.Metrics {
	return nil
}
func (NilAuth) Authenticate(req *http.Request) (string, error) {
	return "", nil
}
func (NilAuth) Verify(req AuthRequest, expected string) error {
	return nil
}

func (NilAuth) SetLogger(log.Logger) {
	return
}
