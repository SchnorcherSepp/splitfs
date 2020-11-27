package webdav

/*
	IN THIS FILE: FileSystem implementation
		- update loop (db update)
		- FS: Open(), Stat()
		- no I/O implementations (@see file.go)
*/

import (
	"context"
	"github.com/SchnorcherSepp/splitfs/core"
	"github.com/SchnorcherSepp/splitfs/db"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"golang.org/x/net/webdav"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var _ webdav.FileSystem = (*_FileSystem)(nil)

// A FileSystem implements access to a collection of named files. The elements
// in a file path are separated by slash ('/', U+002F) characters, regardless
// of host operating system convention.
//
// Each method has the same semantics as the os package's function of the same
// name.
type _FileSystem struct {
	service  interf.Service
	dbKey    []byte
	debugLvl uint8

	vDb      db.Db
	dbFileId string // to detect db changes
	dbMux    *sync.RWMutex
}

// NewFileSystem creates a new webdav file system.
// It provides the plaintext data and accesses a storage in the background.
//
// 'service' is used to access the encrypted parts in the background.
// 'dbKey' is the key to the database file (stored online).
// 'debugLvl' controls the print output (@see impl.DebugOff, impl.DebugLow and impl.DebugHigh).
// 'updateInterval' in seconds controls how often database is updated in the background.
// The value 0 deactivates the update loop.
func NewFileSystem(service interf.Service, dbKey []byte, debugLvl uint8, updateInterval int) webdav.FileSystem {
	// check nil service
	if service == nil {
		service = impl.NewRamService(nil, impl.DebugOff) // dummy service
	}

	// build FileSystem
	fs := &_FileSystem{
		service:  service,
		dbKey:    dbKey,
		debugLvl: debugLvl,

		vDb:      db.NewDb(),
		dbFileId: "",
		dbMux:    new(sync.RWMutex),
	}

	// start update loop
	go fs.startUpdateLoop(updateInterval)
	go fs.startInitDbLoop(updateInterval) // runs 1 minute only

	// return
	return fs
}

// ------------------------------------------------------------------------------------------------------------------ //

// OpenFile @see os.OpenFile
//
// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (fs *_FileSystem) OpenFile(_ context.Context, relPath string, _ int, _ os.FileMode) (webdav.File, error) {
	fs.dbMux.RLock()         // R LOCK
	defer fs.dbMux.RUnlock() // R UNLOCK

	// path fix
	relPath = pathFix(relPath)

	// get VirtFile
	f, ok := fs.vDb.VFiles[relPath]
	if !ok {
		return nil, os.ErrNotExist
	}

	/*
		Webdav opens files and folders when listing the folder contents.
		The real open is called by the first read (@see file.go).
	*/

	// return webdav file
	return newFile(f, fs), nil
}

// Stat @see os.Stat
//
// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (fs *_FileSystem) Stat(_ context.Context, relPath string) (os.FileInfo, error) {
	fs.dbMux.RLock()         // R LOCK
	defer fs.dbMux.RUnlock() // R UNLOCK

	// path fix
	relPath = pathFix(relPath)

	// get VirtFile
	f, ok := fs.vDb.VFiles[relPath]
	if !ok {
		return nil, os.ErrNotExist
	}

	// return stat
	return newFileInfo(f), nil
}

// ---------  not implemented (ReadOnly)  --------------------------------------------------------------------------- //

// Mkdir @see os.Mkdir
//
// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (fs *_FileSystem) Mkdir(_ context.Context, _ string, _ os.FileMode) error {
	return webdav.ErrForbidden // read only
}

// RemoveAll @see os.RemoveAll
//
// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (fs *_FileSystem) RemoveAll(_ context.Context, _ string) error {
	return webdav.ErrForbidden // read only
}

// Rename @see os.Rename
//
// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (fs *_FileSystem) Rename(_ context.Context, _, _ string) error {
	return webdav.ErrForbidden // read only
}

// ---------  Helper  ----------------------------------------------------------------------------------------------- //

// pathFix change the path to a db friendly format.
// The root call '/' need to be '.' and other paths cannot begin or end with '/'.
func pathFix(relPath string) string {

	// trim '/' on both sides
	relPath = strings.Trim(relPath, "/")

	// root path fix ('/' -> '.')
	if relPath == "" {
		relPath = "." // set root
	}

	return relPath
}

// startUpdateLoop is started by NewFileSystem in its own goroutine.
// The loop run service.Update() every n seconds.
func (fs *_FileSystem) startUpdateLoop(updateInterval int) {
	// disable update loop
	if updateInterval < 1 {
		return
	}

	// start update loop
	log.Printf("INFO: %s/FileSystem: start Update loop with %d sec", packageName, updateInterval)
	for { // ---------------------------------------------------------------
		// inner update
		err := fs.service.Update()

		// log
		if err != nil {
			log.Printf("WARNING: %s/UpdateLoop: error: %v", packageName, err)
		} else {
			log.Printf("INFO: %s/UpdateLoop: successful", packageName)
		}

		// check for db updates
		fs.checkDb(false) // set new DB (thread save)

		// sleep (updateInterval)
		time.Sleep(time.Duration(updateInterval) * time.Second) // sleep
	} // -------------------------------------------------------------------
}

// startInitDbLoop is started by NewFileSystem in its own goroutine.
// The loop load the database every 5 seconds (that happens in the separate function checkDb).
// This loop ends after the (fast) update initialization is finished (~ 1 minutes)
func (fs *_FileSystem) startInitDbLoop(updateInterval int) {
	// disable update loop
	if updateInterval < 1 {
		return
	}

	// start update loop
	log.Printf("INFO: %s/FileSystem: start init DB loop", packageName)
	for i := 0; i < 6; i++ { // ----------------------------------------------
		// check for db updates
		if fs.checkDb(true) { // set new DB (thread save)
			break
		}

		// sleep (updateInterval)
		time.Sleep(5 * time.Second) // sleep
	} // -------------------------------------------------------------------
	log.Printf("INFO: %s/FileSystem: stop init DB loop", packageName)
}

// checkDb checks the database for changes. The check is based on the service file list (offline).
// If the database has to be updated, it will be downloaded (online connection).
func (fs *_FileSystem) checkDb(silence bool) bool {

	fs.dbMux.Lock()         // W LOCK
	defer fs.dbMux.Unlock() // W UNLOCK

	// get db file
	f, err := fs.service.Files().ByName(core.IndexName)
	if err != nil {
		if !silence {
			log.Printf("WARNING: %s/checkDb: %v", packageName, err)
		}
		return false // ERROR
	}

	// changes?
	// new file id -> db change!
	if fs.dbFileId == f.Id() && fs.dbFileId != "" {
		return false // do nothing
	}

	log.Printf("INFO: %s/Update: download db", packageName)

	// open reader to db file
	r, err := fs.service.Reader(f, 0)
	if err != nil {
		log.Printf("WARNING: %s/checkDb: %v", packageName, err)
		return false // ERROR
	}
	defer r.Close() // CLOSE

	// read db
	newDb, err := db.FromReader(r, fs.dbKey)
	if err != nil {
		log.Printf("WARNING: %s/checkDb: %v", packageName, err)
		return false // ERROR
	}

	// success:
	// set new db and return
	fs.vDb = newDb
	fs.dbFileId = f.Id()
	return true
}
