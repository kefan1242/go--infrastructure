package server_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
	"github.com/kris/go-infrastructure/pkg/testutil"
)

// genSelfSignedCert writes a fresh ECDSA cert + key to dir/cert.pem and
// dir/key.pem, returning the paths.
func genSelfSignedCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "kris-test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("createcert: %v", err)
	}
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certOut, _ := os.Create(certPath)
	_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	_ = certOut.Close()

	keyDER, _ := x509.MarshalECPrivateKey(priv)
	keyOut, _ := os.Create(keyPath)
	_ = pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	_ = keyOut.Close()

	return certPath, keyPath
}

func TestTLSFromFiles_LoadsCertAndKey(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := genSelfSignedCert(t, dir)

	conf, err := pkgserver.TLSFromFiles(certFile, keyFile, "")
	if err != nil {
		t.Fatalf("TLSFromFiles: %v", err)
	}
	if len(conf.Certificates) != 1 {
		t.Errorf("certs: want 1, got %d", len(conf.Certificates))
	}
	if conf.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion: want TLS12, got %d", conf.MinVersion)
	}
	if conf.ClientAuth != tls.NoClientCert {
		t.Errorf("ClientAuth: server-only TLS should be NoClientCert, got %d", conf.ClientAuth)
	}
}

func TestTLSFromFiles_MTLSRequiresClientCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := genSelfSignedCert(t, dir)
	// reuse the same cert as the client CA bundle (any valid PEM works for the test)
	caFile := certFile

	conf, err := pkgserver.TLSFromFiles(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("TLSFromFiles: %v", err)
	}
	if conf.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("mTLS: want RequireAndVerifyClientCert, got %d", conf.ClientAuth)
	}
	if conf.ClientCAs == nil {
		t.Error("expected ClientCAs pool")
	}
}

func TestTLSFromFiles_BadCertReturnsError(t *testing.T) {
	if _, err := pkgserver.TLSFromFiles("/does/not/exist.crt", "/no.key", ""); err == nil {
		t.Fatal("expected error for missing cert/key")
	}
}

func TestTLSFromFiles_BadClientCAReturnsError(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := genSelfSignedCert(t, dir)
	garbage := filepath.Join(dir, "garbage.pem")
	_ = os.WriteFile(garbage, []byte("not a cert"), 0o600)

	if _, err := pkgserver.TLSFromFiles(certFile, keyFile, garbage); err == nil {
		t.Fatal("expected error for unparseable client CA")
	}
}

func TestNewBizHTTPServer_TLSEndpointServesHTTPS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := genSelfSignedCert(t, dir)
	tlsConf, err := pkgserver.TLSFromFiles(certFile, keyFile, "")
	if err != nil {
		t.Fatalf("TLSFromFiles: %v", err)
	}

	logger, _ := testutil.NewMemoryLogger()
	addr := freePort(t)

	srv := pkgserver.NewBizHTTPServer(
		pkgserver.HTTPConfig{
			Network:   "tcp",
			Addr:      addr,
			TLSConfig: tlsConf,
		},
		logger,
		func(s *pkgserver.BizHTTPServer) {
			s.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("tls-ok"))
			})
		},
	)
	go func() { _ = srv.S.Start(context.Background()) }()
	t.Cleanup(func() { _ = srv.S.Stop(context.Background()) })
	waitForReachable(t, addr)

	// Client trusts our self-signed cert by skipping verification.
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}}
	resp, err := client.Get("https://" + addr + "/")
	if err != nil {
		t.Fatalf("https GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "tls-ok" {
		t.Errorf("body: want tls-ok, got %q", string(body))
	}
}
