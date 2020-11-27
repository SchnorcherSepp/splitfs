package db

import (
	"bytes"
	"crypto/md5"
	"crypto/sha512"
	"errors"
	"fmt"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	"io"
	"io/ioutil"
	"log"
	"os"
)

// ScanFile read a single file and calculate all values for a virtual file struct.
// relPath is for VirtFile.RelPath used only!
func ScanFile(absPath, relPath string, keyFile *enc.KeyFile) (VirtFile, error) {
	var errorFile = VirtFile{}

	// get file basics
	fileSize, modTime, err := getFileStat(absPath)
	if err != nil {
		return errorFile, err // stat error (file not found)
	}

	// check compression
	useCompression, comprSize, err := tryCompression(absPath, fileSize)
	if err != nil {
		return errorFile, err // compression error
	}

	// open file handler
	fh, err := os.Open(absPath)
	if err != nil {
		return errorFile, err // open error
	}
	defer fh.Close() // CLOSE

	// PART LOOP
	partList := make([]VFilePart, 0)
	for partNo := 0; true; partNo++ {

		// calc file part plain hash
		// the plain hash is the starting point for other calculations
		plainSHA512, partSize, err := plainSHA512(fh, partNo)
		if err != nil {
			return errorFile, err // hash error
		}

		// EXIT LOOP (part len = 0)
		// don't add empty parts to the partList
		if partSize == 0 {
			break
		}

		// calc storage file stuff
		dataKey := keyFile.DataKey(plainSHA512)
		storageName := keyFile.CryptName(plainSHA512)

		storageSize := partSize // DEFAULT (withOUT compression): storageSize == partSize
		if useCompression {
			storageSize = comprSize // with compression: storageSize == comprSize
		}

		// md5 file hash of the encrypted content
		storageMd5, err := cryptMD5(fh, partNo, useCompression, storageSize, dataKey)
		if err != nil {
			return errorFile, err // hash or partSize error
		}

		// build & add file part struct to list
		part := VFilePart{
			PlainSHA512:  plainSHA512,
			StorageName:  storageName,
			StorageSize:  storageSize,
			StorageMd5:   storageMd5,
			CryptDataKey: dataKey,
		}
		partList = append(partList, part)
	}

	// return VirtFile
	return VirtFile{
		RelPath:        relPath,
		FileSize:       fileSize,
		MTime:          modTime,
		IsDir:          false, // no folder
		FolderContent:  nil,   // no folder
		Parts:          partList,
		UseCompression: useCompression,
		AlsoInBundle:   "", // no bundles at this point
	}, nil
}

// ----------  HELPER  -----------------------------------------------------------------------------------------------//

// getFileStat read the basic file attributes
// return error if file not exist
func getFileStat(absPath string) (fileSize, modTime int64, err error) {
	// read file info
	var st os.FileInfo
	st, err = os.Stat(absPath)
	if err != nil {
		return // file not exist
	}

	// file check
	if st.IsDir() {
		log.Printf("ERROR: %s/getFileStat: file is a folder: '%s'", packageName, absPath)
		err = errors.New("file is a folder")
		return // file is a folder !?
	}

	// return
	fileSize = st.Size()
	modTime = st.ModTime().Unix()
	return
}

// tryCompression checks whether the entire file can be compressed.
// The entire file is checked and NOT the parts.
// There is a size limit: MaxFileSizeForCompression
func tryCompression(absPath string, fileSize int64) (useCompression bool, comprSize int64, err error) {
	// init with default value
	useCompression = false
	comprSize = fileSize

	// file small enough for compression (MaxFileSizeForCompression)
	if fileSize <= MaxFileSizeForCompression && fileSize < PartSize {
		// read all data
		var buf []byte
		buf, err = ioutil.ReadFile(absPath)
		if err != nil {
			return // read error
		}
		// compression
		var ratio float32
		buf, ratio, err = enc.Compress(buf)
		if err != nil {
			return // compression error
		}
		// check results
		if ratio < 0.8 {
			// USE COMPRESSION!
			useCompression = true
			comprSize = int64(len(buf))
		}
	}
	return
}

// seek sets the offset for the next Read on file to the part start
func seek(fh *os.File, partNo int) (int64, error) {
	// calc offset
	offset := int64(partNo) * PartSize

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

// plainSHA512 calc the plain part hash
func plainSHA512(fh *os.File, partNo int) (plainSHA512 []byte, partSize int64, err error) {
	// go to: part beginning
	_, err = seek(fh, partNo)
	if err != nil {
		return // seek error
	}

	// hashing
	hh := sha512.New()
	partSize, err = io.Copy(hh, io.LimitReader(fh, PartSize)) // read part
	plainSHA512 = hh.Sum(nil)                                 // calc hash

	// return plainSHA512, partSize AND error
	return
}

// cryptMD5 calc the storage file hash (= crypt content)
// compression is only for the first part [0] possible
func cryptMD5(fh *os.File, partNo int, useCompr bool, storageSize int64, cryptKey []byte) (cryptMD5 string, err error) {
	// There is no second part with active compression!
	if useCompr && partNo > 0 {
		err = errors.New("can't compress second part")
		return // check error
	}

	// go to: part beginning
	_, err = seek(fh, partNo)
	if err != nil {
		return // seek error
	}

	// build reader
	r := io.LimitReader(fh, PartSize) // file part (plain) reader
	if useCompr {                     // compression reader
		// read all bytes
		var b []byte
		b, err = ioutil.ReadAll(r)
		if err != nil {
			return // read error
		}
		// compression
		b, _, err = enc.Compress(b)
		if err != nil {
			return // compression error
		}
		// set dummy reader
		r = bytes.NewReader(b)
	}
	r = enc.CryptoReader(ioutil.NopCloser(r), 0, cryptKey) // encryption reader: encryption offset is 0 for each part

	// hashing
	var n int64
	hh := md5.New()
	n, err = io.Copy(hh, r)
	cryptMD5 = fmt.Sprintf("%x", hh.Sum(nil))

	// check storageSize:
	//   without compression storageSize is partSize
	//   with compression storageSize is comprSize
	if storageSize != n {
		// extend other error
		err = fmt.Errorf("storageSize check fail: %v", err)
	}

	// return cryptMD5 AND error
	return
}
