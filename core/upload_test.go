package core_test

import (
	"bytes"
	"crypto/md5"
	"crypto/sha512"
	"fmt"
	"github.com/SchnorcherSepp/splitfs/core"
	"github.com/SchnorcherSepp/splitfs/db"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"
)

var (
	testUploadFolderPath = path.Join(os.TempDir(), "scanTestFolder")
	testUploadKeyFile    *enc.KeyFile
)

// init '/tmp/scanTestFolder/'
func init() {
	// create temp folder
	_ = os.Mkdir(testUploadFolderPath, 0700)

	// generate files
	s := impl.NewRamService(nil, impl.DebugOff)
	if err := impl.InitDemo(s); err != nil {
		panic(err)
	}

	// write files
	for _, f := range s.Files().All() {
		// ignore update-test files
		if strings.HasPrefix(f.Name(), "small-test-file-") && f.Name() != "small-test-file-1.dat" {
			continue
		}

		// check exist
		p := path.Join(testUploadFolderPath, f.Name())
		if _, err := os.Stat(p); err == nil {
			continue
		}

		// read & write
		r, err := s.Reader(f, 0)
		if err != nil && err != io.EOF {
			panic(err)
		}

		b := make([]byte, 0)
		if err != io.EOF {
			b, err = ioutil.ReadAll(r)
			if err != nil {
				panic(err)
			}
		}

		err = ioutil.WriteFile(p, b, 0600)
		if err != nil {
			panic(err)
		}
	}

	// write big test file for "next-part-test"
	p := path.Join(testUploadFolderPath, "next-part-test-file.dat")
	if _, err := os.Stat(p); err != nil {
		err := ioutil.WriteFile(p, make([]byte, db.PartSize+1), 0600)
		if err != nil {
			panic(err)
		}
	}

	// load keyfile
	var err error
	testUploadKeyFile, err = enc.LoadKeyFile(path.Join(os.TempDir(), "testCryptKeyFile.dat"))
	if err != nil {
		panic(err)
	}
}

func TestUpload(t *testing.T) {

	// scan folder
	vDb, _, _, err := db.FromScan(testUploadFolderPath, db.NewDb(), impl.DebugOff, testUploadKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	// make bundle
	vDb.MakeBundles(testUploadKeyFile, impl.DebugOff)

	// -----  TEST  ---------------------------------------------------------------

	// upload to ram storage
	service := impl.NewRamService(nil, impl.DebugOff)
	err = core.Upload(testUploadFolderPath, vDb, testUploadKeyFile.IndexKey(), service, impl.DebugOff)
	if err != nil {
		t.Fatal(err)
	}

	// TEST RE-UPLOAD! (no changes!)
	{
		// LOG TEST START: write logger output to buffer
		buf := bytes.NewBufferString("") // new buffer
		log.SetOutput(buf)               // write logs to buffer

		// TEST
		err = core.Upload(testUploadFolderPath, vDb, testUploadKeyFile.IndexKey(), service, impl.DebugOff)
		if err != nil {
			t.Fatal(err)
		}

		// CHECK
		if len(buf.String()) != 0 {
			t.Fatalf("log:\n\n%s", buf.String())
		}

		// LOG TEST START: reset logger output
		log.SetOutput(os.Stdout)
	}

	// ----- CHECK --------------------------------------
	known := []string{
		"9d8d983a07bf750c2a9b03ddd69b07af", // special-file-16777216-0.330000.dat
		"723a6e6fdb88265c2c7135b800119aac", // special-file-1048575-0.990000.dat
		"0b86588868bf477aa7b4296ecc47a3aa", // special-file-1048576-0.990000.dat
		"12f61c26c9fe42d833b9ba7767b110d7", // special-file-131071-0.000000.dat
		"76848a351b845d475ef2c54714c502e0", // special-file-1048576-0.660000.dat
		"6f76b4c76c73096fbfb6964474a93060", // special-file-12582912-0.990000.dat
		"d595e917a8b4d89f6a3828d5a1cf6628", // special-file-16777217-0.000000.dat
		"c89f68cd66aae48135703c931a45eda7", // special-file-16777217-0.660000.dat
		//"????????????????????????????????", // special-file-0-0.990000.dat (0 bytes)
		"902ceb4aaa8f9e5289ad32b757994848", // special-file-1048575-0.660000.dat
		"d002fb55bf494e52d76f93a4c25fdd18", // special-file-1048576-0.330000.dat
		"01f1524122a101f1aea26b573c338365", // special-file-1048577-0.000000.dat
		"5835cf5827833e7f583a8a168cbce121", // special-file-1048577-0.330000.dat
		"286cc84d06817fd7e5ec5a792347c908", // special-file-12582912-0.330000.dat
		"17c1930ebcc1e31ebd1383ea1ce1bc75", // special-file-131071-0.660000.dat
		"f0d60adfc2e6060cca732bf36668d624", // special-file-16777216-0.660000.dat
		//"????????????????????????????????", // special-file-0-0.660000.dat (0 bytes)
		"9fe681e04c86fa3000c6b8a223d71f20", // special-file-1048575-0.330000.dat
		"b4cc7756cc3e5864ad497da328f7ab0c", // special-file-12582912-0.000000.dat
		"7b8f2dc1774bf9d10b38e1b31bba3393", // special-file-131072-0.000000.dat
		"ecb5c3f5325fee070a940ce767144de6", // special-file-12582911-0.990000.dat
		"4c75d49cbfae0b5d993cde2c4b39ef48", // special-file-12582912-0.660000.dat
		"bcbc4794ea1f8d6acbedc7c2b27c3970", // special-file-12582913-0.990000.dat
		"d1b44a0d3093785bf589fb4ac0488b38", // special-file-12582911-0.330000.dat
		"e48ccc22786055a03d0d946a64e5277b", // special-file-131073-0.000000.dat
		"607ddc7863991ea60e7e88d0a609cd37", // special-file-131073-0.990000.dat
		"4137a14cb4a4659904c1117dae628cc0", // special-file-12582913-0.660000.dat
		"563a9cec74d9d9319a02a30d7ebbf915", // special-file-131072-0.330000.dat
		"df5e6691ebcb2d1229f75d661dc15b79", // special-file-131073-0.330000.dat
		"c24571cb07b0809665416c32a944955d", // special-file-131073-0.660000.dat
		//"????????????????????????????????", // special-file-0-0.000000.dat (0 bytes)
		"b1904919fe736e3fbaf3606f589de07e", // special-file-1048576-0.000000.dat
		"ec13b332797a4e184368de7abe7ae01b", // special-file-12582911-0.660000.dat
		"e6de72e4d18105bb569d74d09329be08", // special-file-12582913-0.000000.dat
		"51b3e1980c4e16f268b9b2bd14764401", // special-file-131072-0.660000.dat
		"716f2329aa0acbb5e6e54029bfdff209", // small-test-file-1.dat
		"c7256e2df433b3284030ed11132bf37b", // big-test-file-150.dat
		"0d6b79ced7cb799c59d7bf6d83d16413", // special-file-16777216-0.000000.dat
		"459d78cf0d638f24fb64d5c26df35c4b", // special-file-16777217-0.330000.dat
		//"????????????????????????????????", // . (root folder)
		"33b4c5aec9e200649bea5002e0ae3b74", // special-file-1048575-0.000000.dat
		"4b9d18821268b38f22cd431b6d812199", // special-file-16777215-0.990000.dat
		"ebe56dc5d5c48cf4cf8877bf35ce8a2e", // special-file-1048577-0.990000.dat
		"66baf3e75f3d30c374bdca8cb978593a", // special-file-16777215-0.000000.dat
		"4aeefcfa9a2388eeb576fd329e77a961", // special-file-16777216-0.990000.dat
		"b1b8e8e4c9741835c271be3b26a80134", // special-file-16777215-0.330000.dat
		"fa045fd39fb576056142b240e5780e79", // special-file-16777215-0.660000.dat
		"019042e344932c527d1075ac5af3c46c", // special-file-1048577-0.660000.dat
		"fefa77f7197e64b1de460df7228b9df8", // special-file-131071-0.330000.dat
		"225fdfb3a358d86ad4efd765b901c42a", // special-file-131072-0.990000.dat
		//"????????????????????????????????", // special-file-0-0.330000.dat (0 bytes)
		"15b39786257cea8153ab19e13554659f", // special-file-12582911-0.000000.dat
		"1c32a59cd2033bd3707f1628c995e339", // special-file-16777217-0.990000.dat
		"a0d0616603e3ddbf04cf3b104c56b5c8", // special-file-12582913-0.330000.dat
		"6d2382442af495a5396167774257c9b7", // special-file-131071-0.990000.dat
		"e0862de1e47ae5859c34ceee23fa1595", // bundle 6x B_aa77027cbff0d3f3a6ac55efbed342fc20b228a4e0d442a8434731a36aaba5899405323b4325ba1c7d057566499b7d6e972118def39aec1e7daff24125c6ae46
		"191b19f954f0e17096cc4377d40ed991", // bundle 10x B_f81bb6d89e91f53e1cffa9f708dc48fde4db9ce2b05e7ef42c780641ac4e9f198e623827746586bf8ae16632d4b1c8a1ee6a7631c22945e1ca4ec5f8a050dea9
		"d150879873d998737e81ba4bc2c7392a", // bundle 13x B_314238503fe3cd601b3543fad6c0baf49c9912eae7fd5336bca292414cc8b1f7190b39d2894814c5f03c9698492d0d50ab6b698d8cef0afea32956bd61a371cd
		"e98c9d4f43259e118ad9de9a4d8271dd", // next-part-test-file.dat (part 0, partSize byte)
		"415290769594460e2e485922904f345d", // next-part-test-file.dat (part 1, 1 byte)
	}

	// count files
	if len(service.Files().All()) != len(known)+1 { // known+1 because of 'index.db'
		t.Error("len not equal")
	}

	// check storage md5
	for _, v := range service.Files().All() {
		if !strings.Contains(strings.Join(known, ","), v.Md5()) && v.Name() != core.IndexName {
			t.Errorf("not found: %s\t%s\t%d", v.Name(), v.Md5(), v.Size())
		}
	}

	// check db
	f, err := service.Files().ByName(core.IndexName)
	if err != nil {
		t.Error(err)
	}
	if f.Size() < 10 {
		t.Error("db size error")
	}

	// check encryption: next-part-test-file.dat (part 1)
	// content: '0x00'
	// size: 1
	// crypto offset: 0
	// key: '0x00'.....
	data := make([]byte, 1)
	// plain hash (PART 1)
	h := sha512.New()
	h.Write(data)
	plainHash := h.Sum(nil)
	// data key
	dataKey := testUploadKeyFile.DataKey(plainHash)
	// storage hash
	enc.CryptBytes(data, 0, dataKey)
	h2 := md5.New()
	h2.Write(data)
	storageHash := h2.Sum(nil)
	sh := fmt.Sprintf("%x", storageHash)
	if "415290769594460e2e485922904f345d" != sh {
		t.Errorf("wrong storage hash for part 1: %s", sh)
	}
}
