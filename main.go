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
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
)

// version is set by `go build`
var version = "<version>"

// CLI commands (see https://github.com/alecthomas/kong)
var CLI struct {
	Debug int `short:"v" type:"counter" help:"Enable debug mode (-v for DebugLow, -vv for DebugHigh)."`

	Version struct {
	} `cmd help:"Show the program version."`

	Oauth struct {
		ClientFile string `short:"c" type:"path" default:"client.json" help:"The identifier for a app, to use the google api."`
		TokenFile  string `short:"t" type:"path" default:"token.json"  help:"Token for access to your gdrive."`
		// optional
		ReadOnly bool `short:"r" help:"Requests only read rights (no upload possible)."`
	} `cmd help:"Create the Google OAuth 2.0 files (ClientFile and TokenFile)."`

	Keygen struct {
		KeyFile string `short:"k" type:"path" default:"key.dat" help:"Path to the key file (must not exist)."`
	} `cmd help:"Creates a new key file (used for file encryption)."`

	Scan struct {
		RootDir string `short:"o" type:"path" default:"/data"     help:"Path to the folder with the plain text files (becomes the root directory)"`
		DbFile  string `short:"d" type:"path" default:"index.db2" help:"Path to the db file."`
		KeyFile string `short:"k" type:"path" default:"key.dat"   help:"Path to the key file."`
		// optional
		Force    bool `short:"f" help:"Forces a scan even if the content has not changed."`
		NoBundle bool `short:"n" help:"Bundles small files into large files for faster read access."`
	} `cmd help:"Scan a folder and create/update an encrypted database file."`

	Upload struct {
		RootDir    string `short:"o" type:"path" default:"/data"       help:"Path to the folder with the plain text files (becomes the root directory)"`
		DbFile     string `short:"d" type:"path" default:"index.db2"   help:"Path to the db file."`
		KeyFile    string `short:"k" type:"path" default:"key.dat"     help:"Path to the key file."`
		ClientFile string `short:"c" type:"path" default:"client.json" help:"The identifier for a app, to use the google api."`
		TokenFile  string `short:"t" type:"path" default:"token.json"  help:"Token for access to your gdrive."`
		CacheFile  string `short:"a" type:"path" default:"cache.dat"   help:"The online index file to speed up the program start."`
		// optional
		Force        bool   `short:"f" help:"Forces a scan/upload even if the content has not changed."`
		NoBundle     bool   `short:"n" help:"Bundles small files into large files for faster read access."`
		SkipFullInit bool   `short:"s" help:"Accelerates the program start with many files. (Experimental!)"`
		Cleanup      bool   `short:"l" help:"Deletes files that are no longer needed online after the upload. (WARNING: Do not use this mode regularly!)"`
		TryCleanup   bool   `short:"y" help:"Switches the -c cleanup mode to 'log only' and does not delete any files."`
		FolderID     string `short:"i" default:"root" help:"The google drive FolderID with the storage files."`
	} `cmd help:"Saves the local files encrypted in the online folder."`

	Webdav struct {
		UserFile   string `short:"u" type:"path" default:"webdav.users" help:"Path to the file with usernames and password hashes."`
		KeyFile    string `short:"k" type:"path" default:"key.dat"      help:"Path to the key file."`
		ClientFile string `short:"c" type:"path" default:"client.json"  help:"The identifier for a app, to use the google api."`
		TokenFile  string `short:"t" type:"path" default:"token.json"   help:"Token for access to your gdrive."`
		CacheFile  string `short:"a" type:"path" default:"cache.dat"    help:"The online index file to speed up the program start."`
		// optional
		CacheSizeMB    int    `short:"m" default:"500"           help:"The buffer in RAM enables high-performance random read access. (Don't use all of your memory!)"`
		LocalAddr      string `short:"l" default:":8080"         help:"The local server address like '1.2.3.4:8080' or '[::1]:443'."`
		FolderID       string `short:"i" default:"root"          help:"The google drive FolderID with the storage files."`
		UseTLS         bool   `short:"e"                         help:"Encrypt connection with TLS."`
		Cert           string `short:"q" default:"fullchain.pem" help:"Path to the server certificate."`
		CertKey        string `short:"p" default:"privkey.pem"   help:"Path to the server certificate key."`
		UpdateInterval int    `short:"x" default:"300"           help:"The database is checked for changes every n seconds."`
	} `cmd help:"Starts a WebDav server to access the files online."`

	Adduser struct {
		Username   string `arg help:"WebDav username (user must not yet exist)."`
		Password   string `arg help:"Password (saved as a bcrypt hash)"`
		PathPrefix string `arg help:"prefix for accessible resources (separated with ':')."`
		// optional
		UserFile string `short:"u" type:"path" default:"webdav.users" help:"Path to the file with usernames and password hashes."`
	} `cmd help:"Adds another webdav user to the user file."`
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
			fmt.Printf("[FATAL ERROR] %v\n", err)
			os.Exit(21)
		}
		break

	case "keygen":
		err := enc.CreateKeyFile(CLI.Keygen.KeyFile)
		if err != nil {
			fmt.Printf("[FATAL ERROR] %v\n", err)
			os.Exit(31)
		}
		break

	case "scan":
		debug := uint8(CLI.Debug)
		a := CLI.Scan
		upload(true, debug, false, "", "", a.KeyFile, "", "", a.DbFile, a.RootDir, a.Force, !a.NoBundle, false, true)
		break

	case "upload":
		debug := uint8(CLI.Debug)
		a := CLI.Upload
		upload(false, debug, a.SkipFullInit, a.ClientFile, a.TokenFile, a.KeyFile, a.FolderID, a.CacheFile, a.DbFile, a.RootDir, a.Force, !a.NoBundle, a.Cleanup, a.TryCleanup)
		break

	case "webdav":
		debug := uint8(CLI.Debug)
		a := CLI.Webdav
		startWebdav(debug, a.ClientFile, a.TokenFile, a.KeyFile, a.FolderID, a.CacheFile, a.LocalAddr, a.UserFile, a.CacheSizeMB, a.UseTLS, a.Cert, a.CertKey, a.UpdateInterval)
		break

	case "adduser":
		a := CLI.Adduser
		addUser(a.Username, a.Password, a.PathPrefix, a.UserFile)
		break

	default:
		fmt.Printf("[FATAL ERROR] command not implemented: '%s'\n", ctx.Command())
		os.Exit(99)
	}
}

//-##################################################################################################################-//

func upload(scanOnly bool, debugLvl uint8, skipFullInit bool, clientStr, tokenStr, keyStr, folderId, cacheStr, dbStr, rootStr string, forceFlag, bundleFlag, cleanUpFlag, cleanUpSimulation bool) {

	// load keyfile
	keyFile, err := enc.LoadKeyFile(keyStr)
	if err != nil {
		fmt.Printf("[FATAL ERROR] %v\n", err)
		os.Exit(501)
	}

	// load db (if exist)
	oldDb, err := db.FromFile(dbStr, keyFile.IndexKey())
	if err != nil {
		println(err) // WARNING: NO EXIT!
	}

	// SCAN DIR
	newDb, change, _, err := db.FromScan(rootStr, oldDb, debugLvl, keyFile)
	if err != nil {
		fmt.Printf("[FATAL ERROR] %v\n", err)
		os.Exit(502)
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
			fmt.Printf("[FATAL ERROR] %v\n", err)
			os.Exit(503)
		}

		// build service for upload
		service := gdrive.NewGService(folderId, cacheStr, skipFullInit, oauth, nil, debugLvl)

		// UPLOAD files & db
		err = core.Upload(rootStr, newDb, keyFile.IndexKey(), service, debugLvl)
		if err != nil {
			fmt.Printf("[FATAL ERROR] %v\n", err)
			os.Exit(504)
		}

		// unnecessary files online (optional)
		if cleanUpFlag {
			err = service.Update()
			if err != nil {
				fmt.Printf("[FATAL ERROR] %v\n", err)
				os.Exit(505)
			}
			err = core.Clean(newDb, service, cleanUpSimulation, debugLvl)
			if err != nil {
				fmt.Printf("[FATAL ERROR] %v\n", err)
				os.Exit(506)
			}
		}
	}
	//-----------------------------------------------------

	// save changed db local
	err = db.ToFile(newDb, keyFile.IndexKey(), dbStr)
	if err != nil {
		fmt.Printf("[FATAL ERROR] %v\n", err)
		os.Exit(507)
	}
}

func startWebdav(debugLvl uint8, clientStr, tokenStr, keyStr, folderId, cacheStr, lAddr, userDbStr string, cacheSizeMB int, useTLS bool, certStr, certKeyStr string, updateInterval int) {

	// check free ram
	checkFreeRam(cacheSizeMB)

	// load keyfile
	keyFile, err := enc.LoadKeyFile(keyStr)
	if err != nil {
		fmt.Printf("[FATAL ERROR] %v\n", err)
		os.Exit(601)
	}

	// build oauth
	oauth, err := gdrive.OAuth(clientStr, tokenStr, true)
	if err != nil {
		fmt.Printf("[FATAL ERROR] %v\n", err)
		os.Exit(602)
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
		fmt.Printf("[DEBUG] %v\n", err) // SOFT FAIL
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

func addUser(username, password, pathPrefix string, userFile string) {

	// prepare username
	username = strings.ReplaceAll(username, "\n", "") // remove new line
	username = strings.ReplaceAll(username, "\t", "") // remove tab
	username = strings.ReplaceAll(username, " ", "")  // remove space
	username = strings.TrimSpace(username)            // trim space

	// prepare password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		fmt.Printf("[FATAL ERROR] %v\n", err)
		os.Exit(701)
	}

	// prepare prefix
	strings.ReplaceAll(pathPrefix, "\n", "") // new line

	//-------------------------------------------------------

	var users string

	// read user file
	if _, err := os.Stat(userFile); err == nil {
		// read all
		b, err := ioutil.ReadFile(userFile)
		if err != nil {
			fmt.Printf("WARNING: %v\n", err)
		} else {
			users = string(b)
		}
	}

	// add
	line := username + ":" + string(hash) + ":" + pathPrefix
	users += "\n" + line

	// write file
	err = ioutil.WriteFile(userFile, []byte(users), 0600)
	if err != nil {
		fmt.Printf("[FATAL ERROR] %v\n", err)
		os.Exit(702)
	}
}
