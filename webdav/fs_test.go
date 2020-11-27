package webdav

import (
	"bytes"
	"fmt"
	"github.com/SchnorcherSepp/splitfs/core"
	"github.com/SchnorcherSepp/splitfs/db"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	interf "github.com/SchnorcherSepp/storage/interfaces"
	"golang.org/x/net/webdav"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------  not implemented (ReadOnly)  --------------------------------------------------------------------------- //

func TestFileSystem_Mkdir_RemoveAll_Rename(t *testing.T) {
	service := impl.NewRamService(impl.NewCache(17), impl.DebugHigh)
	fs := NewFileSystem(service, make([]byte, 16), impl.DebugHigh, -1)

	// [1] Mkdir
	if err := fs.Mkdir(nil, "", 0666); err != webdav.ErrForbidden {
		t.Fatal(err)
	}

	// [2] RemoveAll
	if err := fs.RemoveAll(nil, ""); err != webdav.ErrForbidden {
		t.Fatal(err)
	}

	// [3] Rename
	if err := fs.Rename(nil, "", ""); err != webdav.ErrForbidden {
		t.Fatal(err)
	}
}

// ---------  Helper  ----------------------------------------------------------------------------------------------- //

func TestFileSystem_startUpdateLoop_checkDb(t *testing.T) {
	// vars
	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	service := impl.NewRamService(impl.NewCache(17), impl.DebugHigh)
	_ = impl.InitDemo(service)
	dbKey := make([]byte, 16)

	// NewFileSystem:  without loop
	startLogTests(stdoutBuf)
	{
		_ = NewFileSystem(service, dbKey, impl.DebugHigh, -1)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "", "UpdateLoop", t, "Test A")

	// NewFileSystem:  without loop
	startLogTests(stdoutBuf)
	{
		_ = NewFileSystem(service, dbKey, impl.DebugHigh, 0)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "", "UpdateLoop", t, "Test B")

	// NewFileSystem:  with loop & NO DB
	startLogTests(stdoutBuf)
	{
		_ = NewFileSystem(service, dbKey, impl.DebugHigh, 1)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "start Update loop with 1 sec", "", t, "Test C")

	//-----------------------------------------------------------------------------------------

	// NewFileSystem:  with loop & zero DB
	startLogTests(stdoutBuf)
	{
		_, _ = service.Save(core.IndexName, bytes.NewReader(make([]byte, 0)), 0)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "webdav/checkDb: EOF", "", t, "Test D")

	// NewFileSystem:  with loop & short DB
	startLogTests(stdoutBuf)
	{
		_, _ = service.Save(core.IndexName, bytes.NewReader(make([]byte, 1)), 0)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "db/enc2zip: size check fail", "", t, "Test E")

	// NewFileSystem:  with loop & invalid DB
	startLogTests(stdoutBuf)
	{
		_, _ = service.Save(core.IndexName, bytes.NewReader(make([]byte, 100)), 0)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "message authentication failed", "", t, "Test F")

	// NewFileSystem:  with loop & with DB
	startLogTests(stdoutBuf)
	{
		writeDb(service, dbKey, t)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "download db", "", t, "Test G")

	// NewFileSystem:  with loop & with DB no CHANGE
	startLogTests(stdoutBuf)
	{
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "successful", "download db", t, "Test H")

	// NewFileSystem:  with loop & with DB CHANGE
	startLogTests(stdoutBuf)
	{
		writeDb(service, dbKey, t)
		time.Sleep(2 * time.Second)
	}
	endLogTests(stdoutBuf, "download db", "", t, "Test I")
}

//====================================================================================================================//

func startLogTests(buf *bytes.Buffer) {
	buf.Reset()
	log.SetOutput(buf)
}

func endLogTests(buf *bytes.Buffer, in, notIn string, t *testing.T, description string) {
	log.SetOutput(os.Stdout)
	s := buf.String()

	if in != "" && !strings.Contains(s, in) {
		fmt.Printf("%s", s)
		t.Fatalf("ERROR (%s) NOT IN '%s'", description, in)
	}

	if notIn != "" && strings.Contains(s, notIn) {
		fmt.Printf("%s", s)
		t.Fatalf("ERROR (%s) IN '%s'", description, notIn)
	}
}

func writeDb(service interf.Service, indexKey []byte, t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := db.ToWriter(db.NewDb(), indexKey, buf); err != nil {
		t.Error(err)
	}
	if _, err := service.Save(core.IndexName, buf, 0); err != nil {
		t.Error(err)
	}
}
