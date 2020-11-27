package enc_test

import (
	"fmt"
	enc "github.com/SchnorcherSepp/splitfs/encoding"
	"reflect"
	"testing"
)

func TestCompress(t *testing.T) {

	// wrong input
	b, r, err := enc.Compress(nil)
	if err != nil || len(b) != 0 || r != 1 {
		t.Fatalf("fail")
	}
	_, _, err = enc.Compress(make([]byte, 0))
	if err != nil || len(b) != 0 || r != 1 {
		t.Fatalf("fail")
	}
	//----------------------------------
	b, err = enc.Decompress(nil)
	if err != nil || len(b) != 0 {
		t.Fatalf("fail")
	}
	_, err = enc.Decompress(make([]byte, 0))
	if err != nil || len(b) != 0 {
		t.Fatalf("fail")
	}
	_, err = enc.Decompress(make([]byte, 3))
	if fmt.Sprintf("%v", err) != "magic number invalid" {
		t.Fatalf("wrong error: %v", err)
	}
	_, err = enc.Decompress(make([]byte, 4))
	if fmt.Sprintf("%v", err) != "invalid input: magic number mismatch" {
		t.Fatalf("wrong error: %v", err)
	}

	// TEST
	data := []byte{0x00}
	out, ratio, err := enc.Compress(data)
	if len(out) != 14 || ratio != 14 || err != nil {
		t.Fatalf("fail: r=%f, l=%d, e=%v", ratio, len(out), err)
	}
	out, err = enc.Decompress(out)
	if len(out) != len(data) || !reflect.DeepEqual(out, data) || err != nil {
		t.Fatalf("fail: l=%d, e=%v", len(out), err)
	}

	// TEST
	data = make([]byte, 100)
	out, ratio, err = enc.Compress(data)
	if len(out) != 21 || ratio != 0.21 || err != nil {
		t.Fatalf("fail: r=%f, l=%d, e=%v", ratio, len(out), err)
	}
	out, err = enc.Decompress(out)
	if len(out) != len(data) || !reflect.DeepEqual(out, data) || err != nil {
		t.Fatalf("fail: l=%d, e=%v", len(out), err)
	}

	// TEST
	data = []byte("This is a splitStorage. A lively splitStorage. an English splitStorage text. hi, ha, ho. a a a a a a a a a a a a a a a a a ")
	out, ratio, err = enc.Compress(data)
	if len(out) != 79 || err != nil {
		t.Fatalf("fail: r=%f, l=%d, e=%v", ratio, len(out), err)
	}
	out, err = enc.Decompress(out)
	if len(out) != len(data) || !reflect.DeepEqual(out, data) || err != nil {
		t.Fatalf("fail: l=%d, e=%v", len(out), err)
	}

}
