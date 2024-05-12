package cygwin

import (
	"bytes"
	"testing"
)

func TestParse(t *testing.T) {
	port, uuid, err := parse("../../ccrap/c/x/foo.sock")
	t.Errorf("port: %d %+#v error: %v", port, uuid, err)
}

func TestHandshake(t *testing.T) {
	var b bytes.Buffer
	uuid := [4]uint32{0x686a723f, 0x83eea6e6, 0x82b318b1, 0x9be411bc}
	err := handshake(&b, uuid, 1000, 1001, 1002)
	t.Errorf("err: %v 1000=%08x %#v", err, 1002, b.Bytes())
}

func TestDial(t *testing.T) {
	c, err := dial("C:\\cygwin64\\tmp\\foo.sock")
	if err != nil {
		t.Fatal(err)
	}
	c.Write([]byte("hello world"))
	c.Close()
}
