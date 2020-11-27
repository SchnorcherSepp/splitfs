package enc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var (
	testCryptHashSecret   []byte
	testCryptCryptSecret  []byte
	testCryptChunkHash    []byte
	testCryptChunkKey     []byte
	testCryptIndexSecret  []byte
	testCryptKeyFile      string
	testCryptKeyFileFail  string
	testCryptKeyFileWrite string
)

func init() {

	// key file testCryptData
	testCryptData, _ := hex.DecodeString("60a47fe220af89723bebda9fb741b479e15b74c817df1326b26d807d086376f6f3fe03a457d8458168cdc89f09303fe570f51305b48180e7d9fc6ef3e6aa2796915d5ca065469277d7a7eb4983f6dbcd932180cb6115bf1334c725a72b9be480b35a30a821f38a9b44660bdf0baabdf6391ad67fa1b5484503751d9afe0d4cf0")
	testCryptDataFail, _ := hex.DecodeString("60a47fe220af68cdc89f09303fe570f51305b48180e7d9fc6ef3e6aa2796915d5ca065469277d7a7eb4983f6dbcd932180cb6115bf1334c725a72b9be480b35a30a821f38a9b44660bdf0baabdf639b35a30a821f38a9b44660bdf0baabdf639b35a30a821f38a9b44660bdf0baabdf639b35a30a821f38a9b44660bdf0baabdf6391ad67fa1b5484503751d9afe0d4cf0")

	// file paths
	testCryptKeyFile = path.Join(os.TempDir(), "testCryptKeyFile.dat")
	testCryptKeyFileFail = path.Join(os.TempDir(), "testCryptKeyFileFail.dat")
	testCryptKeyFileWrite = path.Join(os.TempDir(), "testCryptKeyFileWrite.dat")

	err := ioutil.WriteFile(testCryptKeyFile, testCryptData, 0600)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(testCryptKeyFileFail, testCryptDataFail, 0600)
	if err != nil {
		panic(err)
	}

	// init other stuff
	testCryptHashSecret, _ = hex.DecodeString("d25e1be922e922bfe6492218d42bf0f8f3753ce6de030a78cf38a7c47e4b5882999baffa6c40d790bde0b30ac675af5a2b60f1026bf30ffe50656f17a0a4d68e")
	testCryptCryptSecret, _ = hex.DecodeString("e4c91c0559eb3db0e4d1df7d3d5a394619758231c2fe07ea0d7de2f6f8802ea539c46609a8b574d1ac320ee0ff08cf9c93caa3e82e031fd6377c62ee2a0b8948")
	testCryptChunkKey, _ = hex.DecodeString("1f685083dcddadb70c3d9d93da8eabb42176a09e2784d5766c06302ef542d2db")
	testCryptChunkHash = []byte("testparthash")
	testCryptIndexSecret, _ = hex.DecodeString("de936cc4451729817a60b3b8d66921cf7e39760ee1f7b64c4b539aba7a83dbb1d93d58ce44a7da8bf6b1854ac1e45ce3c4915449fe51b5988a6686b59b73e28a")
}

// ================================================================================================================== //

func TestDataKey(t *testing.T) {
	k := KeyFile{
		cryptSecret: testCryptCryptSecret,
	}

	key := k.DataKey(testCryptChunkHash)

	if !bytes.Equal(testCryptChunkKey, key) {
		t.Errorf("testCryptChunkKey %x is not %x", key, testCryptChunkKey)
	}
}

func TestLoadKeyFileFail(t *testing.T) {
	_, err := LoadKeyFile(testCryptKeyFileFail)
	if err == nil {
		t.Error("no error")
	}
}

func TestLoadKeyFile(t *testing.T) {
	k, err := LoadKeyFile(testCryptKeyFile)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(k.cryptSecret, testCryptCryptSecret) {
		t.Errorf("testCryptCryptSecret %x is not %x", k.cryptSecret, testCryptCryptSecret)
	}
	if !bytes.Equal(k.hashSecret, testCryptHashSecret) {
		t.Errorf("testCryptHashSecret %x is not %x", k.hashSecret, testCryptHashSecret)
	}
	if !bytes.Equal(k.indexSecret, testCryptIndexSecret) {
		t.Errorf("testCryptIndexSecret %x is not %x", k.indexSecret, testCryptIndexSecret)
	}
}

func TestNewRandomKeyfile(t *testing.T) {
	_ = os.Remove(testCryptKeyFileWrite)

	err := CreateKeyFile(testCryptKeyFileWrite)
	if err != nil {
		t.Error(err)
	}
}

func TestChunkName(t *testing.T) {
	hashsecret := []byte("oijajfoiajfdoiajsdojassdfo")
	parthash := []byte("ich bin ein kleiner knuddeliger part")
	enchash, _ := hex.DecodeString("01a3a9314eb0357c3eb0fd8ddb88cd0c90423c38f2b9b0a808334999dce717d0b3cda79eab836433f8c4162f3270c5af10f0248d13b931978b0ddd48f207da07")

	k := KeyFile{hashSecret: hashsecret}
	b := k.CryptName(parthash)

	if b != fmt.Sprintf("%x", enchash) {
		t.Errorf("TestCalcChunkCryptHash:\n%v\n%v", b, enchash)
	}
}
