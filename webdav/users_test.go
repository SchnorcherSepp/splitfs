package webdav

import (
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"io/ioutil"
	"testing"
	"time"
)

func Test_userUpdate(t *testing.T) {
	us := initUsers("../test/webdav.users", impl.DebugLow)
	us.updateInterval = 1 * time.Second // change for faster test

	/*
		userUpdate return codes
		 > return -1 // EXIT: too early
		 > return -2 // EXIT: file not found
		 > return -3 // EXIT: file not changed
	*/

	// right after init
	if us.userUpdate() != -1 { // EXIT: too early
		t.Errorf("wrong return code")
	}

	// after init
	time.Sleep(400 * time.Millisecond)
	if us.userUpdate() != -1 { // EXIT: too early
		t.Errorf("wrong return code")
	}

	//-------------------------------------------------

	// no change
	time.Sleep(999 * time.Millisecond)
	if us.userUpdate() != -3 { // EXIT: file not changed
		t.Errorf("wrong return code")
	}

	// after 'no change'
	time.Sleep(400 * time.Millisecond)
	if us.userUpdate() != -1 { // EXIT: too early
		t.Errorf("wrong return code")
	}

	//-------------------------------------------------

	// file error
	time.Sleep(999 * time.Millisecond)
	oldPath := us.userFile
	us.userFile = "no/file/found/error"
	if us.userUpdate() != -2 { // EXIT: file not found
		t.Errorf("wrong return code")
	}
	us.userFile = oldPath

	// after 'file error'
	time.Sleep(400 * time.Millisecond)
	if us.userUpdate() != -1 { // EXIT: too early
		t.Errorf("wrong return code")
	}

	// no change
	time.Sleep(999 * time.Millisecond)
	if us.userUpdate() != -3 { // EXIT: file not changed
		t.Errorf("wrong return code")
	}

	//-------------------------------------------------

	// change
	time.Sleep(1111 * time.Millisecond)
	b, _ := ioutil.ReadFile(us.userFile)
	_ = ioutil.WriteFile(us.userFile, b, 0600)
	if us.userUpdate() <= 0 { // EXIT: OK
		t.Errorf("wrong return code")
	}

	// after 'change'
	time.Sleep(400 * time.Millisecond)
	if us.userUpdate() != -1 { // EXIT: too early
		t.Errorf("wrong return code")
	}
}

func Test_Get(t *testing.T) {
	// init
	us := initUsers("../test/webdav.users", impl.DebugOff)
	if len(us.users) != 1 {
		t.Fatal("wrong user count")
	}

	// get wrong user
	_, ok := us.Get("b")
	if ok != false {
		t.Fatal("get wrong user")
	}

	// get user
	u, ok := us.Get("a")
	if ok != true {
		t.Fatal("get error")
	}

	// check user
	if u.Username != "a" {
		t.Fatal("wrong")
	}
	if u.PassHash != "$2y$12$PAKzhy7Q1PIlsgWFgmkQNutoUiDSHiFDtbRyEGbWbZv9yz9WPB2ra" {
		t.Fatal("wrong")
	}
	if len(u.pathPrefix) != 3 {
		t.Fatal("wrong")
	}

	// prefix
	if u.Allowed("file") != false {
		t.Fatal("wrong")
	}
	if u.Allowed("folder") != false {
		t.Fatal("wrong")
	}
	if u.Allowed("folder1") != true {
		t.Fatal("wrong")
	}
	if u.Allowed("folder2") != false {
		t.Fatal("wrong")
	}
	if u.Allowed("/folder2") != true {
		t.Fatal("wrong")
	}
	if u.Allowed("/folder2/") != true {
		t.Fatal("wrong")
	}
	if u.Allowed("/folder2/abc") != true {
		t.Fatal("wrong")
	}
	if u.Allowed("/folder3") != false {
		t.Fatal("wrong")
	}
	if u.Allowed("/folder3/") != true {
		t.Fatal("wrong")
	}
	if u.Allowed("/folder3/abc") != true {
		t.Fatal("wrong")
	}
}
