package webdav

/*
	IN THIS FILE: helper (encapsulation)
		- ResponseWriter HEAD requests
		- drop body content
*/

import (
	"net/http"
)

var _ http.ResponseWriter = (*_ResponseWriterNoBody)(nil)

// _ResponseWriterNoBody is a wrapper used to suppresses the body of the response
// to a request. Mainly used for HEAD requests.
type _ResponseWriterNoBody struct {
	http.ResponseWriter
}

// newResponseWriterNoBody creates a new responseWriterNoBody.
func newResponseWriterNoBody(w http.ResponseWriter) http.ResponseWriter {
	return &_ResponseWriterNoBody{w}
}

//--------------------------------------------------------------------------------------------------------------------//

// Header executes the Header method from the http.ResponseWriter.
func (w *_ResponseWriterNoBody) Header() http.Header {
	return w.ResponseWriter.Header()
}

// WriteHeader writes the header to the http.ResponseWriter.
func (w *_ResponseWriterNoBody) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

//------------------------------------------------------------------

// Write suppresses the body.
func (w *_ResponseWriterNoBody) Write(_ []byte) (int, error) {
	return 0, nil // RETURN NOTHING
}
