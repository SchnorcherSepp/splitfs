package enc_test

import (
	"bytes"
	"encoding/hex"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	"testing"
)

func TestCryptBytes(t *testing.T) {
	data := []byte("Das ist ein sehr langer und geheimer text den ich hier entschluessel will! Jajaja, so ist das. Geheim und geheimer und so ein Zeug! Penis!?= ENDE")
	encData0, _ := hex.DecodeString("5a81c011433c79455bb7a3cbcdcc33e77dd25f6b859c876dd9c0a292476e05b4463e5ef33d88e49099291964936f2b824e92bfa9e135f943b50f63869940fcc4c2ca435147ab73c4c116ea40cc46ede6d93b8b5596d8a4b1471e55883874a6c25cbde345f0d77df47658e2c0661e43adbf6350eac073866e1b9b26248c0253a82d1d77504d2b2444cb89e1f9604f51d781")
	encData1G, _ := hex.DecodeString("79db76e0a5ec269d4c8b20105592123aa8125d08e0355d7b4fa80cb4d83ec4aa575c1f8b2095926aaee5c173416aab638ca55ee281f183302601e0ce6e2f0b2e3bda2ca8c9d8ab8a895b07c6d02f3d3a4c3c2dc2e046173690cc8fe0d319e347ac28baae5aabd0f0f868ba004198912b1e458f28b5b7306bbefeb31820279eb7badc05ff84a4c87aa4b0eb8defcb691b51")
	key, _ := hex.DecodeString("8374fd0d213ab30f4eb6ae85d43dd4981234b566fff84cfb161e3500b709563e")

	// start offset 0
	for i := 0; i <= len(data)-50; i++ {
		work := make([]byte, len(encData0))
		copy(work, encData0)

		enc.CryptBytes(work[i:], int64(i), key)

		if !bytes.Equal(work[i:], data[i:]) {
			t.Errorf("%s\n is not\n %s\n", work[i:], data[i:])
		}
	}

	// start offset 1000000000
	for i := 1000000000; i <= len(data)-50; i++ {
		work := make([]byte, len(encData1G))
		copy(work, encData1G)

		enc.CryptBytes(work[i:], int64(i), key)

		if !bytes.Equal(work[i:], data[i:]) {
			t.Errorf("%s\n is not\n %s\n", work[i:], data[i:])
		}
	}
}
