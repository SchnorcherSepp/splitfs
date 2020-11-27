package core

import (
	"bytes"
	"github.com/SchnorcherSepp/splitfs/db"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"os"
	"testing"
)

func Test_posInBundle(t *testing.T) {

	target1 := db.VirtFile{
		RelPath:      "target.1",
		FileSize:     31,
		Parts:        []db.VFilePart{{StorageSize: 31}},
		AlsoInBundle: "bundleId",
	}
	target2 := db.VirtFile{
		RelPath:      "target.2",
		FileSize:     32,
		Parts:        []db.VFilePart{{StorageSize: 32}},
		AlsoInBundle: "bundleId",
	}
	target3 := db.VirtFile{
		RelPath:      "target.3",
		FileSize:     33,
		Parts:        []db.VFilePart{{StorageSize: 33}},
		AlsoInBundle: "bundleId",
	}
	bundle := db.Bundle{
		VFilePart: db.VFilePart{
			StorageName:  "bundleId",
			CryptDataKey: []byte{'k', 'e', 'y'},
		},
		Content: []string{target1.Id(), target2.Id(), target3.Id()},
	}
	vDb := db.Db{
		VFiles:  map[string]db.VirtFile{"target.1": target1, "target.2": target2, "target.3": target3},
		Bundles: map[string]db.Bundle{"bundleId": bundle},
	}

	//--------------------------------------------------------------------------------------------------

	// find all target files
	if i, off, ok := posInBundle(bundle, vDb, target1); i != 0 || off != 0 || ok != true {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}
	if i, off, ok := posInBundle(bundle, vDb, target2); i != 1 || off != 31 || ok != true {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}
	if i, off, ok := posInBundle(bundle, vDb, target3); i != 2 || off != 63 || ok != true {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}

	// nil test
	if i, off, ok := posInBundle(db.Bundle{}, vDb, target1); i != 0 || off != 0 || ok != false {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}
	if i, off, ok := posInBundle(bundle, db.Db{}, target2); i != 0 || off != 0 || ok != false {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}
	if i, off, ok := posInBundle(bundle, vDb, db.VirtFile{}); i != 0 || off != 0 || ok != false {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}

	// error: file not found in bundle
	target3.RelPath = "targetErr"
	if i, off, ok := posInBundle(bundle, vDb, target3); i != 0 || off != 0 || ok != false {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}

	// error:  bundle content link error
	delete(vDb.VFiles, "target.2")
	if i, off, ok := posInBundle(bundle, vDb, target2); i != 0 || off != 0 || ok != false {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}

	// error:  part check fail
	target1.Parts = make([]db.VFilePart, 2)
	vDb.VFiles["target.1"] = target1
	if i, off, ok := posInBundle(bundle, vDb, target1); i != 0 || off != 0 || ok != false {
		t.Errorf("i=%d, off=%d, ok=%v", i, off, ok)
	}

}

func Test_bundleOrPart(t *testing.T) {

	dummy := db.VirtFile{
		RelPath:      "dummy",
		FileSize:     99,
		Parts:        []db.VFilePart{{StorageSize: 99}},
		AlsoInBundle: "bundleId",
	}
	file1 := db.VirtFile{
		RelPath:      "file.1",
		FileSize:     31,
		Parts:        []db.VFilePart{{StorageSize: 31, StorageName: "file1.part"}},
		AlsoInBundle: "bundleId",
	}
	file2 := db.VirtFile{
		RelPath:      "file.2",
		FileSize:     32,
		Parts:        []db.VFilePart{{StorageSize: 32, StorageName: "file2.part"}},
		AlsoInBundle: "",
	}
	bundle := db.Bundle{
		VFilePart: db.VFilePart{
			StorageName:  "bundleId",
			CryptDataKey: []byte{'k', 'e', 'y'},
			StorageSize:  170,
		},
		Content: []string{dummy.Id(), file1.Id()},
	}
	vDb := db.Db{
		VFiles:  map[string]db.VirtFile{"dummy": dummy, "file.1": file1, "file.2": file2},
		Bundles: map[string]db.Bundle{"bundleId": bundle},
	}
	service := impl.NewRamService(nil, impl.DebugOff)
	_, _ = service.Save(file1.Parts[0].StorageName, bytes.NewReader(make([]byte, file1.FileSize)), 0)
	_, _ = service.Save(file2.Parts[0].StorageName, bytes.NewReader(make([]byte, file2.FileSize)), 0)
	_, _ = service.Save(bundle.StorageName, bytes.NewReader(make([]byte, bundle.StorageSize)), 0)
	_ = service.Update()

	//--------------------------------------------------------------------------------------------------

	// ERROR: file not found
	if _, off, n, key, err := bundleOrPart(dummy, vDb, service, true); err != os.ErrNotExist || off != 0 || len(key) != 0 || n != 0 {
		t.Fatalf("off=%d, n=%d, key=%d, err=%v", off, n, len(key), err)
	}

	// OK: from bundle
	if sf, off, n, key, err := bundleOrPart(file1, vDb, service, true); err != nil || sf.Name() != "bundleId" || off != 99 || len(key) != 3 || n != 31 {
		name := "NIL"
		if sf != nil {
			name = sf.Name()
		}
		t.Fatalf("sf=%s, off=%d, n=%d, key=%d, err=%v", name, off, n, len(key), err)
	}

	// OK: from part
	if sf, off, n, key, err := bundleOrPart(file2, vDb, service, true); err != nil || sf.Name() != "file2.part" || off != 0 || len(key) != 0 || n != 32 {
		name := "NIL"
		if sf != nil {
			name = sf.Name()
		}
		t.Fatalf("sf=%s, off=%d, n=%d, key=%d, err=%v", name, off, n, len(key), err)
	}

	// ERROR bundle: get from part
	file1.AlsoInBundle = "err"
	if sf, off, n, key, err := bundleOrPart(file1, vDb, service, true); err != nil || sf.Name() != "file1.part" || off != 0 || len(key) != 0 || n != 31 {
		name := "NIL"
		if sf != nil {
			name = sf.Name()
		}
		t.Fatalf("sf=%s, off=%d, n=%d, key=%d, err=%v", name, off, n, len(key), err)
	}
}
