package webdav

/*
	IN THIS FILE: HTTP Handler
		- Authentication
		- Authorization
		- list dir
*/

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
	"os"
)

var _ http.Handler = (*_Config)(nil)

// _Config is the configuration of a WebDAV instance.
type _Config struct {
	debugLvl      uint8
	webdavHandler *webdav.Handler
	users         *_Users
}

// newConfig return a http.Handler with authentication and authorization for splitFs webdav files (@see NewFileSystem).
func newConfig(fs webdav.FileSystem, userFile string, debugLvl uint8) http.Handler {
	return &_Config{
		debugLvl: debugLvl,
		webdavHandler: &webdav.Handler{
			Prefix:     "", // not used
			FileSystem: fs,
			LockSystem: webdav.NewMemLS(),
			Logger:     requestLogger, // use own logger
		},
		users: initUsers(userFile, debugLvl),
	}
}

//--------------------------------------------------------------------------------------------------------------------//

// requestLogger is an error logger. If non-nil, it will be called for all HTTP requests.
var requestLogger = func(r *http.Request, err error) {
	// no error, no error log
	if err == nil {
		return // exit
	}

	// request is nil
	if r == nil {
		log.Printf("WARNING: %s/RequestLogger: '%v': http request is nil", packageName, err)
		return // exit
	}

	// get values ('r' is not nil)
	remoteAddr := r.RemoteAddr // "86.15.80.32:40971"
	relPath := r.RequestURI    // "/data/files/test.txt"

	// remove auth header (contain password)
	if r.Header != nil {
		r.Header.Del("Authorization")
		r.Header.Del("authorization")
		r.Header.Del("auth")
	}

	// LOG: "file does not exist"
	if err == os.ErrNotExist {
		log.Printf("WARNING: %s/RequestLogger: '%v': path='%s', client='%s'", packageName, err, relPath, remoteAddr)
		return // exit
	}

	// LOG: default
	log.Printf("WARNING: %s/RequestLogger: '%v': path='%s', client='%s': %#v", packageName, err, relPath, remoteAddr, r)
}

// A Handler responds to an HTTP request.
//
// ServeHTTP should write reply headers and data to the ResponseWriter
// and then return. Returning signals that the request is finished; it
// is not valid to use the ResponseWriter or read from the
// Request.Body after or concurrently with the completion of the
// ServeHTTP call.
//
// Depending on the HTTP client software, HTTP protocol version, and
// any intermediaries between the client and the Go server, it may not
// be possible to read from the Request.Body after writing to the
// ResponseWriter. Cautious handlers should read the Request.Body
// first, and then reply.
//
// Except for reading the body, handlers should not modify the
// provided Request.
//
// If ServeHTTP panics, the server (the caller of ServeHTTP) assumes
// that the effect of the panic was isolated to the active request.
// It recovers the panic, logs a stack trace to the server error log,
// and either closes the network connection or sends an HTTP/2
// RST_STREAM, depending on the HTTP protocol. To abort a handler so
// the client sees an interrupted response but the server doesn't log
// an error, panic with the value ErrAbortHandler.
func (c *_Config) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Authentication
	//---------------------------------------------------------------------------------------------
	var user *_User
	{
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		// BasicAuth returns the username and password provided in the request's
		// Authorization header, if the request uses HTTP Basic Authentication.
		// See RFC 2617, Section 2.
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "Not authorized", http.StatusUnauthorized) // AUTH ERROR
			return
		}

		// is the username known?
		user, ok = c.users.Get(username)
		if !ok || user == nil {
			http.Error(w, "Not authorized", http.StatusUnauthorized) // AUTH ERROR
			return
		}

		// CompareHashAndPassword compares a bcrypt hashed password with its possible
		// plaintext equivalent. Returns nil on success, or an error on failure.
		if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
			log.Printf("WARNING: %s/ServeHTTP: wrong password for user '%s': %v", packageName, username, err)
			http.Error(w, "Not authorized", http.StatusUnauthorized) // AUTH ERROR
			return
		}
	}

	// Authorization
	//---------------------------------------------------------------------------------------------
	{
		// Checks for user permissions relatively to this PATH.
		if !user.Allowed(r.URL.Path) {
			http.Error(w, "Forbidden", http.StatusForbidden) // FORBIDDEN ERROR
			return
		}

		// BLACKLIST: If this request modified the files, return forbidden (read only)
		//   * POST - The POST method is used to submit an entity to the specified resource, often causing a change in state or side effects on the server.
		//   * PUT - The PUT method replaces all current representations of the target resource with the request payload.
		//   * DELETE - The DELETE method deletes the specified resource.
		//   * PATCH - The PATCH method is used to apply partial modifications to a resource.
		//   * COPY - copy a resource from one URI to another (WebDAV extends)
		//   * MKCOL - create collections (a.k.a. a directory) (WebDAV extends)
		//   * MOVE - move a resource from one URI to another (WebDAV extends)
		//   * PROPPATCH - change and delete multiple properties on a resource in a single atomic act (WebDAV extends)
		m := r.Method
		if m == "POST" || m == "PUT" || m == "DELETE" || m == "PATCH" ||
			m == "COPY" || m == "MKCOL" || m == "MOVE" || m == "PROPPATCH" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed) // MethodNotAllowed ERROR
			return
		}

		// WHITELIST: check for unknown header
		//   * GET - The GET method requests a representation of the specified resource. Requests using GET should only retrieve data.
		//   * HEAD - The HEAD method asks for a response identical to that of a GET request, but without the response body.
		//   * CONNECT - The CONNECT method establishes a tunnel to the server identified by the target resource.
		//   * OPTIONS - The OPTIONS method is used to describe the communication options for the target resource.
		//   * TRACE - The TRACE method performs a message loop-back test along the path to the target resource.
		//   * LOCK - put a lock on a resource. WebDAV supports both shared and exclusive locks. (WebDAV extends)
		//   * PROPFIND - retrieve properties, stored as XML, from a web resource. Allow to retrieve the directory hierarchy. (WebDAV extends)
		//   * UNLOCK - remove a lock from a resource (WebDAV extends)
		if m != "GET" && m != "HEAD" && m != "CONNECT" && m != "OPTIONS" && m != "TRACE" &&
			m != "LOCK" && m != "PROPFIND" && m != "UNLOCK" {
			log.Printf("WARNING: %s/ServeHTTP: wrong method: '%s'", packageName, m)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed) // MethodNotAllowed ERROR
			return
		}
	}

	// answer the request
	//---------------------------------------------------------------------------------------------
	{
		// The HEAD method asks for a response identical to that of a GET request, but without the response body.
		if r.Method == "HEAD" {
			w = newResponseWriterNoBody(w)
		}

		// The GET method requests a representation of the specified resource.
		// Requests using GET should only retrieve data.
		//
		// Excerpt from RFC4918, section 9.4:
		// 		GET, when applied to a collection, may return the contents of an
		//		"index.html" resource, a human-readable view of the contents of
		//		the collection, or something else altogether.
		//
		// Get, when applied to collection, will return the same as PROPFIND method.
		if r.Method == "GET" {
			// get stat
			info, err := c.webdavHandler.FileSystem.Stat(context.TODO(), r.URL.Path)
			// file is dir?
			if err == nil && info.IsDir() {

				// html hook
				// example: "http://server:8080/Folder/?html"
				if r.URL.RawQuery == "html" {
					htmlHook(w, r.URL.Path, c.webdavHandler.FileSystem) // HTML VIEW
					return                                              // -> EXIT
				}

				// resource is an collection -> change request to 'PROPFIND'
				r.Method = "PROPFIND"
				if r.Header.Get("Depth") == "" {
					r.Header.Add("Depth", "1")
				}

			}
		}

		// delegate to inner webdav handler
		c.webdavHandler.ServeHTTP(w, r)
	}
}
