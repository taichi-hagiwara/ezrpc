package ezrpc

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/pkg/errors"
)

// Client は、リモート呼び出しを行う。
type Client struct {
	service    Service
	registry   *ServiceRegistry
	httpClient *http.Client
	address    string
}

// NewClient は、クライアントを作成する。
func NewClient(s Service, address, serverName string, cert *CertInfo) (*Client, error) {
	registry := newServiceRegistry()

	err := s.Init(registry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize service")
	}

	certPool, err := cert.CertPool()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load CA")
	}

	keyPair, err := cert.X509KeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load x509 key pair")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		RootCAs:      certPool,
		ServerName:   serverName,
	}
	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	httpClient := &http.Client{Transport: transport}

	return &Client{
		service:    s,
		registry:   registry,
		httpClient: httpClient,
		address:    address,
	}, nil
}

// Invoke は、リモート呼び出しを行う。
func (c *Client) Invoke(name string, args interface{}) (interface{}, error) {
	ep, ok := c.registry.endpoints[name]
	if !ok {
		return nil, errors.Errorf("unknown endpoint: %s", name)
	}

	var req *http.Request
	if args == nil {
		var err error
		req, err = http.NewRequest("GET", fmt.Sprintf("https://%s/%s", c.address, name), nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
	} else {
		b, err := json.Marshal(args)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal args")
		}

		req, err = http.NewRequest("POST", fmt.Sprintf("https://%s/%s", c.address, name), bytes.NewReader(b))
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http error")
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		result := reflect.New(ep.Result).Interface()
		if err := json.Unmarshal(b, result); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal json")
		}

		return result, nil
	case http.StatusNoContent:
		return nil, nil
	default:
		serverError := &ServerError{}
		if err := json.Unmarshal(b, serverError); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal error json")
		}

		return nil, errors.Wrap(serverError, "server error")
	}
}
