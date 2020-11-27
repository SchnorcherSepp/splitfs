package core

import (
	"errors"
	"fmt"
	"github.com/SchnorcherSepp/splitfs/db"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"log"
	"strings"
)

// Clean removes no longer referenced data from the storage.
// If the database does not contain any bundles, all bundles are ignored in storage (BundleMode=off).
// If the try flag is true, no data is deleted.
func Clean(vDB db.Db, service interf.Service, try bool, debugLvl uint8) error {
	// debug (0=off, 1=debug, 2=high)
	debug := debugLvl >= impl.DebugLow

	// nil check
	if service == nil {
		return errors.New("service is nil")
	}

	// update service list
	if err := service.Update(); err != nil {
		return err
	}

	// get all files
	unknownParts, unknownBundles, _ := unknown(vDB, service, debug)
	duplicates := duplicates(service, debug)

	// build remove list
	removeList := make([]interf.File, 0)

	removeList = append(removeList, unknownParts...)
	removeList = append(removeList, duplicates...)

	bundleMode := len(vDB.Bundles) > 0
	if bundleMode {
		log.Printf("INFO: %s/Clean: bundle mode on", packageName)
		removeList = append(removeList, unknownBundles...)
	}

	// log 'try' mode
	if try {
		log.Printf("INFO: %s/Clean: try mode on: nothing is deleted", packageName)
	}

	// REMOVE
	for i, f := range removeList {
		// log
		if debug {
			log.Printf("DEBUG: %s/Clean: remove [%d/%d] '%s': id=%s, size=%d", packageName, i+1, len(removeList), f.Name(), f.Id(), f.Size())
		}
		// remove
		if !try {
			if err := service.Trash(f); err != nil {
				return err
			}
		}
	}

	// success
	return nil
}

// unknown returns all online files that are not in the database
func unknown(vDB db.Db, service interf.Service, debug bool) (unknownParts, unknownBundles, unknownRest []interf.File) {
	unknownParts = make([]interf.File, 0)
	unknownBundles = make([]interf.File, 0)
	unknownRest = make([]interf.File, 0)

	// get lists
	dbParts := allDbParts(vDB)
	onlineParts, onlineBundles, onlineRest := allOnlineParts(service)

	// 1) onlineParts
	for _, op := range onlineParts {
		add := true
		// find in db
		for _, dp := range dbParts {
			if op.Name() == dp.StorageName {
				if op.Size() == dp.StorageSize && op.Md5() == dp.StorageMd5 {
					add = false
					continue
				} else {
					if debug {
						log.Printf("DEBUG: %s/unknown: defect part '%s': %d != %d or %s != %s", packageName, op.Name(), op.Size(), dp.StorageSize, op.Md5(), dp.StorageMd5)
					}
				}
			}
		}
		// unknown part
		if add {
			unknownParts = append(unknownParts, op)
		}
	}

	// 2) onlineBundles
	for _, ob := range onlineBundles {
		add := true
		// find in db
		for _, dp := range dbParts {
			if ob.Name() == dp.StorageName {
				if ob.Size() == dp.StorageSize {
					add = false
					continue
				} else {
					if debug {
						log.Printf("DEBUG: %s/unknown: defect bundle '%s': %d != %d", packageName, ob.Name(), ob.Size(), dp.StorageSize)
					}
				}
			}
		}
		// unknown bundle
		if add {
			unknownBundles = append(unknownBundles, ob)
		}
	}

	// 3) rest
	for _, f := range onlineRest {
		add := true
		// valid files
		if f.Name() == IndexName {
			add = false
			continue
		}
		// unknown file
		if add {
			unknownRest = append(unknownRest, f)
		}
	}

	// debug & return
	knownParts := len(onlineParts) - len(unknownParts)
	knownBundles := len(onlineBundles) - len(unknownBundles)
	knownRest := len(onlineRest) - len(unknownRest)
	log.Printf("INFO: %s/unknown: Parts:   known=%d, unknown=%d, online=%d", packageName, knownParts, len(unknownParts), len(onlineParts))
	log.Printf("INFO: %s/unknown: Bundles: known=%d, unknown=%d, online=%d", packageName, knownBundles, len(unknownBundles), len(onlineBundles))
	log.Printf("INFO: %s/unknown: Rest:    known=%d, unknown=%d, online=%d", packageName, knownRest, len(unknownRest), len(onlineRest))
	return
}

// allDbParts extracts all parts from db.
func allDbParts(vDB db.Db) []db.VFilePart {
	var allParts = make(map[string]db.VFilePart)

	// get all file parts
	for _, file := range vDB.VFiles {
		for _, part := range file.Parts {
			key := part.StorageName + "|" + part.StorageMd5
			allParts[key] = part
		}
	}

	// get all bundle parts
	for _, bundle := range vDB.Bundles {
		var part = bundle.VFilePart
		key := part.StorageName + "|" + part.StorageMd5
		allParts[key] = part
	}

	// return list
	list := make([]db.VFilePart, 0, len(allParts))
	for _, p := range allParts {
		list = append(list, p)
	}
	return list
}

// allOnlineParts extracts all parts from Service (online).
func allOnlineParts(service interf.Service) (parts, bundles, rest []interf.File) {
	parts = make([]interf.File, 0)
	bundles = make([]interf.File, 0)
	rest = make([]interf.File, 0)

	// get all files
	for _, file := range service.Files().All() {

		// part
		if len(file.Name()) == 128 {
			parts = append(parts, file)
			continue
		}

		// bundle
		if len(file.Name()) == 130 && file.Name()[0] == 'B' && file.Name()[1] == '_' {
			bundles = append(bundles, file)
			continue
		}

		// rest
		rest = append(rest, file)
	}
	return
}

// duplicates finds all duplicates (same name, same size, same hash)
func duplicates(service interf.Service, debug bool) []interf.File {
	dup := make(map[string][]interf.File)

	// all files
	for _, f := range service.Files().All() {
		key := fmt.Sprintf("%s|%d|%s", f.Name(), f.Size(), f.Md5())
		// get list
		list := dup[key]
		if list == nil {
			list = make([]interf.File, 0)
		}
		// add to list
		list = append(list, f)
		// set list
		dup[key] = list
	}

	// remove single files
	for key, list := range dup {
		if len(list) <= 1 {
			delete(dup, key)
		}
	}

	// debug
	if debug {
		for key, list := range dup {
			key = strings.ReplaceAll(key, "|", "\t")
			log.Printf("DEBUG: %s/duplicates: %dx %s", packageName, len(list), key)
		}
	}
	if len(dup) > 0 {
		log.Printf("WARNING: %s/duplicates: %d duplicates found", packageName, len(dup))
	}

	// return as list
	ret := make([]interf.File, 0, len(dup))
	for _, list := range dup {
		for i := 1; i < len(list); i++ { // skip index 0
			ret = append(ret, list[i])
		}
	}
	return ret
}
