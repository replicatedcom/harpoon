package dockerreg

import (
	"fmt"
	"net/http"

	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/requests"
)

func (dockerRemote *DockerRemote) Auth() error {
	uri := fmt.Sprintf("https://%s/v2/", dockerRemote.Hostname)

	client := requests.GlobalHttpClient()

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := doRequest(req, client, dockerRemote, 0)
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

	return nil
}
