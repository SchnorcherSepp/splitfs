package webdav

/*
	IN THIS FILE: helper (encapsulation)
		- db.VirtFile  ---to-->  os.FileInfo
*/

import (
	"context"
	"github.com/SchnorcherSepp/splitfs/db"
	"golang.org/x/net/webdav"
	"mime"
	"os"
	"path/filepath"
	"time"
)

var _ os.FileInfo = (*_FileInfo)(nil)
var _ webdav.ContentTyper = (*_FileInfo)(nil)

// _FileInfo hold stats returned by Stat and Lstat.
type _FileInfo struct {
	innerFile db.VirtFile
}

// newFileInfo return a db.VirtFile as os.FileInfo.
func newFileInfo(innerFile db.VirtFile) os.FileInfo {
	return &_FileInfo{
		innerFile: innerFile,
	}
}

// ------------------------------------------------------------------------------------------------------------------ //

// Name return the base name of the file
func (i *_FileInfo) Name() string {
	return i.innerFile.Name()
}

// Size return the length in bytes for regular files
func (i *_FileInfo) Size() int64 {
	return i.innerFile.FileSize
}

// Mode return the file mode bits.
//   File: 0666
//   Dir: 0777
func (i *_FileInfo) Mode() os.FileMode {
	if i.IsDir() {
		return 0777 // folder
	} else {
		return 0666 // file
	}
}

// ModTime return the modification time
func (i *_FileInfo) ModTime() time.Time {
	return time.Unix(i.innerFile.MTime, 0)
}

// IsDir return if the object is a dir
func (i *_FileInfo) IsDir() bool {
	return i.innerFile.IsDir
}

// Sys is not used and return nil
func (i *_FileInfo) Sys() interface{} {
	return nil // not used
}

// ContentType returns the content type for the file.
// This is an optional interface for the os.FileInfo objects returned by the FileSystem.
// If this interface is defined then it will be used to read the content type from the object (read 512 bytes).
//
// If this returns error ErrNotImplemented then the error will
// be ignored and the base implementation will be used
// instead.
func (i *_FileInfo) ContentType(_ context.Context) (string, error) {

	// This implementation is based on serveContent's code in the standard net/http package.
	cType := mime.TypeByExtension(filepath.Ext(i.Name()))
	if cType != "" {
		return cType, nil
	}

	// default: binary
	return "application/octet-stream", nil
}
