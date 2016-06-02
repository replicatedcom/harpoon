package requests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

var UnknownContentTypeError = errors.New("Unknown response content")

var (
	globalRestClient        *RestClient
	globalRestClientProxied *RestClient
)

type RestClient struct {
	*HttpClient
}

func InitGlobalRestClients(proxyParam string) error {
	var err error

	globalRestClient, err = NewRestClient("", "", "")
	if err != nil {
		return err
	}

	if proxyParam != "" {
		globalRestClientProxied, err = NewRestClient("", "", proxyParam)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewRestClient(ua, pemFilename, proxyAddress string) (*RestClient, error) {
	httpClient, err := NewHttpClient(ua, pemFilename, proxyAddress)
	if err != nil {
		return nil, err
	}

	c := &RestClient{
		HttpClient: httpClient,
	}

	c.Header.Set("Accept", "application/json")

	return c, nil
}

func GlobalRestClient(useProxy bool) *RestClient {
	if globalRestClientProxied == nil || !useProxy {
		return globalRestClient
	}

	return globalRestClientProxied
}

func (c *RestClient) Get(url string) (*http.Response, error) {
	req, err := c.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *RestClient) Head(url string) (*http.Response, error) {
	req, err := c.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *RestClient) Post(url string, payload interface{}) (*http.Response, error) {
	req, err := c.NewRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *RestClient) Put(url string, payload interface{}) (*http.Response, error) {
	req, err := c.NewRequest("PUT", url, payload)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *RestClient) Patch(url string, payload interface{}) (*http.Response, error) {
	req, err := c.NewRequest("PATCH", url, payload)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *RestClient) Delete(url string) (*http.Response, error) {
	req, err := c.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *RestClient) NewRequest(method, urlStr string, payload interface{}) (*http.Request, error) {
	b, err := json.Marshal(&payload)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	req, err := c.HttpClient.NewRequest(method, urlStr, buf)
	if err != nil {
		return nil, err
	}
	if method == "POST" || method == "PUT" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *RestClient) Do(req *http.Request) (*http.Response, error) {
	return c.GetTransport().doRequest(req)
}

func ReadJsonResponseBody(res *http.Response, v interface{}) error {
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if IsResponseJson(res) {
		if string(data) == "" {
			return nil
		}
		return json.Unmarshal(data, v)
	}

	if res.StatusCode >= 400 {
		return errors.New(string(data))
	}

	return UnknownContentTypeError
}

func IsResponseJson(res *http.Response) bool {
	contentType := res.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return true
	}
	return false
}
