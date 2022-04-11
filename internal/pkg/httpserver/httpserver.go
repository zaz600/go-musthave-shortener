package httpserver

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/zaz600/go-musthave-shortener/internal/pkg/cert"
)

const _defaultKeepAlivePeriod = 3 * time.Minute

type tcpKeepAliveListener struct {
	*net.TCPListener
	keepAlivePeriod time.Duration
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(_defaultKeepAlivePeriod)
	return tc, nil
}

type TLSServer struct {
	server          *http.Server
	keepAlivePeriod time.Duration
	address         string
}

func NewTLSServer(server *http.Server, address string) *TLSServer { // TODO опшины
	return &TLSServer{
		server:          server,
		keepAlivePeriod: _defaultKeepAlivePeriod,
		address:         address,
	}
}

func (t TLSServer) ListenAndServe() error {
	tlsCert, err := cert.New()
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{}
	tlsConfig.NextProtos = []string{"http/1.1"}
	tlsConfig.Certificates = []tls.Certificate{tlsCert}

	ln, err := net.Listen("tcp", t.address)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener), t.keepAlivePeriod}, tlsConfig)

	return t.server.Serve(tlsListener)
}
