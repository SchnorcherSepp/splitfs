package db

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"github.com/SchnorcherSepp/splitfs/encoding"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
)

// db2gob serializes the database object and return bytes.
func db2gob(db Db) ([]byte, error) {
	// input validation
	if db.VFiles == nil {
		return []byte{}, errors.New("db is nul")
	}

	// encode
	var buf = new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(db); err != nil {
		return []byte{}, err
	}

	// return
	return buf.Bytes(), nil
}

// gob2zip uses Compress() for data compression. This halves the size of the database.
func gob2zip(p []byte) ([]byte, error) {
	// input validation
	if p == nil || len(p) == 0 {
		return []byte{}, errors.New("input is nul or empty")
	}

	// compress
	b, _, err := enc.Compress(p)
	if err != nil {
		return []byte{}, err
	}

	// checksum
	c := make([]byte, 4)
	binary.LittleEndian.PutUint32(c, crc32.ChecksumIEEE(b))

	// return bytes
	return append(c, b...), nil
}

// zip2enc encrypts bytes (AES Galois Counter Mode)
func zip2enc(zip []byte, key []byte) ([]byte, error) {
	// create AES cipher with 16, 24, or 32 bytes key
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("ERROR: %s/zip2enc: NewCipher: %v", packageName, err)
		return []byte{}, err
	}

	// Galois Counter Mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("ERROR: %s/zip2enc: NewGCM: %v", packageName, err)
		return []byte{}, err
	}

	// create random nonce with standard length
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		log.Printf("ERROR: %s/zip2enc: nonce: %v", packageName, err)
		return []byte{}, err
	}

	// encrypts and authenticates plaintext
	encByt := gcm.Seal(nil, nonce, zip, nil)

	return append(nonce, encByt...), nil
}

// ToWriter serializes, compresses, encrypts and writes a database to a writer.
func ToWriter(db Db, key []byte, w io.Writer) error {

	p, err := db2gob(db)
	if err != nil {
		return err // logging in sub function
	}

	zip, err := gob2zip(p)
	if err != nil {
		return err // logging in sub function
	}

	encByt, err := zip2enc(zip, key)
	if err != nil {
		return err // logging in sub function
	}

	_, err = w.Write(encByt)
	if err != nil {
		log.Printf("ERROR: %s/ToWriter: %v", packageName, err)
		return err
	}

	return nil
}

// ToFile serializes, compresses, encrypts and writes a database to a file.
func ToFile(db Db, key []byte, path string) error {
	// overwrite db file
	fh, err := os.Create(path)
	if err != nil {
		log.Printf("ERROR: %s/ToFile: %v", packageName, err)
		return err
	}
	defer fh.Close()

	return ToWriter(db, key, fh)
}

// ------------------------------------------------------------------------------------------------------------------ //

// FromFile load a database from a file.
func FromFile(path string, key []byte) (Db, error) {
	// no file -> return empty DB
	_, err := os.Stat(path)
	if err != nil {
		log.Printf("WARNING: %s/FromFile: return empty db: %v", packageName, err)
		return NewDb(), nil // NO ERROR! (warning)
	}

	// open file
	fh, err := os.Open(path)
	if err != nil {
		log.Printf("ERROR: %s/FromFile: %v", packageName, err)
		return NewDb(), err
	}
	defer fh.Close()

	// read
	return FromReader(fh, key)
}

// FromReader load a database from a reader.
func FromReader(r io.Reader, key []byte) (Db, error) {
	// read all
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Printf("ERROR: %s/FromReader: %v", packageName, err)
		return NewDb(), err
	}

	zip, err := enc2zip(b, key)
	if err != nil {
		return NewDb(), err // logging in sub function
	}

	p, err := zip2gob(zip)
	if err != nil {
		return NewDb(), err // logging in sub function
	}

	ret, err := gob2db(p)
	if err != nil {
		return NewDb(), err // logging in sub function
	}

	return ret, nil
}

// enc2zip decrypts bytes (AES Galois Counter Mode)
func enc2zip(enc []byte, key []byte) ([]byte, error) {
	// big enough for nonce?
	const gcmStandardNonceSize = 12
	if len(enc) <= gcmStandardNonceSize {
		log.Printf("ERROR: %s/enc2zip: size check fail", packageName)
		return []byte{}, errors.New("size check fail")
	}
	nonce := enc[:gcmStandardNonceSize]
	enc = enc[gcmStandardNonceSize:]

	// create AES cipher with 16, 24, or 32 bytes key
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("ERROR: %s/enc2zip: NewCipher: %v", packageName, err)
		return []byte{}, err
	}

	// Galois Counter Mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("ERROR: %s/enc2zip: NewGCM: %v", packageName, err)
		return []byte{}, err
	}

	// decrypts and authenticates cipher text
	b, err := gcm.Open(nil, nonce, enc, nil)
	if err != nil {
		log.Printf("ERROR: %s/enc2zip: enc: %v", packageName, err)
		return []byte{}, err
	}

	return b, nil
}

// zip2gob uses db.Decompress() for data decompression.
func zip2gob(zip []byte) ([]byte, error) {
	// big enough for checksum?
	const checkSumSize = 4
	if len(zip) <= checkSumSize {
		err := errors.New("size check fail")
		log.Printf("ERROR: %s/zip2gob: %v", packageName, err)
		return []byte{}, err
	}
	sum := zip[:checkSumSize]
	b := zip[checkSumSize:]

	// checksum
	c := make([]byte, 4)
	binary.LittleEndian.PutUint32(c, crc32.ChecksumIEEE(b))
	if !reflect.DeepEqual(c, sum) {
		err := errors.New("checksum fail")
		log.Printf("ERROR: %s/zip2gob: %v", packageName, err)
		return []byte{}, err
	}

	// decompress
	b, err := enc.Decompress(b)
	if err != nil {
		log.Printf("ERROR: %s/zip2gob: %v", packageName, err)
		return []byte{}, err
	}

	return b, nil
}

// gob2db de-serializes the database object.
func gob2db(p []byte) (Db, error) {
	var ret Db

	decoder := gob.NewDecoder(bytes.NewReader(p))
	err := decoder.Decode(&ret)

	if err != nil {
		log.Printf("ERROR: %s/gob2db: %v", packageName, err)
		return NewDb(), err
	}

	return ret, nil
}
