package main

import (
	"fmt"
	"github.com/SchnorcherSepp/splitfs/core"
	"github.com/SchnorcherSepp/splitfs/db"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	"github.com/SchnorcherSepp/splitfs/webdav"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"github.com/SchnorcherSepp/storage/gdrive"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"github.com/alecthomas/kong"
	"github.com/mackerelio/go-osstat/memory"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"os"
	"path"
	"runtime"
)

// version is set by `go build`
var version = "<version>"

// CLI commands (see https://github.com/alecthomas/kong)
var CLI struct {
	Debug int `short:"v" type:"counter" help:"Enable debug mode (-v for DebugLow, -vv for DebugHigh)."`

	Version struct {
	} `cmd help:"Show the program version."`

	Oauth struct {
		ReadOnly bool `short:"r"  help:"Requests only read rights (no upload possible)."`
		//-----------------
		ClientFile string `arg type:"path"  help:"The identifier for a app, to use the google api."`
		TokenFile  string `arg type:"path"  help:"Token for access to your gdrive."`
	} `cmd help:"Create the Google OAuth 2.0 files (ClientFile and TokenFile)."`

	Keygen struct {
		KeyFile string `arg type:"path"  help:"Path to the key file (must not exist)."`
	} `cmd help:"Creates a new key file (used for file encryption)."`

	Scan struct {
		Force  bool `short:"f"  help:"Forces a scan even if the content has not changed."`
		Bundle bool `short:"b"  help:"Bundles small files into large files for faster read access."`
		//-----------------
		KeyFile string `arg type:"existingfile"  help:"Path to the key file."`
		DbFile  string `arg type:"path"          help:"Path to the db file."`
		RootDir string `arg type:"existingdir"   help:"Path to the folder with the plain text files (becomes the root directory)"`
	} `cmd help:"Scan a folder and create/update an encrypted database file."`

	Upload struct {
		Force        bool `short:"f"  help:"Forces a scan/upload even if the content has not changed."`
		Bundle       bool `short:"b"  help:"Bundles small files into large files for faster read access."`
		SkipFullInit bool `short:"s"  help:"Accelerates the program start with many files. (Experimental!)"`
		Cleanup      bool `short:"c"  help:"Deletes files that are no longer needed online after the upload. (WARNING: Do not use this mode regularly!)"`
		TryCleanup   bool `short:"t"  help:"Switches the -c cleanup mode to 'log only' and does not delete any files."`
		//-----------------
		KeyFile string `arg type:"existingfile"  help:"Path to the key file."`
		DbFile  string `arg type:"path"          help:"Path to the db file."`
		RootDir string `arg type:"existingdir"   help:"Path to the folder with the plain text files (becomes the root directory)"`
		//-----------------
		ClientFile string `arg type:"existingfile"  help:"The identifier for a app, to use the google api."`
		TokenFile  string `arg type:"existingfile"  help:"Token for access to your gdrive."`
		CacheFile  string `arg type:"path"          help:"The online index file to speed up the program start."`
		FolderID   string `arg                      help:"The google drive FolderID with the storage files (Default 'root')."`
	} `cmd help:"Saves the local files encrypted in the online folder."`

	Webdav struct {
		LocalAddr      string `short:"l" default:":8080"  help:"The local server address like '1.2.3.4:8080' or '[::1]:443'."`
		UseTLS         bool   `short:"t"  help:"Encrypt connection with TLS."`
		Cert           string `short:"c" type:"Path to the server certificate."`
		CertKey        string `short:"k" type:"Path to the server certificate key."`
		UpdateInterval int    `short:"u" default:"300"  help:"The database is checked for changes every n seconds (default 300)."`
		//-----------------
		KeyFile string `arg type:"existingfile"  help:"Path to the key file."`
		//-----------------
		ClientFile string `arg type:"existingfile"  help:"The identifier for a app, to use the google api."`
		TokenFile  string `arg type:"existingfile"  help:"Token for access to your gdrive."`
		CacheFile  string `arg type:"path"          help:"The online index file to speed up the program start."`
		FolderID   string `arg                      help:"The google drive FolderID with the storage files (Default 'root')."`
		//-----------------
		UserFile    string `arg type:"existingfile"  help:"Path to the file with usernames and password hashes."`
		CacheSizeMB int    `arg                      help:"The buffer in RAM enables high-performance random read access. (Don't use all of your memory!)"`
	} `cmd help:"Starts a WebDav server to access the files online."`
}

func main() {
	description := "The program synchronizes local files with Google Drive and makes them available."
	ctx := kong.Parse(&CLI, kong.UsageOnError(), kong.Description(description))
	switch ctx.Selected().Name {

	case "version":
		fmt.Printf("%s %s\n", path.Base(os.Args[0]), version)
		fmt.Printf("%s %s/%s (%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)
		break

	case "oauth":
		_, err := gdrive.OAuth(CLI.Oauth.ClientFile, CLI.Oauth.TokenFile, CLI.Oauth.ReadOnly)
		if err != nil {
			panic(err)
		}
		break

	case "keygen":
		err := enc.CreateKeyFile(CLI.Keygen.KeyFile)
		if err != nil {
			panic(err)
		}
		break

	case "scan":
		debug := uint8(CLI.Debug)
		a := CLI.Scan
		upload(true, debug, false, "", "", a.KeyFile, "", "", a.DbFile, a.RootDir, a.Force, a.Bundle, false, true)
		break

	case "upload":
		debug := uint8(CLI.Debug)
		a := CLI.Upload
		upload(false, debug, a.SkipFullInit, a.ClientFile, a.TokenFile, a.KeyFile, a.FolderID, a.CacheFile, a.DbFile, a.RootDir, a.Force, a.Bundle, a.Cleanup, a.TryCleanup)
		break

	case "webdav":
		debug := uint8(CLI.Debug)
		a := CLI.Webdav
		startWebdav(debug, a.ClientFile, a.TokenFile, a.KeyFile, a.FolderID, a.CacheFile, a.LocalAddr, a.UserFile, a.CacheSizeMB, a.UseTLS, a.Cert, a.CertKey, a.UpdateInterval)
		break

	default:
		panic(fmt.Sprintf("command not implemented: '%s'", ctx.Command()))
	}
}

//-##################################################################################################################-//

func upload(scanOnly bool, debugLvl uint8, skipFullInit bool, clientStr, tokenStr, keyStr, folderId, cacheStr, dbStr, rootStr string, forceFlag, bundleFlag, cleanUpFlag, cleanUpSimulation bool) {

	// load keyfile
	keyFile, err := enc.LoadKeyFile(keyStr)
	if err != nil {
		panic(err)
	}

	// load db (if exist)
	oldDb, err := db.FromFile(dbStr, keyFile.IndexKey())
	if err != nil {
		println(err) // WARNING: NO EXIT!
	}

	// SCAN DIR
	newDb, change, _, err := db.FromScan(rootStr, oldDb, debugLvl, keyFile)
	if err != nil {
		panic(err)
	}
	if !change && !forceFlag {
		// no change AND no upload-force
		return // --> EXIT
	}

	// make bundles (optional)
	if bundleFlag {
		newDb.MakeBundles(keyFile, debugLvl)
	}

	//-----------------------------------------------------
	if !scanOnly {
		// build oauth
		oauth, err := gdrive.OAuth(clientStr, tokenStr, false)
		if err != nil {
			panic(err)
		}

		// build service for upload
		service := gdrive.NewGService(folderId, cacheStr, skipFullInit, oauth, nil, debugLvl)

		// UPLOAD files & db
		err = core.Upload(rootStr, newDb, keyFile.IndexKey(), service, debugLvl)
		if err != nil {
			panic(err)
		}

		// unnecessary files online (optional)
		if cleanUpFlag {
			err = service.Update()
			if err != nil {
				panic(err)
			}
			err = core.Clean(newDb, service, cleanUpSimulation, debugLvl)
			if err != nil {
				panic(err)
			}
		}
	}
	//-----------------------------------------------------

	// save changed db local
	err = db.ToFile(newDb, keyFile.IndexKey(), dbStr)
	if err != nil {
		panic(err)
	}
}

func startWebdav(debugLvl uint8, clientStr, tokenStr, keyStr, folderId, cacheStr, lAddr, userDbStr string, cacheSizeMB int, useTLS bool, certStr, certKeyStr string, updateInterval int) {

	// check free ram
	checkFreeRam(cacheSizeMB)

	// load keyfile
	keyFile, err := enc.LoadKeyFile(keyStr)
	if err != nil {
		panic(err)
	}

	// build oauth
	oauth, err := gdrive.OAuth(clientStr, tokenStr, true)
	if err != nil {
		panic(err)
	}

	// build sector cache for random read access
	var sectorCache interf.Cache
	if cacheSizeMB > 0 {
		sectorCache = impl.NewCache(cacheSizeMB)
	}

	// build service for READ ACCESS
	service := gdrive.NewGService(folderId, cacheStr, false, oauth, sectorCache, debugLvl)

	// RUN webdav server
	fs := webdav.NewFileSystem(service, keyFile.IndexKey(), debugLvl, updateInterval)
	err = webdav.Serve(lAddr, useTLS, certStr, certKeyStr, fs, userDbStr, debugLvl)
	if err != nil {
		println(err) // SOFT FAIL
	}
}

// checkFreeRam check and print the ram usage.
// The program crashes if the cache does not have enough memory available.
// cacheSizeMB + 20% is needed!
func checkFreeRam(cacheSizeMB int) {
	// check free ram
	mem, err := memory.Get()
	if err == nil {
		// calc
		totalMB := int(mem.Total / (1024 * 1024))
		usedMB := int(mem.Used / (1024 * 1024))
		freeMB := int(mem.Free / (1024 * 1024))

		// limits
		limit1 := int(float64(cacheSizeMB)*1.2 + 200)
		limit2 := cacheSizeMB*2 + 200

		if freeMB < limit1 {
			// too small
			fmt.Printf("WARNING: NOT ENOUGH FREE MEMORY!\n")
		} else if freeMB < limit2 {
			// warning
			fmt.Printf("Keep an eye on memory usage!\n")
		} else {
			// OK
			return // print nothing
		}

		// print ram stats
		p := message.NewPrinter(language.German)
		_, _ = p.Printf("+ memory total: %d MB\n", totalMB)
		_, _ = p.Printf("+ memory used: %d MB\n", usedMB)
		_, _ = p.Printf("+ memory free: %d MB\n", freeMB)
		_, _ = p.Printf("+ cache size: %d MB\n", cacheSizeMB)
		_, _ = p.Printf("+ free memory after cache: %d MB\n", freeMB-cacheSizeMB)
	}
}
