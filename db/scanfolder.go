package db

import (
	"fmt"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"golang.org/x/text/unicode/norm"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FromScan scan a root folder and return a new db.
// Bundles and links are removed.
func FromScan(rootPath string, oldDB Db, debugLvl uint8, keyFile *enc.KeyFile) (newDB Db, changed bool, summary string, retErr error) {
	// debug (0=off, 1=debug, 2=high)
	debug := debugLvl >= impl.DebugLow

	// replace oldDB with clone (first level)
	clone := NewDb()
	if oldDB.VFiles != nil {
		for k, v := range oldDB.VFiles {
			clone.VFiles[k] = v
		}
	}
	oldDB = clone

	// reset bundles
	resetOldBundles(&oldDB)

	// init
	countNewOrUpdate := 0
	newDB = NewDb()

	// Walk
	retErr = filepath.Walk(rootPath, func(absPath string, info os.FileInfo, err error) error {
		// WalkFunc errors
		if err != nil {
			return err
		}

		// relative path
		relPath, err := filepath.Rel(rootPath, absPath)
		if err != nil {
			return err
		}

		// UTF8 FIX: Text normalization
		// https://blog.golang.org/normalization
		relPath = norm.NFC.String(relPath)
		// WINDOWS/LINUX FIX: path separator = '/'
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		// get element attributes
		isDir := info.IsDir()
		mtime := info.ModTime().Unix()
		size := info.Size()

		// WINDOWS/LINUX FIX: set folder size to 0
		if isDir {
			size = 0
		}

		// if folder: get folder content
		var dirEntries []FolderEl
		if isDir {
			dirEntries, err = getDirEntries(absPath)
			if err != nil {
				return err
			}
		}

		// find element in old DB
		e, ok := oldDB.VFiles[relPath]

		// element not found (new) OR element changed
		if !ok || e.FileSize != size || e.IsDir != isDir || e.MTime != mtime {
			countNewOrUpdate++
			changed = true

			detail := ""
			if !isDir {
				start := time.Now()
				// is file -> scan
				vf, err := ScanFile(absPath, relPath, keyFile)
				if err != nil {
					return err
				}
				e = vf
				// write detail
				var sinceInSec = float64(time.Since(start)) / float64(time.Second)
				if sinceInSec < 0.001 {
					sinceInSec = 0.001
				}
				var sizeInMb = float64(e.FileSize) / (1024 * 1024)
				detail = fmt.Sprintf("\t[%.2f MB/s]", sizeInMb/sinceInSec)

			} else {
				// is folder -> create
				e = VirtFile{ // override db element (dir)
					RelPath:       relPath,
					FileSize:      0,
					MTime:         mtime,
					IsDir:         isDir,
					FolderContent: dirEntries,
				}
			}

			if debug {
				if !ok {
					log.Printf("DEBUG: %s/ScanFolder: new: '%s'%s", packageName, relPath, detail)
				} else {
					log.Printf("DEBUG: %s/ScanFolder: changed: '%s'%s", packageName, relPath, detail)
				}
			}
		}

		// FIX: Always update the folder content. If no file changes have been made,
		// the database will not be updated. If the database is updated, then the
		// folder content will also be up to date.
		e.FolderContent = dirEntries

		// Delete exist elements from old db. At the end we can detect old item no longer available.
		// If there is anything left, something has changed!
		delete(oldDB.VFiles, relPath)

		// Write the element to the new database and exit the walk function
		newDB.VFiles[relPath] = e
		return nil
	})

	// finale changed?
	if len(oldDB.VFiles) > 0 {
		changed = true
	}

	// statistic
	summary = fmt.Sprintf("SCAN: error=%v, sum=%d, changed=%v, newOrUpdate=%d, removed=%d", retErr, len(newDB.VFiles), changed, countNewOrUpdate, len(oldDB.VFiles))
	if debug {
		log.Printf("DEBUG: %s/ScanFolder: %s", packageName, summary)
	}
	return
}

// ----------  HELPER  -----------------------------------------------------------------------------------------------//

// getDirEntries return folder content
func getDirEntries(dir string) ([]FolderEl, error) {
	// open folder
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// read folder list
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	// sort list
	sort.Strings(names)

	// return stuff
	retList := make([]FolderEl, 0, len(names))
	for _, name := range names {

		// sub-element is file oder folder
		absPath := filepath.Join(dir, name)
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, err
		}
		isDir := info.IsDir()

		// UTF8 FIX: Text normalization
		// https://blog.golang.org/normalization
		name = norm.NFC.String(name)
		// WINDOWS/LINUX FIX: path separator = '/'
		name = strings.ReplaceAll(name, "\\", "/")

		// append
		retList = append(retList, FolderEl{
			RelPath: name,
			IsDir:   isDir,
		})
	}
	return retList, nil
}
