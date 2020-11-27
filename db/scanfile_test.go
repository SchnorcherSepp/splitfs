package db_test

import (
	"encoding/hex"
	"fmt"
	"github.com/SchnorcherSepp/splitfs/db"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestScanFile_basics(t *testing.T) {
	// DEMO files
	s := impl.NewRamService(nil, impl.DebugOff)
	if err := impl.InitDemo(s); err != nil {
		t.Error(err)
	}
	keyFile, err := enc.LoadKeyFile(path.Join(os.TempDir(), "testCryptKeyFile.dat"))
	if err != nil {
		t.Error(err)
	}

	// TESTS with all files
	for _, f := range s.Files().All() {
		if strings.HasPrefix(f.Name(), "small-test-file-") {
			continue // ignore small test files
		}

		// WRITE TEST FILE
		name := f.Name()
		absPath := path.Join(os.TempDir(), name)
		writeTestFileToDisk(absPath, s, f, t)

		// SCAN
		vf, err := db.ScanFile(absPath, name, keyFile)
		if err != nil {
			t.Error(err)
		}

		// TEST: getFileStat
		if f.Size() != vf.FileSize { // size check
			t.Errorf("fail: %v != %v", f, vf)
		}
		if len(vf.Parts) != 1 && vf.FileSize != 0 { // not empty
			t.Errorf("fail: %v != %v", f, vf)
		}
		if len(vf.Parts) != 0 && vf.FileSize == 0 { // empty file
			t.Errorf("fail: %v != %v", f, vf)
		}
		if f.Name() != vf.Name() { // name check
			t.Errorf("fail: %v != %v", f, vf)
		}
		if vf.MTime < 1000 { // modTime not 0
			t.Errorf("fail: %v != %v", f, vf)
		}

		// TEST: tryCompression
		if vf.FileSize > 0 { // no content -> no parts
			useCompr := vf.UseCompression
			partSize := vf.FileSize
			comprSize := vf.Parts[0].StorageSize

			// compression:  comprSize is smaller than partSize
			if partSize <= comprSize {
				// no compression
				if useCompr {
					t.Errorf("E: [%v], part=%d, compr=%d", useCompr, partSize, comprSize)
				}
			} else {
				// partSize > comprSize : USE COMPRESSION
				if !useCompr {
					t.Errorf("E: [%v], part=%d, compr=%d", useCompr, partSize, comprSize)
				}
			}

			// no compression for big files
			if vf.FileSize > db.MaxFileSizeForCompression {
				if useCompr {
					t.Errorf("E: [%v], part=%d, compr=%d", useCompr, partSize, comprSize)
				}
			} else {
				// no compression for "0.000000.dat" files
				if strings.HasSuffix(vf.Name(), "0.000000.dat") {
					if useCompr {
						t.Errorf("E: [%v], part=%d, compr=%d", useCompr, partSize, comprSize)
					}
				} else {
					// small files with rate > 0
					if !useCompr {
						t.Errorf("E: [%v], part=%d, compr=%d: %s", useCompr, partSize, comprSize, absPath)
					}
				}
			}
		}
	}
}

func TestScanFile_hash(t *testing.T) {
	keyName := "testCryptKeyFile.dat"
	absPath := path.Join(os.TempDir(), keyName)
	keyFile, err := enc.LoadKeyFile(absPath)
	if err != nil {
		t.Error(err)
	}

	vf, err := db.ScanFile(absPath, keyName, keyFile)
	if err != nil {
		t.Error(err)
	}

	if vf.FileSize != 128 {
		t.Fatal("error")
	}
	if vf.MTime < 1000 {
		t.Fatal("error")
	}
	if vf.Name() != keyName {
		t.Fatal("error")
	}
	if vf.RelPath != keyName {
		t.Fatal("error")
	}
	ht, _ := hex.DecodeString("DD5610DABC3B5C9BF4F567AAD68AABA0489DD5B9C6552C8C8B6AC4EC6DFA71430C827DD2675BA6760BB635C59964218A3F17F6B995932F5C47CFEF666761CE69")
	if len(vf.Parts) != 1 || !reflect.DeepEqual(vf.Parts[0].PlainSHA512, ht) || vf.FileSize != 128 {
		t.Fatal("error")
	}
	if len(vf.Parts) != 1 || vf.Parts[0].StorageMd5 != "48dc244300e593ed87a08bc48538eeb4" || vf.FileSize != 128 {
		t.Fatal("error")
	}
	if len(vf.Parts) != 1 || vf.Parts[0].StorageName != "88a42a5c8c911a126d09afa29c377821a7d426a78bfd8baa99a3be3890c2e37d5daf455fdf7e60695edcc9c64716add0ff39ea794f0b47d07c0f528079240d46" || vf.FileSize != 128 {
		t.Fatal("error")
	}
}

func TestScanFile_parts(t *testing.T) {
	keyFile, err := enc.LoadKeyFile(path.Join(os.TempDir(), "testCryptKeyFile.dat"))
	if err != nil {
		t.Error(err)
	}

	// write big test file
	buf := make([]byte, db.PartSize)
	for i := 0; i < len(buf); i++ {
		buf[i] = byte(0xa0 + i%10)
	}

	name := "bigPartTestFile.dat"
	absPath := path.Join(os.TempDir(), name)
	err = ioutil.WriteFile(absPath, buf, 0666)
	if err != nil {
		t.Error(err)
	}

	// SCAN
	vf, err := db.ScanFile(absPath, name, keyFile)
	if err != nil {
		t.Error(err)
	}

	// TEST
	if len(vf.Parts) != 1 {
		t.Errorf("len %d", len(vf.Parts))
		t.Fatalf("%#v", vf)
	}
	if vf.FileSize != db.PartSize || vf.Parts[0].StorageSize != db.PartSize {
		t.Fatal("error") // wrong size
	}
	su := "8951d233a7ca48b074f186f40a0804c57525ccd6d95987bef2cd189993c22a40bfed4df63606432b2a8cd7c9db0c020cdd2827439afa5cb08fa7979b89c5cd8a"
	is := fmt.Sprintf("%x", vf.Parts[0].PlainSHA512)
	if is != su {
		t.Fatalf("\nis=%s\nsu=%s\n", is, su) // wrong plain hash
	}

	// ------ add next part -----

	// add one byte
	fh, err := os.OpenFile(absPath, os.O_APPEND, 0666)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	_, err = fh.Write([]byte{'x'})
	if err != nil {
		t.Fatal(err)
	}
	_ = fh.Close()

	// SCAN 2
	vf2, err := db.ScanFile(absPath, name, keyFile)
	if err != nil {
		t.Error(err)
	}

	// TEST 2
	if len(vf2.Parts) != 2 {
		t.Errorf("len %d", len(vf2.Parts))
		t.Fatalf("%#v", vf2)
	}
	// part 0
	is = fmt.Sprintf("%x", vf2.Parts[0].PlainSHA512)
	if is != su {
		t.Fatalf("\nis=%s\nsu=%s\n", is, su) // wrong plain hash
	}
	// part 1
	su = "a4abd4448c49562d828115d13a1fccea927f52b4d5459297f8b43e42da89238bc13626e43dcb38ddb082488927ec904fb42057443983e88585179d50551afe62"
	is = fmt.Sprintf("%x", vf2.Parts[1].PlainSHA512)
	if is != su {
		t.Fatalf("\nis=%s\nsu=%s\n", is, su) // wrong plain hash
	}

	// check StorageName
	storageName1 := "e8500c2ec27f5de517bdb874600bf0350c3ca3c8c8cb7728f5648330abb15e6932b636d11cf7549bb18985e6864b57f4d993d39175605fbef7e878958b3333f6"
	storageName2 := "0035af136ea18f9c45a6582ddf9a7d73003c066efb3b5817b0d5f3cb2ad658303feed8097d98c7dc89c550d2627e880a146d9a6f060375ac64ade385becc4fad"
	if vf2.Parts[0].StorageName != storageName1 || vf2.Parts[1].StorageName != storageName2 {
		t.Fatalf("error")
	}

	// check md5 hash
	storageMd5a := "e5baf007392ede826f2f039a7cd6b532"
	storageMd5b := "d1a3d501c8b9ec199a2e4c93f8ba438d"
	if vf2.Parts[0].StorageMd5 != storageMd5a {
		t.Fatalf("error part 0: %s", vf2.Parts[0].StorageMd5)
	}
	if vf2.Parts[1].StorageMd5 != storageMd5b {
		t.Fatalf("error part 1: %s", vf2.Parts[1].StorageMd5)
	}

	// CryptDataKey
	if !reflect.DeepEqual(vf2.Parts[0].CryptDataKey, keyFile.DataKey(vf2.Parts[0].PlainSHA512)) {
		t.Fatalf("error")
	}
	if !reflect.DeepEqual(vf2.Parts[1].CryptDataKey, keyFile.DataKey(vf2.Parts[1].PlainSHA512)) {
		t.Fatalf("error")
	}
}

// ----------  HELPER  -----------------------------------------------------------------------------------------------//

func writeTestFileToDisk(absPath string, service interf.Service, file interf.File, t *testing.T) {
	// get reader
	reader, err := service.Reader(file, 0)
	if err != nil && err != io.EOF {
		t.Error(err)
	}

	// open fh
	fh, err := os.Create(absPath)
	if err != nil {
		t.Error(err)
	}
	defer fh.Close()

	if reader == nil {
		// zero file
		return
	}

	// write data
	n, err := io.Copy(fh, reader)
	if err != nil {
		t.Error(err)
	}

	// check size
	if n != file.Size() {
		t.Error("size error")
	}
}
