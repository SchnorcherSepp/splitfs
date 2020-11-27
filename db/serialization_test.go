package db

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

var testSerializationDb Db

func init() {
	testSerializationDb = NewDb()
	testSerializationDb.VFiles["./splitStorage.file"] = VirtFile{
		RelPath:        "./splitStorage.file",
		FileSize:       33,
		MTime:          65787654,
		IsDir:          false,
		UseCompression: false,
		Parts: []VFilePart{{
			PlainSHA512:  []byte{00, 00, 12},
			StorageName:  "aabbccddeeff",
			StorageSize:  33,
			StorageMd5:   "a0b0c0",
			CryptDataKey: nil,
		}},
		FolderContent: nil,
	}
}

func Test_db2gob_gob2db(t *testing.T) {

	// nil
	b, err := db2gob(Db{})
	if err == nil {
		t.Fatal("no error")
	}
	if len(b) != 0 {
		t.Fatal("wrong bytes")
	}

	// empty
	b, err = db2gob(NewDb())
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 100 || !strings.Contains(string(b), "db.VirtFile") {
		t.Fatal("wrong bytes")
	}

	// test db
	b, err = db2gob(testSerializationDb)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 100 || !strings.Contains(string(b), "splitStorage.file") {
		t.Fatal("wrong bytes")
	}

	//-----------------------------------------

	// test db
	tdb, err := gob2db(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(testSerializationDb, tdb) {
		t.Fatalf("wrong db: %#v", tdb)
	}

	// deep var test
	if tdb.VFiles["./splitStorage.file"].Parts[0].StorageMd5 != "a0b0c0" {
		t.Fatal("fail")
	}

	// nil
	_, err = gob2db(nil)
	if err == nil {
		t.Fatal("no error")
	}

	// wrong bytes
	_, err = gob2db([]byte{01, 02, 0xab, 12})
	if err == nil {
		t.Fatal("no error")
	}
}

func Test_gob2zip_zip2gob(t *testing.T) {

	// nil
	b, err := gob2zip(nil)
	if err == nil {
		t.Fatal("no error")
	}
	if len(b) != 0 {
		t.Fatalf("wrong output")
	}

	// empty
	b, err = gob2zip([]byte{})
	if err == nil {
		t.Fatal("no error")
	}
	if len(b) != 0 {
		t.Fatalf("wrong output")
	}

	// test bytes
	testBytes := []byte("Test String! Test String! Test String! Test String! Test String! Test String! ")
	b, err = gob2zip(testBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) <= 0 || len(b) > len(testBytes)/2 {
		t.Fatalf("wrong output: %d: %x", len(b), b)
	}

	//----------------------------------------------------------------------------

	b, err = zip2gob(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != len(testBytes) || !reflect.DeepEqual(b, testBytes) {
		t.Fatalf("wrong output: %d: %x", len(b), b)
	}

	// nil
	b, err = zip2gob(nil)
	if err == nil {
		t.Fatal("no error")
	}

	// empty
	b, err = zip2gob([]byte{})
	if err == nil {
		t.Fatal("no error")
	}

	// empty + checksum = 4 byte (err: size check fail)
	b, err = zip2gob([]byte{0, 0, 0, 0})
	if err == nil {
		t.Fatal("no error")
	}

	// checksum error
	b, err = zip2gob([]byte{0, 0, 0, 0, 0})
	if err == nil || fmt.Sprintf("%v", err) != "checksum fail" {
		t.Fatal("no error")
	}

	// compression error  (checksum is ok)
	b, err = zip2gob([]byte{0x1b, 0xdf, 0x05, 0xa5, 1})
	if err == nil || fmt.Sprintf("%v", err) != "magic number invalid" {
		t.Fatal("no error")
	}
}

func Test_zip2enc_enc2zip(t *testing.T) {
	const gcmStandardNonceSize = 12
	plain := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x01, 0x10}
	key := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10} // 128 bit key

	// wrong key
	cryp, err := zip2enc(plain, key[0:15])
	if err == nil {
		t.Fatal("no error")
	}

	// success
	cryp, err = zip2enc(plain, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(cryp) != gcmStandardNonceSize+16+8 {
		t.Fatalf("block cypher output error: %d", len(cryp))
	}

	// check nonce
	nonce1 := cryp[:gcmStandardNonceSize]
	cryp2, _ := zip2enc(plain, key)
	nonce2 := cryp2[:gcmStandardNonceSize]
	if reflect.DeepEqual(nonce1, nonce2) {
		t.Fatalf("no random nonce")
	}

	//-----------------------------------------------------------

	// wrong key
	_, err = enc2zip(cryp, key[0:15])
	if err == nil {
		t.Fatal("no error")
	}

	// invalid nonce
	_, err = enc2zip(cryp[:gcmStandardNonceSize], key)
	if err == nil || fmt.Sprintf("%v", err) != "size check fail" {
		t.Fatalf("nonce fail")
	}

	// invalid sig
	crypFail := append(append(append([]byte{}, cryp[:20]...), 0xff), cryp[21:]...)
	_, err = enc2zip(crypFail, key)
	if err == nil || fmt.Sprintf("%v", err) != "cipher: message authentication failed" {
		t.Fatalf("no or wrong error: %v", err)
	}
	crypFail = append([]byte{0x00}, cryp[1:]...)
	_, err = enc2zip(crypFail, key)
	if err == nil || fmt.Sprintf("%v", err) != "cipher: message authentication failed" {
		t.Fatalf("no or wrong error: %v", err)
	}

	// success
	enc, err := enc2zip(cryp, key)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(enc, plain) || len(enc) != len(plain) {
		t.Fatalf("output check")
	}
}

func Test_ALL_ToFile_FromFile(t *testing.T) {
	key := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10} // 128 bit key
	p := path.Join(os.TempDir(), "Test_ALL_ToFile_FromFile.dat")
	_ = os.Remove(p)

	for n := 0; n < 4; n++ {
		// write
		err := ToFile(testSerializationDb, key, p)
		if err != nil {
			t.Fatal(err)
		}
		// read
		tmp, err := FromFile(p, key)
		if err != nil {
			t.Fatal(err)
		}
		// check
		_, ok := tmp.VFiles["./splitStorage.file"]
		if !ok {
			t.Fatalf("wrong db")
		}
		if tmp.VFiles["./splitStorage.file"].Parts[0].StorageSize != 33 {
			t.Fatalf("wrong db")
		}
		if !reflect.DeepEqual(tmp, testSerializationDb) {
			t.Fatalf("wrong db")
		}
	}
}
