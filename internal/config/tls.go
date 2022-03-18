package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

type TLSConfig struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	ServerAddress string
	Server        bool
}

func SetupTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		if cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile); err != nil {
			return nil, err
		} else {
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	if cfg.CAFile != "" {
		pemCerts, err := ioutil.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, err
		}

		ca := x509.NewCertPool()
		ok := ca.AppendCertsFromPEM([]byte(pemCerts))
		if !ok {
			return nil, fmt.Errorf("failed to parse root certificate: %q", cfg.CAFile)
		}

		if cfg.Server {
			tlsConfig.ClientCAs = ca
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.RootCAs = ca
		}

		tlsConfig.ServerName = cfg.ServerAddress
	}

	return tlsConfig, nil
}
