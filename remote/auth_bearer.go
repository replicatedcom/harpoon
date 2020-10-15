// Borrowed from https://github.com/docker/distribution/blob/2800ab02245e2eafc10e338939511dd1aeb5e135/registry/client/auth/session.go

package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth"
)

var (
	// ErrNoToken is returned if a request is successful but the body does not
	// contain an authorization token.
	ErrNoToken = errors.New("authorization server did not include a token in the response")
)

const defaultClientID = "registry-client"

// This is the minimum duration a token can last (in seconds).
// A token must not live less than 60 seconds because older versions
// of the Docker client didn't read their expiration from the token
// response and assumed 60 seconds.  So to remain compatible with
// those implementations, a token must live at least this long.
const minimumTokenLifetimeSeconds = 60

// Private interface for time used by this package to enable tests to provide their own implementation.
type clock interface {
	Now() time.Time
}

type tokenHandler struct {
	header    http.Header
	creds     auth.CredentialStore
	transport http.RoundTripper
	clock     clock

	offlineAccess bool
	forceOAuth    bool
	clientID      string
	scopes        []auth.Scope

	tokenLock       sync.Mutex
	tokenCache      string
	tokenExpiration time.Time
}

// An implementation of clock for providing real time data.
type realClock struct{}

// Now implements clock
func (realClock) Now() time.Time { return time.Now() }

// NewTokenHandlerWithOptions creates a new token handler using the provided
// options structure.
func NewTokenHandlerWithOptions(options auth.TokenHandlerOptions) *tokenHandler {
	handler := &tokenHandler{
		transport:     options.Transport,
		creds:         options.Credentials,
		offlineAccess: options.OfflineAccess,
		forceOAuth:    options.ForceOAuth,
		clientID:      options.ClientID,
		scopes:        options.Scopes,
		clock:         realClock{},
	}

	return handler
}

func (th *tokenHandler) client() *http.Client {
	return &http.Client{
		Transport: th.transport,
		Timeout:   15 * time.Second,
	}
}

func (th *tokenHandler) GetToken(params map[string]string, additionalScopes ...string) (string, error) {
	scopes := make([]string, 0, len(th.scopes)+len(additionalScopes))
	for _, scope := range th.scopes {
		scopes = append(scopes, scope.String())
	}
	var addedScopes bool
	for _, scope := range additionalScopes {
		scopes = append(scopes, scope)
		addedScopes = true
	}

	now := th.clock.Now()
	if now.After(th.tokenExpiration) || addedScopes {
		token, expiration, err := th.fetchToken(params, scopes)
		if err != nil {
			return "", err
		}

		// do not update cache for added scope tokens
		if !addedScopes {
			th.tokenCache = token
			th.tokenExpiration = expiration
		}

		return token, nil
	}

	return th.tokenCache, nil
}

type postTokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	Scope        string    `json:"scope"`
}

func (th *tokenHandler) fetchTokenWithOAuth(realm *url.URL, refreshToken, service string, scopes []string) (token string, expiration time.Time, err error) {
	form := url.Values{}
	form.Set("scope", strings.Join(scopes, " "))
	form.Set("service", service)

	clientID := th.clientID
	if clientID == "" {
		// Use default client, this is a required field
		clientID = defaultClientID
	}
	form.Set("client_id", clientID)

	if refreshToken != "" {
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", refreshToken)
	} else if th.creds != nil {
		form.Set("grant_type", "password")
		username, password := th.creds.Basic(realm)
		form.Set("username", username)
		form.Set("password", password)

		// attempt to get a refresh token
		form.Set("access_type", "offline")
	} else {
		// refuse to do oauth without a grant type
		return "", time.Time{}, fmt.Errorf("no supported grant type")
	}

	resp, err := th.client().PostForm(realm.String(), form)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if !client.SuccessStatus(resp.StatusCode) {
		err := client.HandleErrorResponse(resp)
		return "", time.Time{}, err
	}

	decoder := json.NewDecoder(resp.Body)

	var tr postTokenResponse
	if err = decoder.Decode(&tr); err != nil {
		return "", time.Time{}, fmt.Errorf("unable to decode token response: %s", err)
	}

	if tr.RefreshToken != "" && tr.RefreshToken != refreshToken {
		th.creds.SetRefreshToken(realm, service, tr.RefreshToken)
	}

	if tr.ExpiresIn < minimumTokenLifetimeSeconds {
		// The default/minimum lifetime.
		tr.ExpiresIn = minimumTokenLifetimeSeconds
	}

	if tr.IssuedAt.IsZero() {
		// issued_at is optional in the token response.
		tr.IssuedAt = th.clock.Now().UTC()
	}

	return tr.AccessToken, tr.IssuedAt.Add(time.Duration(tr.ExpiresIn) * time.Second), nil
}

type getTokenResponse struct {
	Token        string    `json:"token"`
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	RefreshToken string    `json:"refresh_token"`
}

func (th *tokenHandler) fetchTokenWithBasicAuth(realm *url.URL, service string, scopes []string) (token string, expiration time.Time, err error) {

	req, err := http.NewRequest("GET", realm.String(), nil)
	if err != nil {
		return "", time.Time{}, err
	}

	reqParams := req.URL.Query()

	if service != "" {
		reqParams.Add("service", service)
	}

	for _, scope := range scopes {
		reqParams.Add("scope", scope)
	}

	if th.offlineAccess {
		reqParams.Add("offline_token", "true")
		clientID := th.clientID
		if clientID == "" {
			clientID = defaultClientID
		}
		reqParams.Add("client_id", clientID)
	}

	if th.creds != nil {
		username, password := th.creds.Basic(realm)
		if username != "" && password != "" {
			reqParams.Add("account", username)
			req.SetBasicAuth(username, password)
		}
	}

	req.URL.RawQuery = reqParams.Encode()

	resp, err := th.client().Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if !client.SuccessStatus(resp.StatusCode) {
		err := client.HandleErrorResponse(resp)
		return "", time.Time{}, err
	}

	decoder := json.NewDecoder(resp.Body)

	var tr getTokenResponse
	if err = decoder.Decode(&tr); err != nil {
		return "", time.Time{}, fmt.Errorf("unable to decode token response: %s", err)
	}

	if tr.RefreshToken != "" && th.creds != nil {
		th.creds.SetRefreshToken(realm, service, tr.RefreshToken)
	}

	// `access_token` is equivalent to `token` and if both are specified
	// the choice is undefined.  Canonicalize `access_token` by sticking
	// things in `token`.
	if tr.AccessToken != "" {
		tr.Token = tr.AccessToken
	}

	if tr.Token == "" {
		return "", time.Time{}, ErrNoToken
	}

	if tr.ExpiresIn < minimumTokenLifetimeSeconds {
		// The default/minimum lifetime.
		tr.ExpiresIn = minimumTokenLifetimeSeconds
	}

	if tr.IssuedAt.IsZero() {
		// issued_at is optional in the token response.
		tr.IssuedAt = th.clock.Now().UTC()
	}

	return tr.Token, tr.IssuedAt.Add(time.Duration(tr.ExpiresIn) * time.Second), nil
}

func (th *tokenHandler) fetchToken(params map[string]string, scopes []string) (token string, expiration time.Time, err error) {
	realm, ok := params["realm"]
	if !ok {
		return "", time.Time{}, errors.New("no realm specified for token auth challenge")
	}

	// TODO(dmcgowan): Handle empty scheme and relative realm
	realmURL, err := url.Parse(realm)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid token auth challenge realm: %s", err)
	}

	service := params["service"]

	var refreshToken string

	if th.creds != nil {
		refreshToken = th.creds.RefreshToken(realmURL, service)
	}

	if refreshToken != "" || th.forceOAuth {
		return th.fetchTokenWithOAuth(realmURL, refreshToken, service, scopes)
	}

	return th.fetchTokenWithBasicAuth(realmURL, service, scopes)
}
