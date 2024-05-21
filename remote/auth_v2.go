package remote

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/transport"
	dockerregistrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/replicatedcom/harpoon/log"
)

var (
	ErrUnauthorized = errors.New("Unauthorized")
)

func (dockerRemote *DockerRemote) Auth(additionalScope ...string) error {
	uri := fmt.Sprintf("https://%s/v2/", dockerRemote.Hostname)

	req, err := dockerRemote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := dockerRemote.DoWithRetry(req, 3, additionalScope...)
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

// resolveAuth will return a new JWT token from the resources in the authenticateHeader string
func (dockerRemote *DockerRemote) resolveAuth(authenticateHeader string, additionalScope ...string) error {
	switch {
	case strings.HasPrefix(authenticateHeader, "Bearer "):
		return dockerRemote.resolveBearerAuth(authenticateHeader, additionalScope...)
	case strings.HasPrefix(authenticateHeader, "Basic "):
		return dockerRemote.resolveBasicAuth(authenticateHeader, additionalScope...)
	default:
		return fmt.Errorf("unsupported authentication type: %s", authenticateHeader)
	}
}

func (dockerRemote *DockerRemote) resolveBearerAuth(authenticateHeader string, additionalScope ...string) error {
	authenticateHeader = strings.TrimPrefix(authenticateHeader, "Bearer ")

	realm, service, scope := parseAuthenticateHeader(authenticateHeader)

	params := map[string]string{
		"realm":   realm,
		"service": service,
	}

	modifiers := registry.Headers("Replicated", nil)
	authTransport := transport.NewTransport(nil, modifiers...)

	credentialAuthConfig := &dockerregistrytypes.AuthConfig{
		Username:      dockerRemote.Username,
		Password:      dockerRemote.Password,
		ServerAddress: dockerRemote.Hostname,
		// TODO: what is dockerRemote.Token?
	}
	creds := registry.NewStaticCredentialStore(credentialAuthConfig)
	th := NewTokenHandlerWithOptions(auth.TokenHandlerOptions{
		Transport:   authTransport,
		Credentials: creds,
	})

	if scope != "" {
		additionalScope = append([]string{scope}, additionalScope...)
	}
	additionalScope = uniqueStringSlice(additionalScope)

	token, err := th.GetToken(params, additionalScope...)
	if err != nil {
		if isUnauthorizedErr(err) {
			return ErrUnauthorized
		}
		log.Errorf("Failed to get token for hostname %s and username %s: %v", dockerRemote.Hostname, dockerRemote.Username, err)
		return err
	}

	dockerRemote.ServiceHostname = service
	dockerRemote.AuthHeader = fmt.Sprintf("Bearer %s", token)

	return nil
}

func isUnauthorizedErr(err error) bool {
	if err, ok := err.(errcode.Errors); ok && err.Len() > 0 {
		if err, ok := err[0].(errcode.Error); ok && err.Code == errcode.ErrorCodeUnauthorized {
			return true
		}
	}
	return false
}

func (dockerRemote *DockerRemote) resolveBasicAuth(authenticateHeader string, additionalScope ...string) error {
	// Logging on info level for troubleshooting
	log.Infof("Resolving basic auth header: %s", authenticateHeader)

	v := url.Values{}
	if len(additionalScope) > 0 {
		v.Set("scope", additionalScope[0])
	}

	uri := fmt.Sprintf("https://%s?%s", dockerRemote.Hostname, v.Encode())

	log.Debugf("auth uri = %s", uri)
	req, err := dockerRemote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	req.SetBasicAuth(dockerRemote.Username, dockerRemote.Password)

	resp, err := dockerRemote.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		log.Error(ErrUnauthorized)
		return ErrUnauthorized
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read response to auth request: %v", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected response: %d", resp.StatusCode)
		log.Errorf("unexpected response status=%d; error=%s", resp.StatusCode, body)
		return err
	}

	// TODO: github responds with `{"status": "ok", "message": "Hello, world! This is the GitHub Package Registry."}`
	// so not sure what to do with response until there is a registry that uses it.

	dockerRemote.AuthHeader = req.Header.Get("Authorization")

	return nil
}

func (dockerRemote *DockerRemote) resolveECRAuth(ecrEndpoint string) error {
	registry, zone, err := parseECREndpoint(ecrEndpoint)
	if err != nil {
		log.Error(err)
		return err
	}

	ecrService := getECRService(dockerRemote.Username, dockerRemote.Password, zone)

	ecrToken, err := ecrService.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			&registry,
		},
	})
	if err != nil {
		log.Error(err)
		return err
	}

	if len(ecrToken.AuthorizationData) == 0 {
		err := fmt.Errorf("Provided ECR repo: %s not accessible with credentials", ecrEndpoint)
		log.Error(err)
		return err
	}

	token := *ecrToken.AuthorizationData[0].AuthorizationToken

	dockerRemote.AuthHeader = fmt.Sprintf("Basic %s", token)
	return nil
}

func parseAuthenticateHeader(authenticateHeader string) (realm, service, scope string) {
	headerParts := strings.Split(authenticateHeader, ",")
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

	return realm, service, scope
}

func getECRService(accessKeyID, secretAccessKey, zone string) *ecr.ECR {
	awsConfig := &aws.Config{Region: aws.String(zone)}
	awsConfig.Credentials = credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
	return ecr.New(session.New(awsConfig))
}

func parseECREndpoint(endpoint string) (registry, zone string, err error) {
	invalidECRErr := errors.New("Invalid ECR URL")
	splitEndpoint := strings.Split(endpoint, ".")
	if len(splitEndpoint) < 6 {
		log.Debugf("Invalid ECR endpoint: %s provided", endpoint)
		return "", "", invalidECRErr
	}

	if splitEndpoint[1] != "dkr" || splitEndpoint[2] != "ecr" {
		log.Debugf("Invalid ECR endpoint: %s provided", endpoint)
		return "", "", invalidECRErr
	}

	return splitEndpoint[0], splitEndpoint[3], nil
}

func isValidAWSEndpoint(host string) bool {
	return strings.HasSuffix(host, ".amazonaws.com")
}

func uniqueStringSlice(strSlice []string) []string {
	keys := make(map[string]bool)
	next := []string{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			next = append(next, entry)
		}
	}
	return next
}
