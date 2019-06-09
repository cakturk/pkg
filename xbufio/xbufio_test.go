package xbufio

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestWriteSeekCloser(t *testing.T) {
	var (
		hdr = []byte{'h', 'd', 'r', '0'}
	)
	f, err := ioutil.TempFile("", "test.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name()) // clean up

	w := NewWriteSeekCloser(f)
	w.Write(make([]byte, 4))
	w.Write([]byte("arbitrary content"))
	_, err = w.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write(hdr)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("trailer"))

	name := f.Name()
	w.Close()

	content, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	got := content[:len(hdr)]
	if !bytes.Equal(got, hdr) {
		t.Errorf("got: %q, want: %q", content, hdr)
	}
}
