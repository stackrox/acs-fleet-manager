package impl

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// ServiceAccountTokenAuthName is the name of the token file auth authentication method.
	ServiceAccountTokenAuthName = "SERVICE_ACCOUNT_TOKEN"
)

var (
	_                          authFactory = (*serviceAccountTokenAuthFactory)(nil)
	_                          Auth        = (*serviceAccountTokenAuth)(nil)
	serviceAccountTokenFactory             = &serviceAccountTokenAuthFactory{}
)

type serviceAccountTokenAuthFactory struct{}

type serviceAccountTokenAuth struct {
	lock   sync.Mutex
	token  *token
	file   string
	leeway time.Duration
	period time.Duration
	now    func() time.Time // used for testing
}

type token struct {
	raw    string
	expiry time.Time
}

// GetName gets the name of the factory.
func (f *serviceAccountTokenAuthFactory) GetName() string {
	return ServiceAccountTokenAuthName
}

// CreateAuth creates a new instance of Auth or returns error.
func (f *serviceAccountTokenAuthFactory) CreateAuth(_ context.Context, o Option) (Auth, error) {
	return &serviceAccountTokenAuth{
		file:   o.ServiceAccount.TokenFile,
		leeway: 5 * time.Second,
		// The token is renewed at 80% of the validity period. The validity period is assumed to be one hour.
		// Therefore, the selected period is equal to half the time between the token renewal and expiration.
		// This ensures that the token will be reloaded after the renewal.
		period: 6 * time.Minute,
		now:    time.Now,
	}, nil
}

// AddAuth add auth token to the request retrieved from the filesystem.
func (a *serviceAccountTokenAuth) AddAuth(req *http.Request) error {
	token, err := a.getOrLoadToken()
	if err != nil {
		return fmt.Errorf("retrieve service account token: %w", err)
	}

	setBearer(req, token)
	return nil
}

func (a *serviceAccountTokenAuth) getOrLoadToken() (string, error) {
	now := a.now()

	a.lock.Lock()
	defer a.lock.Unlock()

	t := a.token

	if t != nil && t.expiry.Add(-1*a.leeway).After(now) {
		return t.raw, nil
	}

	t, err := a.loadToken()
	if err != nil {
		return "", fmt.Errorf("load token from file %q: %w", a.file, err)
	}

	a.token = t
	return t.raw, nil
}

func (a *serviceAccountTokenAuth) loadToken() (*token, error) {
	tokenBytes, err := os.ReadFile(a.file)
	if err != nil {
		return nil, fmt.Errorf("reading token file %q: %w", a.file, err)
	}
	tokenString := strings.TrimSpace(string(tokenBytes))
	if len(tokenString) == 0 {
		return nil, fmt.Errorf("empty token file %q", a.file)
	}
	return &token{
		raw:    tokenString,
		expiry: a.now().Add(a.period),
	}, nil
}
