package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSFromFiles loads a TLS config from filesystem cert + key, optionally
// requiring client certs signed by the given clientCAFile (mTLS). For pure
// server-side TLS pass clientCAFile = "".
//
// Recommended usage:
//
//	tlsConf, err := server.TLSFromFiles("/etc/tls/server.crt", "/etc/tls/server.key", "")
//	if err != nil { return err }
//	cfg := server.HTTPConfig{Addr: ":8443", TLSConfig: tlsConf}
//
// For mTLS, supply the CA bundle the server should trust for client certs:
//
//	tlsConf, _ := server.TLSFromFiles(certFile, keyFile, "/etc/tls/clients-ca.pem")
//	// kratos / Go will then enforce ClientAuth = RequireAndVerifyClientCert.
func TLSFromFiles(certFile, keyFile, clientCAFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("tls: load cert/key %q + %q: %w", certFile, keyFile, err)
	}
	conf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if clientCAFile != "" {
		ca, err := os.ReadFile(clientCAFile)
		if err != nil {
			return nil, fmt.Errorf("tls: read client CA %q: %w", clientCAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("tls: no certs parsed from %q", clientCAFile)
		}
		conf.ClientCAs = pool
		conf.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return conf, nil
}
