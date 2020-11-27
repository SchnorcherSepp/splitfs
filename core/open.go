package core

import (
	"errors"
	"github.com/SchnorcherSepp/splitfs/db"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"io"
	"log"
)

// Open returns a ReaderAt for the desired file.
// The data can be read directly in plain text.
// Smaller files are preferably read from a bundle.
func Open(file db.VirtFile, vDb db.Db, service interf.Service, debugLvl uint8) (interf.ReaderAt, error) {
	// check nil
	if service == nil {
		return nil, errors.New("service is nil")
	}

	// debug (0=off, 1=debug, 2=high)
	debug := debugLvl >= impl.DebugLow

	// TODO: VERBESSERUNG für den Zugriff auf Bundles:
	//       - cache ist nicht nil  (ist gesetzt)
	//       - ziel ist ein bundle
	//       - es wird wiederholt das glaiche bundle aufgerufen
	//       - die aufrufe sind ca in einem bereich (offset)
	//       - DANN ziehe ich einen großen zug in den cache rein!
	//       - nicht mehr als 50% des caches dabei verbrauchen!
	//       - max 50 MB

	// CASE 0: zero file (0 bytes)
	// -> impl.NewZeroReaderAt()
	//-----------------------------------------------------------
	if file.FileSize <= 0 || file.IsDir {
		if debug {
			log.Printf("DEBUG: %s/Open: ZeroReaderAt for '%s' with %d bytes", packageName, file.Name(), file.FileSize)
		}
		return impl.NewZeroReaderAt(), nil // --> EXIT CASE 0
	}

	// CASE 1: compressed file from a single source
	// a) bundle
	// b) part
	// -> newRamReaderAt()
	//-----------------------------------------------------------
	if len(file.Parts) == 1 && file.UseCompression {
		// source? (single file or part)
		sf, off, n, dataKey, err := bundleOrPart(file, vDb, service, debug)
		if err != nil {
			return nil, err // ERROR
		}

		// get readerAt
		rAt, err := service.ReaderAt(sf)
		if err != nil {
			return nil, err // ERROR
		}
		defer rAt.Close()

		// read all bytes
		data := make([]byte, n)
		n2, err := rAt.ReadAt(data, off)
		if err != nil {
			return nil, err // ERROR
		}
		data = data[:n2] // trim buffer

		// check read size
		if n != int64(n2) {
			return nil, io.ErrUnexpectedEOF // ERROR
		}

		// decrypt & decompress
		enc.CryptBytes(data, off, dataKey)
		data, err = enc.Decompress(data)
		if err != nil {
			return nil, err // ERROR
		}

		// return
		if debug {
			log.Printf("DEBUG: %s/Open: RamReaderAt for '%s' with %d bytes", packageName, file.Name(), file.FileSize)
		}
		return impl.NewRamReaderAt(data), nil // --> EXIT CASE 1
	}

	// CASE 2: regular file from a single source
	// a) bundle
	// b) part
	// -> NewSubReaderAt()
	//-----------------------------------------------------------
	if len(file.Parts) == 1 {
		// source? (single file or part)
		sf, off, n, dataKey, err := bundleOrPart(file, vDb, service, debug)
		if err != nil {
			return nil, err // ERROR
		}

		// cryptService
		keys := map[string][]byte{
			sf.Id(): dataKey,
		}
		cryptService := newCryptRService(service, keys)

		// encrypt and return
		if debug {
			log.Printf("DEBUG: %s/Open: SubReaderAt for '%s' with %d bytes", packageName, file.Name(), file.FileSize)
		}
		return impl.NewSubReaderAt(sf, cryptService, service.Cache(), debugLvl, off, n) // --> EXIT CASE 2
	}

	// CASE 3 (default): multi part file
	// -> NewMultiReaderAt
	//-----------------------------------------------------------

	// get all parts as storage file
	files := make([]interf.File, 0, len(file.Parts))
	keys := make(map[string][]byte)
	for _, part := range file.Parts {
		sf, err := service.Files().ByAttr(part.StorageName, part.StorageSize, part.StorageMd5)
		if err != nil {
			return nil, err // ERROR
		}
		files = append(files, sf)
		keys[sf.Id()] = part.CryptDataKey
	}

	// cryptService
	cryptService := newCryptRService(service, keys)

	// return (default)
	if debug {
		log.Printf("DEBUG: %s/Open: MultiReaderAt for '%s' with %d bytes and %d parts", packageName, file.Name(), file.FileSize, len(file.Parts))
	}
	return impl.NewMultiReaderAt(files, cryptService, service.Cache(), debugLvl) // --> EXIT CASE 3
}

// bundleOrPart returns the storage file in which the searched data is stored.
//
// Small files only have one part. In addition, the data can be contained in a bundle.
// This function tries to return the data from the bundle. If there is no bundle,
// or if there was an error, the single part is returned.
// This function is not for multi part files!
func bundleOrPart(file db.VirtFile, vDb db.Db, service interf.Service, debug bool) (sf interf.File, off int64, n int64, dataKey []byte, err error) {
	// nil check
	if service == nil {
		err = errors.New("service is nil")
		return // -> ERROR
	}
	if vDb.VFiles == nil {
		err = errors.New("db is nil")
		return // -> ERROR
	}

	// this function is not for multi part files
	if len(file.Parts) != 1 {
		err = errors.New("this function is not for multi part files")
		log.Printf("ERROR: %s/bundleOrPart: '%s': %v", packageName, file.RelPath, err)
		return // -> ERROR
	}
	part := file.Parts[0]

	// SET DEFAULT: use part[0]
	{ //-------------------------------------
		sf, err = service.Files().ByAttr(part.StorageName, part.StorageSize, part.StorageMd5)
		if err != nil {
			log.Printf("ERROR: %s/bundleOrPart: file not found in storage: '%s': %v", packageName, file.RelPath, err)
			return // -> ERROR
		}
		off = int64(0)
		n = part.StorageSize
		dataKey = part.CryptDataKey
	} //-------------------------------------

	// try bundle
	if file.AlsoInBundle != "" {

		// get bundle from db
		bundle, ok := vDb.Bundles[file.AlsoInBundle]
		if !ok {
			log.Printf("ERROR: %s/bundleOrPart: bundle link error: '%s'", packageName, file.RelPath)
			return // -> use DEFAULT
		}

		// get bundle storage file
		tmpSF, e := service.Files().ByAttr(bundle.StorageName, bundle.StorageSize, bundle.StorageMd5)
		if e != nil {
			log.Printf("ERROR: %s/bundleOrPart: bundle not found in storage: '%s': [%s, %s, %d]: %v", packageName, file.RelPath, bundle.StorageName, bundle.StorageMd5, bundle.StorageSize, e)
			return // -> use DEFAULT
		}

		// find position
		index, bOffset, ok := posInBundle(bundle, vDb, file)
		if !ok {
			// logging in posInBundle function
			return // -> use DEFAULT
		}

		// USE BUNDLE
		{ //-------------------------------------
			sf = tmpSF                    // update
			off = bOffset                 // update
			n = part.StorageSize          // no changes
			dataKey = bundle.CryptDataKey // update
			if debug {
				log.Printf("DEBUG: %s/bundleOrPart: get file '%s' from bundle '%s' (Index %d)", packageName, file.RelPath, sf.Name(), index)
			}
		} //-------------------------------------
	}

	// success
	return
}

// posInBundle returns the offset at which point the file data starts in the bundle.
// return  INDEX, OFFSET, OK
func posInBundle(bundle db.Bundle, vDb db.Db, target db.VirtFile) (int, int64, bool) {
	// nil check
	if bundle.Id() == "" || vDb.VFiles == nil || target.Id() == "" {
		log.Printf("ERROR: %s/posInBundle: nil check fail: bundle='%s', db=%v, target='%s'", packageName, bundle.Id(), vDb.VFiles != nil, target.Id())
		return 0, 0, false
	}

	// do your thing
	off := int64(0)
	for i, vFileId := range bundle.Content {

		// file id from bundle content list -> file
		vFile, ok := vDb.VFiles[vFileId]
		if !ok {
			log.Printf("ERROR: %s/posInBundle: bundle content link error: %s[%d]", packageName, bundle.StorageName, i)
			return 0, 0, false // ERROR
		}

		// check and get first part
		if len(vFile.Parts) != 1 {
			log.Printf("ERROR: %s/posInBundle: part check fail: %s[%d]", packageName, bundle.StorageName, i)
			return 0, 0, false // ERROR
		}
		part := vFile.Parts[0]

		// compare
		if vFile.Id() != target.Id() {
			// next file
			off += part.StorageSize
			continue
		} else {
			// SUCCESS
			return i, off, true
		}
	}
	// not found
	log.Printf("ERROR: %s/posInBundle: file not found in bundle: '%s'", packageName, target.RelPath)
	return 0, 0, false // ERROR
}
