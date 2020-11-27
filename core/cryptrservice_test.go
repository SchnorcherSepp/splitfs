package core

import (
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"reflect"
	"testing"
)

func Test_newCryptRService(t *testing.T) {
	// init
	ramService := impl.NewRamService(nil, impl.DebugOff)
	_ = impl.InitDemo(ramService)
	file1, err := ramService.Files().ByName("small-test-file-1.dat")
	if err != nil {
		t.Fatal(err)
	}
	file2, err := ramService.Files().ByName("small-test-file-2.dat")
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string][]byte{
		file1.Id(): make([]byte, 16),
	}
	// ----------------------------------------------------------------------

	// TEST nil
	if _, err = newCryptRService(nil, nil).Reader(file1, 0); err == nil {
		t.Fatal("no error")
	}
	if _, err = newCryptRService(ramService, nil).Reader(file1, 0); err == nil {
		t.Fatal("no error")
	}
	if _, err = newCryptRService(nil, keys).Reader(file1, 0); err == nil {
		t.Fatal("no error")
	}
	if _, err = newCryptRService(ramService, make(map[string][]byte)).Reader(file1, 0); err == nil {
		t.Fatal("no error")
	}
	if _, err = newCryptRService(ramService, keys).Reader(nil, 0); err == nil {
		t.Fatal("no error")
	}

	// TEST ok
	if _, err = newCryptRService(ramService, keys).Reader(file1, 0); err != nil {
		t.Fatal(err)
	}

	// TEST key not found
	if _, err = newCryptRService(ramService, keys).Reader(file2, 0); err == nil {
		t.Fatal("no error")
	}

	// TEST read
	{
		// get crypt reader
		r, err := newCryptRService(ramService, keys).Reader(file1, 0)
		if err != nil {
			t.Fatal(err)
		}

		// read crypt data
		buf := make([]byte, 100)
		n, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		buf = buf[:n]
		_ = r.Close()

		// read plain data
		bufPlain := make([]byte, 100)
		r, err = ramService.Reader(file1, 0)
		if err != nil {
			t.Fatal(err)
		}
		n, err = r.Read(bufPlain)
		if err != nil {
			t.Fatal(err)
		}
		bufPlain = bufPlain[:n]

		// compare
		if len(buf) != len(bufPlain) || reflect.DeepEqual(buf, bufPlain) {
			t.Fatal("crypto fail")
		}
		enc.CryptBytes(buf, 0, keys[file1.Id()])
		if len(buf) != len(bufPlain) || !reflect.DeepEqual(buf, bufPlain) {
			t.Fatal("crypto fail")
		}
	}
}
