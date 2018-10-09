package ezrpc

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// Server は、リモート呼び出しを実際に実行するインスタンスを表す。
type Server interface {
	Service
	Invoke(name string, client *ClientInfo, args interface{}) (result interface{}, err error)
}

// ClientInfo は、クライアントの情報を表す。
type ClientInfo struct {
	TLSSubject pkix.Name
	Remote     IPEndPoint
}

type server struct {
	Server
	*ServiceRegistry
}

func (h *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			if errResp, ok := err.(*ServerError); ok {
				errResp.write(w, r)
			} else if errError, ok := err.(error); ok {
				(&ServerError{
					Message: errError.Error(),
				}).write(w, r)
			} else {
				(&ServerError{
					Message: fmt.Sprint(err),
				}).write(w, r)
			}
		}
	}()

	w.Header().Add("Content-Type", "application/json")

	if r.Method != "POST" {
		panic(&ServerError{StatusCode: 405})
	}

	epName := strings.TrimPrefix(r.URL.Path, "/")

	remote, _ := ParseIPEndPoint(r.RemoteAddr)

	ci := &ClientInfo{
		TLSSubject: r.TLS.PeerCertificates[0].Subject,
		Remote:     remote,
	}

	if ep, ok := h.endpoints[epName]; ok {
		args := reflect.New(ep.Args).Interface()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		if err := json.Unmarshal(b, args); err != nil {
			panic(err)
		}

		result, err := h.Invoke(epName, ci, args)
		if err != nil {
			panic(err)
		}

		b, err = json.Marshal(result)
		if err != nil {
			panic(err)
		}

		w.Write(b)
	} else {
		panic(&ServerError{StatusCode: 404})
	}
}

// Serve は RPC サーバを開始させる。
func Serve(s Server, address string, cert *CertInfo) error {
	registry := &ServiceRegistry{
		endpoints: make(map[string]*endpointType),
	}

	err := s.Init(registry)
	if err != nil {
		return errors.Wrap(err, "failed to initialize service")
	}

	certPool, err := cert.CertPool()
	if err != nil {
		return errors.Wrap(err, "failed to load CA")
	}

	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  certPool,
	}
	tlsConfig.BuildNameToCertificate()

	handler := &server{
		Server:          s,
		ServiceRegistry: registry,
	}

	server := &http.Server{
		TLSConfig: tlsConfig,
		Addr:      address,
		Handler:   handler,
	}

	if err := server.ListenAndServeTLS(cert.Cert, cert.Private); err != nil {
		return errors.Wrap(err, "server down")
	}

	return nil
}
