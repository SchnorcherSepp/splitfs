package webdav

/*
	IN THIS FILE: run webdav server
		- open listener
		- start server
		- config TLS
*/

import (
	"crypto/tls"
	"golang.org/x/net/webdav"
	"log"
	"net"
	"net/http"
	"time"
)

// ServeTLS accepts incoming HTTPS connections on the listener,
// creating a new service goroutine for each. The service goroutines
// read requests and then call handler to reply to them.
//
// If the lAddr is empty or a literal unspecified IP address, Listen
// listens on all available unicast and anycast IP addresses of the local system.
// The address can use a host name, but this is not recommended,
// because it will create a listener for at most one of the host's IP
// addresses.
// If the port in the address parameter is empty or "0", as in
// "127.0.0.1:" or "[::1]:0", a port number is automatically chosen.
//
// Additionally, files containing a certificate and matching private key
// for the server must be provided. If the certificate is signed by a
// certificate authority, the certFile should be the concatenation of
// the server's certificate, any intermediates, and the CA's certificate.
//
// ServeTLS always returns a non-nil error.
func Serve(lAddr string, useTls bool, certFile string, certKeyFile string, fs webdav.FileSystem, userFile string, debugLvl uint8) error {

	// tcp listener
	listener, err := net.Listen("tcp", lAddr)
	if err != nil {
		log.Printf("ERROR: %s/Serve: listener: %v: lAddr='%s'", packageName, err, lAddr)
		return err
	}

	// Tell the user the port in which is listening.
	log.Printf("INFO: %s/Serve: Listening on %s", packageName, listener.Addr().String())

	// handler
	handler := newConfig(fs, userFile, debugLvl)

	// Starts the server.
	if !useTls { //------------------------------------------------------------------
		log.Printf("INFO: %s/Serve: start ...", packageName)
		if err := http.Serve(listener, handler); err != nil {
			// Serve always returns a non-nil error.
			log.Printf("ERROR: %s/Serve: %v", packageName, err)
			return err
		}

	} else { //----------------------------------------------------------------------
		log.Printf("INFO: %s/Serve: start with TLS ...", packageName)

		// server hardening
		// https://blog.cloudflare.com/exposing-go-on-the-internet/
		srv := &http.Server{
			// Timeouts
			IdleTimeout: 120 * time.Second,
			// TLS config
			TLSConfig: &tls.Config{
				// disable TLS 1.0 and TLS 1.1
				MinVersion: tls.VersionTLS12,
				// Causes servers to use Go's default ciphersuite preferences,
				// which are tuned to avoid attacks. Does nothing on clients.
				PreferServerCipherSuites: true,
			},
			// handler (@see webdav/config.go)
			Handler: handler,
		}

		// server
		if err := srv.ServeTLS(listener, certFile, certKeyFile); err != nil {
			// ServeTLS always returns a non-nil error.
			log.Printf("ERROR: %s/ServeTLS: %v", packageName, err)
			return err
		}
	} //-----------------------------------------------------------------------------

	return nil
}
