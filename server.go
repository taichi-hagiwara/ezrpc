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

// HandlerFunc は、リモード呼び出しの実際の動作を表す。
type HandlerFunc func(client *ClientInfo, args interface{}) (result interface{}, err error)

// Server は、リモート呼び出しのサーバを表す。
type Server struct {
	service  Service
	handlers map[string]HandlerFunc
}

// ClientInfo は、クライアントの情報を表す。
type ClientInfo struct {
	TLSSubject pkix.Name
	Remote     IPEndPoint
}

// NewServer は、リモート呼び出しのサーバを作成する。
func NewServer(service Service) *Server {
	return &Server{
		service:  service,
		handlers: make(map[string]HandlerFunc),
	}
}

// RegisterHandler は、リモート呼び出しの実際の動作を登録する。
func (s *Server) RegisterHandler(name string, f HandlerFunc) {
	s.handlers[name] = f
}

type httpHandler struct {
	*Server
	*ServiceRegistry
}

// Listen は、RPCサーバを起動し、待機する。
func (s *Server) Listen(address string, cert *CertInfo) error {
	registry := newServiceRegistry()
	if err := s.service.Init(registry); err != nil {
		return errors.Wrap(err, "failed to initialize service")
	}

	for i := range registry.endpoints {
		if _, ok := s.handlers[i]; !ok {
			return errors.Errorf("endpoint \"%s\" has no handler", i)
		}
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

	handler := &httpHandler{
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

func (h *httpHandler) Invoke(name string, client *ClientInfo, args interface{}) (result interface{}, err error) {
	return h.handlers[name](client, args)
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

		result, err := h.handlers[epName](ci, args)
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
