package requests

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	transportMu     sync.Mutex
	globalTransport Transport
)

type HttpClient struct {
	Header    http.Header
	Transport Transport
}

func GetHttpClient(proxyParam string) (*HttpClient, error) {
	transportMu.Lock()
	defer transportMu.Unlock()

	if globalTransport == nil {
		t, err := newTransport("", proxyParam)
		if err != nil {
			return nil, err
		}
		globalTransport = t
	}

	c := &HttpClient{
		Header: make(http.Header),
	}
	c.Header.Set("User-Agent", "Harpoon-Client/0_1")

	c.Transport = globalTransport
	return c, nil
}

func newTransport(pemFilename, proxyAddress string) (*TcpTransport, error) {
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
		t.Client.Transport.(*http.Transport).Proxy = func(req *http.Request) (*url.URL, error) {
			// Honor NO_PROXY environment variable
			if !UseProxy(canonicalAddr(req.URL)) {
				return nil, nil
			}
			return url.Parse(proxyAddress)
		}
	}

	return t, nil
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
