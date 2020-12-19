package webdav

/*
	IN THIS FILE: I/O Implementation
		- Read(), Close(), Seek()  ->  return ReaderAt
		- ReadDir()  ->  return dir content
*/

import (
	"github.com/SchnorcherSepp/splitfs/core"
	"github.com/SchnorcherSepp/splitfs/db"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"golang.org/x/net/webdav"
	"io"
	"os"
	"sync"
)

var _ webdav.File = (*_File)(nil)
var lagBuffer = make([]byte, 50*1024*1024) // TODO: WTF!

// _File is returned by a FileSystem's OpenFile method and can be served by a Handler.
type _File struct {
	innerFile   db.VirtFile
	fs          *_FileSystem
	innerReader interf.ReaderAt
	innerOff    int64
	offLock     *sync.Mutex
}

// newFile encapsulate a db.VirtFile and return a webdav.File (random read access)
func newFile(file db.VirtFile, fs *_FileSystem) webdav.File {
	return &_File{
		innerFile:   file,
		fs:          fs,
		innerReader: nil, // set by first Read()
		innerOff:    0,
		offLock:     new(sync.Mutex),
	}
}

// ------------------------------------------------------------------------------------------------------------------ //

// Close @see os.File
//
// Close closes the File, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will be canceled
// and return immediately with an error. Close will return an error if it has already been called.
func (f *_File) Close() error {
	f.offLock.Lock()
	defer f.offLock.Unlock()

	if f.innerReader != nil {
		return f.innerReader.Close()
	}
	return nil
}

// Read @see os.File
//
// Read reads up to len(b) bytes from the File.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *_File) Read(p []byte) (n int, err error) {
	f.offLock.Lock()
	defer f.offLock.Unlock()

	// innerReader set by first Read()
	if f.innerReader == nil {
		rAt, err := core.Open(f.innerFile, f.fs.vDb, f.fs.service, f.fs.debugLvl)
		if err != nil {
			return 0, err
		}
		f.innerReader = rAt
	}

	// lag fix?
	_, _ = f.innerReader.ReadAt(lagBuffer, f.innerOff) // TODO: WTF!

	// return
	n, err = f.innerReader.ReadAt(p, f.innerOff)
	f.innerOff += int64(n)
	return n, err
}

// Seek @see os.File
//
// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means relative
// to the current offset, and 2 means relative to the end. It returns the new offset
// and an error, if any. The behavior of Seek on a file opened with O_APPEND is not
// specified.
//
// If f is a directory, the behavior of Seek varies by operating system; you can seek
// to the beginning of the directory on Unix-like operating systems, but not on Windows.
func (f *_File) Seek(offset int64, whence int) (int64, error) {
	f.offLock.Lock()
	defer f.offLock.Unlock()

	// whence
	newPos := int64(0)
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = f.innerOff + offset
	case io.SeekEnd:
		newPos = f.innerFile.FileSize + offset
	}

	// check
	if newPos < 0 {
		newPos = 0
	}

	// set & return
	f.innerOff = newPos
	return f.innerOff, nil
}

// Readdir @see os.File
//
// Readdir reads the contents of the directory associated with file and returns
// a slice of up to n FileInfo values, as would be returned by Lstat, in directory
// order. Subsequent calls on the same file will yield further FileInfos.
//
// If n > 0, Readdir returns at most n FileInfo structures. In this case,
// if Readdir returns an empty slice, it will return a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdir returns all the FileInfo from the directory in a single slice.
// In this case, if Readdir succeeds (reads all the way to the end of the directory),
// it returns the slice and a nil error. If it encounters an error before the end of the
// directory, Readdir returns the FileInfo read until that point and a non-nil error.
func (f *_File) Readdir(count int) ([]os.FileInfo, error) {

	// get list
	ret := make([]os.FileInfo, 0, len(f.innerFile.FolderContent))
	for _, fc := range f.innerFile.FolderContent {
		vf := db.VirtFile{
			RelPath:  fc.RelPath,
			FileSize: 0, // need?
			MTime:    0, // need?
			IsDir:    fc.IsDir,
		}
		ret = append(ret, newFileInfo(vf))
	}

	// return
	if count <= 0 {
		// return all
		return ret, nil
	} else {
		if len(ret) < count {
			// return not enough
			return ret, io.EOF
		} else {
			// return count
			return ret[:count], nil
		}
	}
}

// Stat @see os.File
//
// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *_File) Stat() (os.FileInfo, error) {
	return newFileInfo(f.innerFile), nil
}

// ---------  not implemented (ReadOnly)  --------------------------------------------------------------------------- //

// Write @see os.File
//
// Write writes len(b) bytes to the File. It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *_File) Write(_ []byte) (n int, err error) {
	return 0, webdav.ErrForbidden // read only
}
