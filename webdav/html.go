package webdav

import (
	"context"
	"fmt"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
	"net/url"
)

// htmlHook is activated, if the target is a folder and the raw query '?html' is appended to the url.
// Example: https://localhost:8080/folder1/?html
func htmlHook(w http.ResponseWriter, name string, fs webdav.FileSystem) {

	// open file (dir)
	dir, err := fs.OpenFile(context.TODO(), name, 0, 0)
	if err != nil {
		log.Printf("ERROR: %s/htmlHook: '%v': path='%s'", packageName, err, name)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer dir.Close() // CLOSE

	// read folder content
	ls, err := dir.Readdir(-1)
	if err != nil {
		log.Printf("ERROR: %s/htmlHook: '%v': path='%s'", packageName, err, name)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// write html text
	for _, v := range ls {
		postfix := ""
		if v.IsDir() {
			postfix = "/?html"
		}

		_, err := fmt.Fprintf(w, "<a href='%s%s'>%s</a><br>\n", url.PathEscape(v.Name()), postfix, v.Name())
		if err != nil {
			log.Printf("ERROR: %s/htmlHook: '%v': path='%s'", packageName, err, name)
			return
		}
	}
}
