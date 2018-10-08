package ezrpc

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/pkg/errors"
)

// CertInfo は、証明書や鍵ファイルの場所を表す。
type CertInfo struct {
	// CA は、CA の証明書ファイルのパス。
	CACert string

	// Private は、秘密鍵ファイルのパス。
	Private string

	// Cert は、証明書ファイルのパス。
	Cert string
}

// CertPool は、CA 証明書を含む CertPool を作成する。
func (c *CertInfo) CertPool() (*x509.CertPool, error) {
	certFile, err := ioutil.ReadFile(c.CACert)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read CA cert file: %s.", c.CACert)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(certFile); !ok {
		return nil, errors.Errorf("failed to append cert from .pem: %s", c.CACert)
	}

	return certPool, nil
}

// X509KeyPair は、キー ペアを読み込む。
func (c *CertInfo) X509KeyPair() (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(c.Cert, c.Private)
	if err != nil {
		return cert, errors.Wrap(err, "failed to load x509 keypair")
	}

	return cert, nil
}
