package core_test

import (
	"bytes"
	"fmt"
	"github.com/SchnorcherSepp/splitfs/core"
	"github.com/SchnorcherSepp/splitfs/db"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"io"
	"log"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"testing"
)

var (
	testOpenDb      db.Db
	testOpenService interf.Service
)

func initTest(t *testing.T) (db.Db, interf.Service) {
	if testOpenDb.VFiles == nil {

		// reuse test files from upload_test
		//---------------------------------------
		// scan folder
		var err error
		testOpenDb, _, _, err = db.FromScan(testUploadFolderPath, db.NewDb(), impl.DebugOff, testUploadKeyFile)
		if err != nil {
			t.Fatal(err)
		}
		// make bundle
		testOpenDb.MakeBundles(testUploadKeyFile, impl.DebugOff)

		// prepare ram service
		//---------------------------------------
		// service
		testOpenService = impl.NewRamService(nil, impl.DebugHigh)
		// upload
		if err := core.Upload(testUploadFolderPath, testOpenDb, testUploadKeyFile.IndexKey(), testOpenService, impl.DebugOff); err != nil {
			t.Fatal(err)
		}
		// update fileList
		if err := testOpenService.Update(); err != nil {
			t.Fatal(err)
		}
	}
	return testOpenDb, testOpenService
}

//--------------------------------------------------------------------------------------------------------------------//

// CASE 0: zero file (0 bytes)
func TestOpen_zero(t *testing.T) {
	vDb, service := initTest(t)
	file, ok := vDb.VFiles["special-file-0-0.990000.dat"]
	if !ok {
		t.Fatal("test file not found")
	}
	if file.FileSize != 0 {
		t.Fatal("wrong file for this test")
	}

	// open
	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	log.SetOutput(stdoutBuf)
	//-------------------------------------------------
	r, err := core.Open(file, vDb, service, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	//-------------------------------------------------
	log.SetOutput(os.Stdout)
	s := stdoutBuf.String()
	fmt.Printf("%s\n", s)

	// check
	if strings.Contains(s, "from bundle") || !strings.Contains(s, "ZeroReaderAt for ") {
		t.Fatal("wrong log")
	}

	// test
	buf := make([]byte, 3)
	if n, err := r.ReadAt(buf, 0); n != 0 || err != io.EOF {
		t.Fatalf("n=%d, err=%v", n, err)
	}
}

// CASE 1: compressed file from a single source   a) bundle
func TestOpen_comp_single_bundle(t *testing.T) {
	vDb, service := initTest(t)
	file, ok := vDb.VFiles["special-file-1048575-0.990000.dat"]
	if !ok {
		t.Fatal("test file not found")
	}
	if file.UseCompression != true || len(file.Parts) != 1 || file.AlsoInBundle == "" {
		t.Fatal("wrong file for this test")
	}

	// open
	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	log.SetOutput(stdoutBuf)
	//-------------------------------------------------
	r, err := core.Open(file, vDb, service, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	//-------------------------------------------------
	log.SetOutput(os.Stdout)
	s := stdoutBuf.String()
	fmt.Printf("%s\n", s)

	// check
	if !strings.Contains(s, "from bundle") || !strings.Contains(s, "RamReaderAt for ") {
		t.Fatal("wrong log")
	}

	// test
	buf := make([]byte, 3)
	if n, err := r.ReadAt(buf, 0); n != 3 || err != nil {
		t.Fatalf("n=%d, err=%v", n, err)
	}
}

// CASE 1: compressed file from a single source   b) part
func TestOpen_comp_single_part(t *testing.T) {
	vDb, service := initTest(t)
	file, ok := vDb.VFiles["special-file-1048575-0.660000.dat"]
	file.AlsoInBundle = "" // FIX: there are no file with compression AND no bundle
	if !ok {
		t.Fatal("test file not found")
	}
	if file.UseCompression != true || len(file.Parts) != 1 || file.AlsoInBundle != "" {
		t.Fatal("wrong file for this test")
	}

	// open
	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	log.SetOutput(stdoutBuf)
	//-------------------------------------------------
	r, err := core.Open(file, vDb, service, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	//-------------------------------------------------
	log.SetOutput(os.Stdout)
	s := stdoutBuf.String()
	fmt.Printf("%s\n", s)

	// check
	if strings.Contains(s, "from bundle") || !strings.Contains(s, "RamReaderAt for ") {
		t.Fatal("wrong log")
	}

	// test
	buf := make([]byte, 3)
	if n, err := r.ReadAt(buf, 0); n != 3 || err != nil {
		t.Fatalf("n=%d, err=%v", n, err)
	}
}

// CASE 2: regular file from a single source   a) bundle
func TestOpen_reg_single_bundle(t *testing.T) {
	vDb, service := initTest(t)
	file, ok := vDb.VFiles["special-file-1048575-0.000000.dat"]
	if !ok {
		t.Fatal("test file not found")
	}
	if file.UseCompression != false || len(file.Parts) != 1 || file.AlsoInBundle == "" {
		t.Fatal("wrong file for this test")
	}

	// open
	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	log.SetOutput(stdoutBuf)
	//-------------------------------------------------
	r, err := core.Open(file, vDb, service, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	//-------------------------------------------------
	log.SetOutput(os.Stdout)
	s := stdoutBuf.String()
	fmt.Printf("%s\n", s)

	// check
	if !strings.Contains(s, "from bundle") || !strings.Contains(s, "SubReaderAt for ") {
		t.Fatal("wrong log")
	}

	// test
	buf := make([]byte, 3)
	if n, err := r.ReadAt(buf, 0); n != 3 || err != nil {
		t.Fatalf("n=%d, err=%v", n, err)
	}
}

// CASE 2: regular file from a single source   b) part
func TestOpen_reg_single_part(t *testing.T) {
	vDb, service := initTest(t)
	file, ok := vDb.VFiles["special-file-16777215-0.990000.dat"]
	if !ok {
		t.Fatal("test file not found")
	}
	if file.UseCompression != false || len(file.Parts) != 1 || file.AlsoInBundle != "" {
		t.Fatal("wrong file for this test")
	}

	// open
	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	log.SetOutput(stdoutBuf)
	//-------------------------------------------------
	r, err := core.Open(file, vDb, service, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	//-------------------------------------------------
	log.SetOutput(os.Stdout)
	s := stdoutBuf.String()
	fmt.Printf("%s\n", s)

	// check
	if strings.Contains(s, "from bundle") || !strings.Contains(s, "SubReaderAt for ") {
		t.Fatal("wrong log")
	}

	// test
	buf := make([]byte, 3)
	if n, err := r.ReadAt(buf, 0); n != 3 || err != nil {
		t.Fatalf("n=%d, err=%v", n, err)
	}
}

// CASE 3 (default): multi part file
func TestOpen_multi_part(t *testing.T) {
	vDb, service := initTest(t)
	file, ok := vDb.VFiles["next-part-test-file.dat"]
	if !ok {
		t.Fatal("test file not found")
	}
	if len(file.Parts) <= 1 {
		t.Fatal("wrong file for this test")
	}

	// open
	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	log.SetOutput(stdoutBuf)
	//-------------------------------------------------
	r, err := core.Open(file, vDb, service, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	//-------------------------------------------------
	log.SetOutput(os.Stdout)
	s := stdoutBuf.String()
	fmt.Printf("%s\n", s)

	// check
	if strings.Contains(s, "from bundle") || !strings.Contains(s, "MultiReaderAt for ") {
		t.Fatal("wrong log")
	}

	// test
	buf := make([]byte, 3)
	if n, err := r.ReadAt(buf, 0); n != 3 || err != nil {
		t.Fatalf("n=%d, err=%v", n, err)
	}
}

func TestOpen_ReadTest(t *testing.T) {
	vDb, service := initTest(t)

	// read all files
	for _, v := range vDb.VFiles {
		if v.IsDir {
			continue
		}
		log.Printf("> %s", v.RelPath)

		// disk
		absPath := path.Join(testUploadFolderPath, v.RelPath)
		bufOrig := make([]byte, 33333)
		rOrig, err := os.Open(absPath)
		if err != nil {
			t.Fatal(err)
		}

		// ram
		bufRam := make([]byte, len(bufOrig))
		rRam, err := core.Open(v, vDb, service, impl.DebugOff)
		if err != nil {
			t.Fatal(err)
		}

		// check
		for off := int64(0); off < v.FileSize+33+int64(len(bufOrig)); off += int64(len(bufOrig)) {
			nR, errR := rRam.ReadAt(bufRam, off)
			nO, errO := rOrig.ReadAt(bufOrig, off)
			if nR != nO || errR != errO {
				t.Errorf("off=%d, nR=%d, nO=%d, errR=%v, errO=%v", off, nR, nO, errR, errO)
			}
			if !reflect.DeepEqual(bufRam[:nR], bufOrig[:nO]) {
				t.Fatal("diff")
			}
		}

		// close
		_ = rOrig.Close()
		_ = rRam.Close()
	}
}

//--------------------------------------------------------------------------------------------------------------------//

func TestRace_Open(t *testing.T) {
	vDb, service := initTest(t)
	file := vDb.VFiles["next-part-test-file.dat"]

	var wg sync.WaitGroup
	wg.Add(5)
	for n := 0; n < 5; n++ {
		go func() {
			//------------------------------
			buf := make([]byte, 1)
			for i := 0; i < 100; i++ {
				r, err := core.Open(file, vDb, service, impl.DebugOff)
				if err != nil {
					t.Fatal(err)
				}
				n, err := r.ReadAt(buf, int64(i))
				if err != nil || n != 1 {
					t.Fatalf("n=%d, err=%v", n, err)
				}
			}
			//------------------------------
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestRace_Open_Read(t *testing.T) {
	vDb, service := initTest(t)
	file := vDb.VFiles["next-part-test-file.dat"]

	r, err := core.Open(file, vDb, service, impl.DebugOff)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(5)
	for n := 0; n < 5; n++ {
		go func() {
			//------------------------------
			buf := make([]byte, 1)
			for i := 0; i < 100; i++ {
				n, err := r.ReadAt(buf, int64(i))
				if err != nil || n != 1 {
					t.Fatalf("n=%d, err=%v", n, err)
				}
			}
			//------------------------------
			wg.Done()
		}()
	}
	wg.Wait()
}
