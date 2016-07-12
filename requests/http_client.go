package requests

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	globalHttpClient *HttpClient
)

type HttpClient struct {
	Header    http.Header
	Transport Transport
}

func InitGlobalHttpClient(proxyParam string) error {
	var err error

	globalHttpClient, err = newHttpClient("Harpoon-Client/0_0", "", proxyParam)
	if err != nil {
		return err
	}

	return nil
}

func newHttpClient(ua, pemFilename, proxyAddress string) (*HttpClient, error) {
	c := &HttpClient{
		Header: make(http.Header),
	}
	if ua != "" {
		c.Header.Set("User-Agent", ua)
	}

	var t *TcpTransport
	if pemFilename == "" {
		t = NewTcpTransport()
	} else {
		var err error
		t, err = NewTlsTransport(pemFilename)
		if err != nil {
			return nil, err
		}
	}

	if proxyAddress != "" {
		p := proxyAddress // necessary?
		t.Client.Transport.(*http.Transport).Proxy = func(req *http.Request) (*url.URL, error) {
			// Honor NO_PROXY environment variable
			if !UseProxy(canonicalAddr(req.URL)) {
				return nil, nil
			}
			return url.Parse(p)
		}
	}

	c.Transport = t

	return c, nil
}

func GlobalHttpClient() *HttpClient {
	return globalHttpClient
}

func (c *HttpClient) Get(url string) (*http.Response, error) {
	req, err := c.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *HttpClient) Head(url string) (*http.Response, error) {
	req, err := c.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *HttpClient) Post(url string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := c.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(req)
}

func (c *HttpClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

func (c *HttpClient) NewRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	for key, vals := range c.Header {
		for _, val := range vals {
			req.Header.Add(key, val)
		}
	}
	return req, err
}

func (c *HttpClient) Do(req *http.Request) (*http.Response, error) {
	return c.GetTransport().doRequest(req)
}

func (c *HttpClient) GetTransport() Transport {
	if c.Transport != nil {
		return c.Transport
	}
	return defaultTransport
}
