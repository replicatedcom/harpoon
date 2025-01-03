package requests

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

// TODO: transport may be a misnomer

var (
	httpTransportPool = make(map[string]*http.Transport)
	defaultTransport  = NewTcpTransport()
)

type Transport interface {
	doRequest(req *http.Request) (*http.Response, error)
}

type TcpTransport struct {
	Client *http.Client
}

func NewTcpTransport() *TcpTransport {
	return &TcpTransport{
		Client: &http.Client{
			Transport: http.DefaultTransport,
		},
	}
}

func NewTlsTransport(pemFilename string) (*TcpTransport, error) {
	roots, err := replicatedCertPool(pemFilename)
	if err != nil {
		return nil, err
	}

	tr := http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: roots,
		},
	}

	return &TcpTransport{
		Client: &http.Client{
			Transport: &tr,
		},
	}, nil
}

func replicatedCertPool(pemFilename string) (*x509.CertPool, error) {
	file, err := ioutil.ReadFile(pemFilename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read pem file %s", pemFilename)
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(file); !ok {
		return nil, errors.New("unable to append root cert")
	}

	return roots, nil
}

func (t *TcpTransport) doRequest(req *http.Request) (*http.Response, error) {
	return t.Client.Do(req)
}

type IpcTransport struct {
	Client  *http.Client
	address string
}

func NewIpcTransport(address string) *IpcTransport {
	return &IpcTransport{
		Client: &http.Client{
			Transport: getUnixTransport(address),
		},
		address: address,
	}
}

func (t *IpcTransport) doRequest(req *http.Request) (*http.Response, error) {
	req.URL.Host = "d"
	req.URL.Scheme = "http"
	return t.Client.Do(req)
}

func getUnixTransport(address string) *http.Transport {
	transport, ok := httpTransportPool[address]
	if !ok {
		transport = &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.Dial("unix", address)
			},
		}
		httpTransportPool[address] = transport
	}
	return transport
}
