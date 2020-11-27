package db_test

import (
	"github.com/SchnorcherSepp/splitfs/db"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"os"
	"path"
	"testing"
)

func TestScanFolder(t *testing.T) {
	keyFile, err := enc.LoadKeyFile(path.Join(os.TempDir(), "testCryptKeyFile.dat"))
	if err != nil {
		t.Error(err)
	}

	// scan local dir
	newDb1, changed1, _, err1 := db.FromScan("../", db.NewDb(), impl.DebugOff, keyFile)
	if err1 != nil {
		t.Fatal(err)
	}

	// scan local dir (again)
	newDb2, changed2, _, err2 := db.FromScan("../", newDb1, impl.DebugHigh, keyFile)
	if err2 != nil {
		t.Fatal(err)
	}

	// add a fake file and scan local dir (again)
	newDb2.VFiles["fakeFolder"] = db.VirtFile{

		RelPath:  "fakeFolder",
		FileSize: 0,
		MTime:    1234,
		Parts:    nil,

		IsDir:         true,
		FolderContent: []db.FolderEl{},
	}
	newDb3, changed3, _, err3 := db.FromScan("../", newDb2, impl.DebugHigh, keyFile)
	if err3 != nil {
		t.Fatal(err)
	}

	// check change flag
	if changed1 != true || changed2 != false || changed3 != true {
		t.Fatalf("changed1=%v, changed2=%v, changed3=%v", changed1, changed2, changed3)
	}

	// check folder content AND chunks
	for k, v := range newDb3.VFiles {
		if !v.IsDir && (v.FolderContent != nil || v.Parts == nil) {
			t.Errorf("error %s", k)
		}
		if v.IsDir && (v.FolderContent == nil || v.Parts != nil) {
			t.Errorf("error %s", k)
		}
		if v.Name() == "" {
			t.Errorf("error %s", k)
		}
	}
}
