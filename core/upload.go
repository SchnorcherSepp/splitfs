package core

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/SchnorcherSepp/splitfs/db"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
)

// Upload uploads all files that are defined in the database.
// If bundles are created (in db), they are also uploaded (@see db.BundlePrefix).
// Finally the database is also uploaded (@see IndexName).
func Upload(rootPath string, vDB db.Db, dbKey []byte, service interf.Service, debugLvl uint8) error {
	// debug (0=off, 1=debug, 2=high)
	debug := debugLvl >= impl.DebugLow

	// update service list to prevent double uploads
	if err := service.Update(); err != nil {
		log.Printf("ERROR: %s/Upload: update file list: %v", packageName, err)
		return err
	}
	// upload part list
	// saves all uploaded parts to prevent double uploads
	var uploadedParts = make(map[string]db.VFilePart)

	// upload all vFiles
	//-------------------------
	// map value to list
	list := make([]db.VirtFile, 0, len(vDB.VFiles))
	for _, vFile := range vDB.VFiles {
		list = append(list, vFile)
	}
	// sort list
	sort.Slice(list, func(i, j int) bool {
		return list[i].RelPath < list[j].RelPath
	})
	// upload
	for _, vFile := range list {
		if err := uploadFile(path.Join(rootPath, vFile.RelPath), vFile, service, uploadedParts, debug); err != nil {
			return err
		}
	}

	// upload bundles (OPTIONAL)
	if err := uploadBundle(rootPath, vDB, service, uploadedParts, debug); err != nil {
		return err
	}

	// upload db file
	if err := uploadDb(vDB, dbKey, service, debug); err != nil {
		return err
	}

	// success
	return nil
}

// ----------  HELPER  -----------------------------------------------------------------------------------------------//

// exists checks whether a file exists on storage.
func exists(part db.VFilePart, service interf.Service, uploadedParts map[string]db.VFilePart) bool {

	// check local list
	for _, up := range uploadedParts {
		if part.StorageName == up.StorageName && part.StorageSize == up.StorageSize && part.StorageMd5 == up.StorageMd5 {
			return true // part exist
		}
	}

	// check service
	_, err := service.Files().ByAttr(part.StorageName, part.StorageSize, part.StorageMd5)
	if err == nil {
		return true // part exist
	}

	// part not found
	return false
}

// seek sets the offset for the next Read on file to the part start
func seek(fh *os.File, partNo int) (int64, error) {
	// calc offset
	offset := int64(partNo) * db.PartSize

	// seek
	o, err := fh.Seek(offset, 0)
	if err != nil {
		return 0, err // seek error
	}

	// check offset
	if o != offset {
		return o, errors.New("seek error: wrong offset") // seek error
	}

	// success
	return offset, nil
}

// uploadPart uploads a part.
// Data are optionally compressed.
// Data are encrypted.
func uploadPart(fh *os.File, partNo int, useCompr bool, cryptKey []byte, storageName string, storageSize int64, service interf.Service) error {

	// There is no second part with active compression!
	if useCompr && partNo > 0 {
		return errors.New("can't compress second part")
	}

	// go to: part beginning
	if _, err := seek(fh, partNo); err != nil {
		return err
	}

	// build reader
	r := io.LimitReader(fh, db.PartSize) // file part (plain) reader
	if useCompr {                        // compression reader
		// read all bytes
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		// compression
		b, _, err = enc.Compress(b)
		if err != nil {
			return err
		}
		// check size
		if int64(len(b)) != storageSize {
			return errors.New("upload size check fail")
		}
		// set compressed reader
		r = bytes.NewReader(b)
	}
	r = enc.CryptoReader(ioutil.NopCloser(r), 0, cryptKey) // encryption reader: encryption offset is 0 for each part

	// upload
	_, err := service.Save(storageName, r, 0)
	if err != nil {
		return err
	}

	// success
	return nil
}

// uploadFile uploads a whole file.
// Uses the uploadPart() function.
// Skip folder and zero files.
func uploadFile(absPath string, vFile db.VirtFile, service interf.Service, uploadedParts map[string]db.VFilePart, debug bool) error {
	// skip folder and zero files
	if vFile.IsDir || vFile.FileSize <= 0 {
		return nil // do nothing
	}

	// open file
	fh, err := os.Open(absPath)
	if err != nil {
		log.Printf("ERROR: %s/uploadFile: %v", packageName, err)
		return err
	}
	defer fh.Close()

	// upload all parts
	for partNo, part := range vFile.Parts {
		if !exists(part, service, uploadedParts) {
			// part not found -> upload
			if debug {
				if vFile.UseCompression {
					sizeInKb := float64(part.StorageSize) / 1024
					log.Printf("DEBUG: %s/uploadFile: part %d from '%s' (%.2f kB, compressed)", packageName, partNo, vFile.RelPath, sizeInKb)
				} else {
					sizeInMb := float64(part.StorageSize) / (1024 * 1024)
					log.Printf("DEBUG: %s/uploadFile: part %d from '%s' (%.2f MB)", packageName, partNo, vFile.RelPath, sizeInMb)
				}
			}
			// upload
			if err := uploadPart(fh, partNo, vFile.UseCompression, part.CryptDataKey, part.StorageName, part.StorageSize, service); err != nil {
				log.Printf("ERROR: %s/uploadFile: part %d from '%s': %v", packageName, partNo, vFile.RelPath, err)
				return err
			}
			// add to uploadedParts
			uploadedParts[part.StorageName+"|"+part.StorageMd5] = part
		}
	}

	// success
	return nil
}

// uploadDb remove all files with the name IndexName and upload the new db file.
func uploadDb(newDb db.Db, indexKey []byte, service interf.Service, debug bool) error {
	if debug {
		log.Printf("DEBUG: %s/uploadDb: new db with %d elements and %d bundles", packageName, len(newDb.VFiles), len(newDb.Bundles))
	}

	// first: remove all old DBs
	for _, f := range service.Files().All() {
		if f.Name() == IndexName {
			if err := service.Trash(f); err != nil {
				log.Printf("ERROR: %s/uploadDb: remove old db '%s': %v", packageName, f.Id(), err)
				return err
			}
		}
	}

	// secondly: upload new db
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := db.ToWriter(newDb, indexKey, buf); err != nil {
		log.Printf("ERROR: %s/uploadDb: save db #1: %v", packageName, err)
		return err
	}
	if _, err := service.Save(IndexName, buf, 0); err != nil {
		log.Printf("ERROR: %s/uploadDb: save db #2: %v", packageName, err)
		return err
	}

	// success
	return nil
}

// uploadBundle uploads the bundles from the database. The function is RAM intensive.
func uploadBundle(rootPath string, vDB db.Db, service interf.Service, uploadedParts map[string]db.VFilePart, debug bool) error {

	// upload bundles (only, if set)
	for _, bundle := range vDB.Bundles {

		// part exist -> skip
		if exists(bundle.VFilePart, service, uploadedParts) {
			continue
		}

		// UPLOAD: build bundle in ram
		var data = make([]byte, 0)
		for _, vFileId := range bundle.Content {
			// small singe file
			vFile := vDB.VFiles[vFileId]
			absPath := path.Join(rootPath, vFile.RelPath)
			// check path
			if len(vFile.Parts) != 1 {
				e := errors.New("wrong part count for a bundle element")
				log.Printf("ERROR: %s/uploadBundle: %v: len=%d, f=%s", packageName, e, len(vFile.Parts), vFile.RelPath)
				return e
			}
			part := vFile.Parts[0]
			// read all bytes
			b, err := ioutil.ReadFile(absPath)
			if err != nil {
				log.Printf("ERROR: %s/uploadBundle: %v", packageName, err)
				return err
			}
			// use compression (optional)
			if vFile.UseCompression {
				b, _, err = enc.Compress(b)
				if err != nil {
					log.Printf("ERROR: %s/uploadBundle: %v", packageName, err)
					return err
				}
			}
			// check size
			if int64(len(b)) != part.StorageSize {
				e := errors.New("part does not have the specified StorageSize")
				log.Printf("ERROR: %s/uploadBundle: %v: is:%d != db:%d; '%s'", packageName, e, len(b), part.StorageSize, vFile.RelPath)
				return e
			}
			// add crypt data
			enc.CryptBytes(b, int64(len(data)), bundle.CryptDataKey)
			data = append(data, b...)
		}

		// check bundle
		if int64(len(data)) != bundle.StorageSize {
			e := errors.New("bundle does not have the specified StorageSize")
			log.Printf("ERROR: %s/uploadBundle: %v: is=%d, ss=%d", packageName, e, int64(len(data)), bundle.StorageSize)
			return e
		}

		// debug
		if debug {
			sizeInMb := float64(bundle.StorageSize) / (1024 * 1024)
			log.Printf("DEBUG: %s/uploadBundle: %s ... %d files, %d (%.0f MB)", packageName, bundle.StorageName, len(bundle.Content), bundle.StorageSize, sizeInMb)
		}

		// upload
		_, err := service.Save(bundle.StorageName, bytes.NewReader(data), 0)
		if err != nil {
			log.Printf("ERROR: %s/uploadBundle: %v", packageName, err)
			return err
		}
		// add to uploadedParts
		uploadedParts[bundle.StorageName+"#"+fmt.Sprintf("%d", bundle.StorageSize)] = bundle.VFilePart
	}

	// success
	return nil
}
