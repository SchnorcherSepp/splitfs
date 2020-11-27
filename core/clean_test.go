package core_test

import (
	"bytes"
	"fmt"
	"github.com/SchnorcherSepp/splitfs/core"
	"github.com/SchnorcherSepp/splitfs/db"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"testing"
)

func TestClean_all(t *testing.T) {

	// service with test files
	service := impl.NewRamService(nil, impl.DebugOff)
	_, _ = service.Save("bbcceeffbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 60)), 0)
	_, _ = service.Save("B_aa77027cbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 80)), 0)
	_, _ = service.Save("rest1", bytes.NewReader(make([]byte, 12)), 0)
	_ = service.Update()

	// TEST: service == nil
	err := core.Clean(db.Db{}, nil, false, impl.DebugOff)
	if fmt.Sprintf("%v", err) != "service is nil" {
		t.Error("no error")
	}

	// TEST: try
	err = core.Clean(db.Db{}, service, true, impl.DebugOff)
	if err != nil {
		t.Error(err)
	}
	_ = service.Update()
	if len(service.Files().All()) != 3 {
		t.Error("wrong list size")
	}

	// TEST: remove all (empty db)
	//   * There are three types of files. A 'part' a 'bundle' and a 'rest'.
	//   * Only 'part' is deleted here.
	//   * 'bundle' is not deleted because the database does not contain any bundles (=> the function 'bundle' is deactivated)
	//   * 'rest' is never deleted.
	err = core.Clean(db.Db{}, service, false, impl.DebugOff)
	if err != nil {
		t.Error(err)
	}
	_ = service.Update()
	if len(service.Files().All()) != 2 {
		t.Error("wrong list size")
	}

	// TEST: bundle-mode-on (TRY)
	vDb := db.Db{Bundles: map[string]db.Bundle{"bundle1": {}}}
	err = core.Clean(vDb, service, true, impl.DebugOff)
	if err != nil {
		t.Error(err)
	}
	_ = service.Update()
	if len(service.Files().All()) != 2 {
		t.Error("wrong list size")
	}

	// TEST: bundle-mode-on (DO)
	err = core.Clean(vDb, service, false, impl.DebugOff)
	if err != nil {
		t.Error(err)
	}
	_ = service.Update()
	if len(service.Files().All()) != 1 {
		t.Error("wrong list size")
	}
}

func TestClean_attribute_duplicate(t *testing.T) {

	// service with test files
	service := impl.NewRamService(nil, impl.DebugOff)
	f1, _ := service.Save("bbcceeffbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 60)), 0)
	_, _ = service.Save("bbcceeffbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 60)), 0)
	_, _ = service.Save("bbcceeffbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 60)), 0)
	_, _ = service.Save("bbcceeffbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 60)), 0)
	f2, _ := service.Save("B_aa77027cbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 80)), 0)
	_, _ = service.Save("B_aa77027cbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46", bytes.NewReader(make([]byte, 80)), 0)
	_, _ = service.Save("rest1", bytes.NewReader(make([]byte, 12)), 0)
	_ = service.Update()
	if len(service.Files().All()) != 7 {
		t.Error("wrong list size")
	}

	// test db
	vDb := db.Db{
		VFiles: map[string]db.VirtFile{
			"bla": {Parts: []db.VFilePart{{StorageName: f1.Name(), StorageSize: f1.Size(), StorageMd5: f1.Md5()}}},
		},
		Bundles: map[string]db.Bundle{
			"bub": {VFilePart: db.VFilePart{StorageName: f2.Name(), StorageSize: f2.Size()}}, // bundle without md5 hash
		},
	}

	// clear duplicates (valid files in db)
	err := core.Clean(vDb, service, false, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	_ = service.Update()
	if len(service.Files().All()) != 3 {
		t.Error("wrong list size")
	}

	// clear file with wrong hash && clear bundle with wrong size
	vDb.VFiles["bla"].Parts[0].StorageMd5 = "error"
	tmp := vDb.Bundles["bub"]
	tmp.StorageSize = 33
	vDb.Bundles["bub"] = tmp
	err = core.Clean(vDb, service, false, impl.DebugHigh)
	if err != nil {
		t.Error(err)
	}
	_ = service.Update()
	if len(service.Files().All()) != 1 {
		t.Error("wrong list size")
	}
}
