package webdav

import (
	"github.com/SchnorcherSepp/splitfs/db"
	"testing"
)

func Test_newFileInfo(t *testing.T) {
	f := db.VirtFile{
		RelPath:  "/i/am/a/path/haha.txt", // for .Name()
		FileSize: 1122,                    // for .Size()
		MTime:    3344,                    // for .ModTime()
		IsDir:    false,                   // for .IsDir() and .Mode()
	}

	// check file
	info := newFileInfo(f)
	if info.IsDir() != false {
		t.Fatalf("error")
	}
	if info.Name() != "haha.txt" {
		t.Fatalf("error")
	}
	if info.Size() != 1122 {
		t.Fatalf("error")
	}
	if info.Mode() != 0666 {
		t.Fatalf("error")
	}
	if info.ModTime().Unix() != 3344 {
		t.Fatalf("error")
	}
	if info.Sys() != nil {
		t.Fatalf("error")
	}

	// check folder
	f.IsDir = true // for .IsDir() and .Mode()
	info = newFileInfo(f)
	if info.IsDir() != true { // changed
		t.Fatalf("error")
	}
	if info.Name() != "haha.txt" {
		t.Fatalf("error")
	}
	if info.Size() != 1122 {
		t.Fatalf("error")
	}
	if info.Mode() != 0777 { // changed
		t.Fatalf("error")
	}
	if info.ModTime().Unix() != 3344 {
		t.Fatalf("error")
	}
	if info.Sys() != nil {
		t.Fatalf("error")
	}
}
