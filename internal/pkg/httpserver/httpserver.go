package httpserver

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/zaz600/go-musthave-shortener/internal/pkg/cert"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func ListenTLS(server *http.Server, address string) error {
	tlsCert, err := cert.New()
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{}
	tlsConfig.NextProtos = []string{"http/1.1"}
	tlsConfig.Certificates = []tls.Certificate{tlsCert}

	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, tlsConfig)

	return server.Serve(tlsListener)
}
