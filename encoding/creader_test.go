package enc_test

import (
	"bytes"
	"encoding/hex"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	"io"
	"io/ioutil"
	"testing"
)

func TestCryptoReader(t *testing.T) {
	key, _ := hex.DecodeString("1f685083dcddadb70c3d9d93da8eabb42176a09e2784d5766c06302ef542d2db")

	// reader is nil
	r := enc.CryptoReader(nil, 0, key)
	if n, err := r.Read(make([]byte, 12)); n != 0 || err == nil {
		t.Error("no error")
	}
	if err := r.Close(); err != nil {
		t.Error(err)
	}

	// empty inner reader
	buf := make([]byte, 4)
	r = enc.CryptoReader(ioutil.NopCloser(bytes.NewReader([]byte{})), 0, key)
	if n, err := r.Read(buf); n != 0 || err != io.EOF {
		t.Errorf("n=%d, err=%v", n, err)
	}
	if err := r.Close(); err != nil {
		t.Error(err)
	}

	// inner reader with 1 byte; read 4 bytes
	r = enc.CryptoReader(ioutil.NopCloser(bytes.NewReader([]byte{'X'})), 0, key)
	if n, err := r.Read(buf); n != 1 || err != nil {
		t.Errorf("n=%d, err=%v", n, err)
	}
	if n, err := r.Read(buf); n != 0 || err != io.EOF { // second read
		t.Errorf("n=%d, err=%v", n, err)
	}
	if err := r.Close(); err != nil {
		t.Error(err)
	}

	// plain text
	plain := make([]byte, 32*1024+1)
	for i := range plain {
		plain[i] = byte(i % 256)
	}

	// read/test/encrypt/decrypt plain text
	buf1 := make([]byte, 3)
	buf2 := make([]byte, 3)
	for off := int64(0); off < int64(len(plain)-3); off++ {
		// create reader
		r1 := enc.CryptoReader(ioutil.NopCloser(bytes.NewReader(plain[off:])), off, key)
		r2 := enc.CryptoReader(enc.CryptoReader(ioutil.NopCloser(bytes.NewReader(plain[off:])), off, key), off, key)

		// read
		n1, err1 := r1.Read(buf1)
		if err1 != nil {
			t.Errorf("ERROR: off=%d", off)
			t.Errorf("n=%d, err=%v", n1, err1)
		}
		n2, err2 := r2.Read(buf2)
		if err2 != nil {
			t.Errorf("ERROR: off=%d", off)
			t.Errorf("n=%d, err=%v", n2, err2)
		}

		// check
		// n equal (same buffer size)
		// errors equal
		if n1 != n2 || err1 != err2 {
			t.Errorf("ERROR: off=%d", off)
			t.Errorf("n=%d, err=%v", n1, err1)
			t.Errorf("n=%d, err=%v", n2, err2)
		}

		// check encryption (data)
		hitCount := 0 // enc and plain equal  (can happen)
		for i := range buf1 {
			if buf1[i] == buf2[i] {
				hitCount++
			}
			c := byte((off + int64(i)) % 256)
			if buf2[i] != c {
				t.Errorf("crypt error [%d]: %x == %x", off, buf2[i], c)
			}
		}

		// statistic check
		if hitCount > int(float32(len(plain))*0.1) {
			t.Errorf("enc error: hit=%d", hitCount)
		}
	}
}
