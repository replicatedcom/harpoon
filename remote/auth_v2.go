package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/replicatedcom/harpoon/log"
)

var (
	ErrUnauthorized = errors.New("Unauthorized")
)

func (dockerRemote *DockerRemote) Auth() error {
	uri := fmt.Sprintf("https://%s/v2/", dockerRemote.Hostname)

	req, err := dockerRemote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := dockerRemote.DoWithRetry(req, 3)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		log.Error(ErrUnauthorized)
		return ErrUnauthorized
	} else if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Unexpected status code for %s: %d", uri, resp.StatusCode)
		log.Error(err)
		return err
	}

	// these are v1 things
	// dockerRemote.RemoteEndpoints = resp.Header.Get("X-Docker-Endpoints")
	// dockerRemote.RemoteToken = resp.Header.Get("X-Docker-Token")
	// dockerRemote.RemoteCookie = resp.Header.Get("Set-Cookie")

	return nil
}

// getJWTToken will return a new JWT token from the resources in the authenticateHeader string
func (dockerRemote *DockerRemote) getJWTToken(authenticateHeader string, additionalScope ...string) error {
	if !strings.HasPrefix(authenticateHeader, "Bearer ") {
		return errors.New("only bearer auth is implemented")
	}
	authenticateHeader = strings.TrimPrefix(authenticateHeader, "Bearer ")

	headerParts := strings.Split(authenticateHeader, ",")
	var realm, scope, service string
	for _, headerPart := range headerParts {
		split := strings.Split(headerPart, "=")
		if len(split) != 2 {
			continue
		}

		switch split[0] {
		case "realm":
			realm = strings.Trim(split[1], "\"")
		case "service":
			service = strings.Trim(split[1], "\"")
		case "scope":
			scope = strings.Trim(split[1], "\"")
		}
	}

	// NOTE: It seems that sometimes scope is not returned with authorization failures.
	// Most of the time scope can be inferred by the client.
	if scope == "" && len(additionalScope) > 0 {
		scope = additionalScope[0]
	}

	v := url.Values{}
	v.Set("service", service)
	if len(scope) > 0 {
		v.Set("scope", scope)
	}
	uri := fmt.Sprintf("%s?%s", realm, v.Encode())

	log.Debugf("auth uri = %s", uri)
	req, err := dockerRemote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	if dockerRemote.Username != "" && dockerRemote.Password != "" {
		req.SetBasicAuth(dockerRemote.Username, dockerRemote.Password)
	} else if dockerRemote.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", dockerRemote.Token))
	}

	resp, err := dockerRemote.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		log.Error(ErrUnauthorized)
		return ErrUnauthorized
	} else if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected response: %d", resp.StatusCode)
		log.Error(err)
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read response to auth request: %v", err)
		return err
	}

	type tokenResponse struct {
		Token string `json:"token"`
	}
	tr := tokenResponse{}
	if err := json.Unmarshal(body, &tr); err != nil {
		log.Errorf("unmarshal error: %v: %q", err, body)
		return err
	}

	dockerRemote.ServiceHostname = service
	dockerRemote.JWTToken = tr.Token

	return nil
}
