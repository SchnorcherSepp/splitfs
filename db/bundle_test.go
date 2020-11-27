package db_test

import (
	"github.com/SchnorcherSepp/splitfs/db"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"math/rand"
	"os"
	"path"
	"testing"
)

func TestMakeBundles(t *testing.T) {

	// init test files
	rootPath := path.Join(os.TempDir(), "Test_MakeBundles")
	initTestDir(t, rootPath)

	// load keyfile
	keyFile, err := enc.LoadKeyFile(path.Join(os.TempDir(), "testCryptKeyFile.dat"))
	if err != nil {
		t.Error(err)
	}

	// scan
	vDb, _, _, err := db.FromScan(rootPath, db.NewDb(), impl.DebugOff, keyFile)
	if err != nil {
		t.Fatal(err)
	}

	// make bundle
	vDb.MakeBundles(keyFile, impl.DebugHigh)

	// ----------------------------------------------------------

	// TEST: bundles and files are linked
	for bundleId, bundle := range vDb.Bundles {
		// check bundle side
		for _, fileId := range bundle.Content {
			file := vDb.VFiles[fileId]
			if file.AlsoInBundle != bundleId || bundleId != bundle.Id() {
				t.Fatalf("bundle id error")
			}
		}
	}
	for _, file := range vDb.VFiles {
		// check file side
		if file.AlsoInBundle != "" {
			bundle := vDb.Bundles[file.AlsoInBundle]
			if file.AlsoInBundle != bundle.Id() {
				t.Fatalf("bundle id error")
			}
		}
	}

	// TEST sample
	// zero -> no bundle
	if vDb.VFiles["a_zero"].AlsoInBundle != "" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["b_zero.txt"].AlsoInBundle != "" {
		t.Fatal("wrong")
	}
	// BUNDLE 1
	if vDb.VFiles["c_1byte"].AlsoInBundle != db.BundlePrefix+"e1b76dc5e5161c69e94a2bdb196193cd69d2e463087fc5a613ed5a986ad9355f9c158f3ecf3bbf9f2aed7e813d9b03672ab0501324f8303bd12a28667fe7e25d" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["d_1byte.dat"].AlsoInBundle != db.BundlePrefix+"e1b76dc5e5161c69e94a2bdb196193cd69d2e463087fc5a613ed5a986ad9355f9c158f3ecf3bbf9f2aed7e813d9b03672ab0501324f8303bd12a28667fe7e25d" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["e_SectorSize"].AlsoInBundle != db.BundlePrefix+"e1b76dc5e5161c69e94a2bdb196193cd69d2e463087fc5a613ed5a986ad9355f9c158f3ecf3bbf9f2aed7e813d9b03672ab0501324f8303bd12a28667fe7e25d" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["f_MaxFileSizeForCompression"].AlsoInBundle != db.BundlePrefix+"e1b76dc5e5161c69e94a2bdb196193cd69d2e463087fc5a613ed5a986ad9355f9c158f3ecf3bbf9f2aed7e813d9b03672ab0501324f8303bd12a28667fe7e25d" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["g_MaxFileSizeForCompression"].AlsoInBundle != db.BundlePrefix+"e1b76dc5e5161c69e94a2bdb196193cd69d2e463087fc5a613ed5a986ad9355f9c158f3ecf3bbf9f2aed7e813d9b03672ab0501324f8303bd12a28667fe7e25d" {
		t.Fatal("wrong")
	}

	//  BUNDLE 2
	if vDb.VFiles["h_MaxFileSizeForCompression"].AlsoInBundle != db.BundlePrefix+"2b1a5fa9ffe7b7f0a257c869f9f5891000ad3ba3b68001fb0a03ff0a147fffef4c861e77397025148610395f9bc6b0751781fec44fc5def91fd74da0123981a4" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["i_MaxFileSizeToBundle"].AlsoInBundle != db.BundlePrefix+"2b1a5fa9ffe7b7f0a257c869f9f5891000ad3ba3b68001fb0a03ff0a147fffef4c861e77397025148610395f9bc6b0751781fec44fc5def91fd74da0123981a4" {
		t.Fatal("wrong")
	}

	// (medium file)  BUNDLE 3
	if vDb.VFiles["mediumFile1"].AlsoInBundle != db.BundlePrefix+"ee0316a2a28f01670395819921044b41d69ee689ff458313622526e6191f7c4f9af981895aa9213b2292be6a28be0b8d628765d6f1080f27af839f3de83463de" {
		t.Fatalf("wrong")
	}
	if vDb.VFiles["mediumFile2"].AlsoInBundle != db.BundlePrefix+"ee0316a2a28f01670395819921044b41d69ee689ff458313622526e6191f7c4f9af981895aa9213b2292be6a28be0b8d628765d6f1080f27af839f3de83463de" {
		t.Fatalf("wrong")
	}
	if vDb.VFiles["mediumFile3"].AlsoInBundle != db.BundlePrefix+"ee0316a2a28f01670395819921044b41d69ee689ff458313622526e6191f7c4f9af981895aa9213b2292be6a28be0b8d628765d6f1080f27af839f3de83463de" {
		t.Fatal("wrong")
	}

	// no bundle (too big)
	if vDb.VFiles["j_MaxFileSizeToBundle"].AlsoInBundle != "" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["k_MaxFileSizeToBundle"].AlsoInBundle != "" {
		t.Fatal("wrong")
	}
	if vDb.VFiles["l_BufferSize"].AlsoInBundle != "" {
		t.Fatal("wrong")
	}
}

// ----------  HELPER  -----------------------------------------------------------------------------------------------//

func initTestDir(t *testing.T, rootPath string) {
	_ = os.Mkdir(rootPath, 0700)

	// write test files
	writeTestFile(t, path.Join(rootPath, "a_zero"), 0, false)     // no bundle (size == 0)
	writeTestFile(t, path.Join(rootPath, "b_zero.txt"), 0, false) // no bundle (size == 0)

	writeTestFile(t, path.Join(rootPath, "c_1byte"), 1, false)     // OK BUNDLE 1
	writeTestFile(t, path.Join(rootPath, "d_1byte.dat"), 1, false) // OK BUNDLE 1

	writeTestFile(t, path.Join(rootPath, "e_SectorSize"), interf.SectorSize, false) // OK BUNDLE 1

	writeTestFile(t, path.Join(rootPath, "f_MaxFileSizeForCompression"), db.MaxFileSizeForCompression-1, false) // OK  BUNDLE 1
	writeTestFile(t, path.Join(rootPath, "g_MaxFileSizeForCompression"), db.MaxFileSizeForCompression, false)   // OK  BUNDLE 1
	writeTestFile(t, path.Join(rootPath, "h_MaxFileSizeForCompression"), db.MaxFileSizeForCompression+1, false) // OK  BUNDLE 2

	writeTestFile(t, path.Join(rootPath, "mediumFile1"), 12*1024*2, true) // OK (medium file)  BUNDLE 3
	writeTestFile(t, path.Join(rootPath, "mediumFile2"), 12*1024*2, true) // OK (medium file)  BUNDLE 3
	writeTestFile(t, path.Join(rootPath, "mediumFile3"), 12*1024*2, true) // no bundle (no bundle only with one file)

	writeTestFile(t, path.Join(rootPath, "i_MaxFileSizeToBundle"), db.MaxFileSizeToBundle-1, false) // OK BUNDLE 2
	writeTestFile(t, path.Join(rootPath, "j_MaxFileSizeToBundle"), db.MaxFileSizeToBundle, false)   // no bundle (too big; < MaxFileSizeForCompression!)
	writeTestFile(t, path.Join(rootPath, "k_MaxFileSizeToBundle"), db.MaxFileSizeToBundle+1, false) // no bundle (too big)

	writeTestFile(t, path.Join(rootPath, "l_BufferSize"), 16777216-1, false) // no bundle (too big)
}

func writeTestFile(t *testing.T, filePath string, size int, random bool) {
	rnd := rand.New(rand.NewSource(876534 + int64(size)))

	// file exist?
	_, err := os.Stat(filePath)
	if err == nil {
		return
	}

	// create file
	fh, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	// write file
	off := 0
	toggle := false
	for size > off {
		toggle = !toggle

		need := size - off
		if need < len(filePath) {
			filePath = filePath[:need]
		}

		data := []byte(filePath)
		if random && toggle {
			data = make([]byte, len(data))
			_, _ = rnd.Read(data)
		}

		n, err := fh.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		off += n
	}
}
