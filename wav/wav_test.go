package wav

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"unsafe"

	"github.com/siddontang/go/ioutil2"
)

func mergeBytes(t *testing.T, files ...string) []byte {
	var buf bytes.Buffer
	for _, f := range files {
		nf, err := os.Open(f)
		if err != nil {
			t.Fatal(err)
		}
		defer nf.Close()

		if _, err := buf.ReadFrom(nf); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

func mergeRead(t *testing.T, files ...string) io.ReadSeeker {
	return bytes.NewReader(mergeBytes(t, files...))
}

func TestEncode(t *testing.T) {
	parts := []string{
		"riffhdr.golden",
		"fmtchunk.golden",
		"datachunk.golden",
		"listchunk.golden",
	}
	f := mergeRead(t, parts...)
	wf, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	b := mergeBytes(t, parts...)
	if err := ioutil.WriteFile("out.wav", b, 0644); err != nil {
		t.Fatal(err)
	}

	w, err := os.OpenFile("out.wav", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	ic := InfoChunk{
		ID:   IENG,
		Text: []byte("merry christmas"),
	}
	id := InfoChunk{
		ID:   ITCH,
		Text: []byte("merry mery christmas"),
	}
	ie := InfoChunk{
		ID:   IGNR,
		Text: []byte("submarine style"),
	}

	wf.List.SubChunks = append(wf.List.SubChunks, ic, id, ie)
	// wf.List = nil

	var sz int64
	if sz, err = wf.Encode(w); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate("out.wav", sz); err != nil {
		t.Fatal(err)
	}

	// got := string(wf.List.SubChunks[3].Text)
	// if got != "foobar" {
	// 	t.Errorf("got: %s, want: %s", got, "foobar")
	// }
}

func TestDecode(t *testing.T) {
	f := mergeRead(
		t,
		"riffhdr.golden",
		"fmtchunk.golden",
		"datachunk.golden",
		"listchunk.golden",
	)
	w, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	got := string(w.List.SubChunks[3].Text)
	if got != "foobar" {
		t.Errorf("got: %s, want: %s", got, "foobar")
	}
}

func TestPackRIFF2(t *testing.T) {
	want, err := ioutil.ReadFile("riffhdr.golden")
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(want)
	var riff RIFFHdr

	if err := riff.Unpack(r); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}

	if err := riff.Pack(buf); err != nil {
		t.Fatal(err)
	}

	if got := buf.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("got: %#02v - %d,\n\t   want: %#02v - %d",
			got, len(got), want, len(want),
		)
	}
}

func TestPackFmt(t *testing.T) {
	b, err := ioutil.ReadFile("fmtchunk.golden")
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(b)
	var fmt FmtChunk

	if err := fmt.Unpack(r); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := fmt.Pack(buf); err != nil {
		t.Fatal(err)
	}
	got := buf.Bytes()
	if !bytes.Equal(got, b) {
		t.Errorf("got: %#02v - %d,\n\t   want: %#02v - %d",
			got, len(got), b, len(b),
		)
	}
}

func TestPackDataChunk(t *testing.T) {
	b, err := ioutil.ReadFile("datachunk.golden")
	if err != nil {
		t.Fatal(err)
	}
	b = b[:8] // ignore PCM samples for now
	r := bytes.NewReader(b)
	var data DataChunk

	if err := data.Unpack(r); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := data.Pack(buf); err != nil {
		t.Fatal(err)
	}
	if got := buf.Bytes(); !bytes.Equal(got, b) {
		t.Errorf("got: %#02v - %d,\n\t   want: %#02v - %d",
			got, len(got), b, len(b),
		)
	}
}

func TestPackListChunk(t *testing.T) {
	b, err := ioutil.ReadFile("listchunk.golden")
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(b)
	var list ListChunk

	if err := list.Unpack(r); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := list.Pack(buf); err != nil {
		t.Fatal(err)
	}
	if got := buf.Bytes(); !bytes.Equal(got, b) {
		t.Errorf("\n got: %#02v - %d,\n\nwant: %#02v - %d",
			got, len(got), b, len(b),
		)
	}
}

// func TestUnpackRIFF(t *testing.T) {
// 	f, err := os.Open("smp.wav")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	var riff RIFFHdr
// 	if err := riff.Unpack(f); err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Errorf("%#+v", riff)
// }

// func TestUnpackFmt(t *testing.T) {
// 	f, err := os.Open("smp.wav")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	skip := int64(unsafe.Sizeof(RIFFHdr{}))
// 	if skip != 12 {
// 		t.Errorf("got: %d, want: %d", skip, 12)
// 	}
// 	if _, err := f.Seek(skip, io.SeekStart); err != nil {
// 		t.Fatal(err)
// 	}
// 	var fmtCk FmtChunk
// 	if err := fmtCk.Unpack(f); err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Errorf("%#+v", fmtCk)
// }

// func TestPackRIFF(t *testing.T) {
// 	riff := RIFFHdr{
// 		ChunkID:   [4]byte{'R', 'I', 'F', 'F'},
// 		ChunkSize: 0xabcd,
// 		Fmt:       [4]byte{'W', 'A', 'V', 'E'},
// 	}
// 	w := &bytes.Buffer{}
// 	if err := riff.Pack(w); err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Errorf("%#+v", w.Bytes())
// }

// func TestUnpackData(t *testing.T) {
// 	f, err := os.Open("smp.wav")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	skip := int64(unsafe.Sizeof(RIFFHdr{}))
// 	if skip != 12 {
// 		t.Errorf("got: %d, want: %d", skip, 12)
// 	}
// 	skip += int64(unsafe.Sizeof(FmtChunk{}))
// 	if skip != 36 {
// 		t.Errorf("got: %d, want: %d", skip, 12)
// 	}
// 	if _, err := f.Seek(skip, io.SeekStart); err != nil {
// 		t.Fatal(err)
// 	}
// 	var dck DataChunk
// 	if err := dck.Unpack(f); err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Errorf("%#+v", dck)
// }

func TestUnpackList(t *testing.T) {
	// f, err := os.Open("smp.wav")
	f, err := os.Open("/tmp/sil/001-qq2.wav")
	if err != nil {
		t.Fatal(err)
	}
	skip := int64(unsafe.Sizeof(RIFFHdr{}))
	if skip != 12 {
		t.Errorf("got: %d, want: %d", skip, 12)
	}
	skip += int64(unsafe.Sizeof(FmtChunk{}))
	if skip != 36 {
		t.Errorf("got: %d, want: %d", skip, 12)
	}
	if _, err := f.Seek(skip, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	var dck DataChunk
	if err := dck.Unpack(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(int64(dck.SubChunkSize), io.SeekCurrent); err != nil {
		t.Fatal(err)
	}
	var lck ListChunk
	if err := lck.Unpack(f); err != nil {
		t.Fatal(err)
	}
	for _, c := range lck.SubChunks {
		t.Logf("%s:\t%s", c.ID, c.Text)
	}
	var ick InfoChunk
	if err := ick.Unpack(f); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s:\t%s", ick.ID, ick.Text)
	if err := ick.Unpack(f); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s:\t%s", ick.ID, ick.Text)
	if err := ick.Unpack(f); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s:\t%s", ick.ID, ick.Text)
	if err := ick.Unpack(f); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s:\t%s", ick.ID, ick.Text)
	if err := ick.Unpack(f); err != nil {
		t.Fatal(err)
	}
	t.Errorf("%s", ick.Text)
}

func TestAt(t *testing.T) {
	file := "/tmp/foo.txt"
	d := []byte("foobar\nlinux\nopenbsd\n")
	if err := ioutil.WriteFile(file, d, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	pos, err := getPos(f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("pos: %d\n", pos)
	defer f.Close()
	var buf bytes.Buffer
	if _, err := buf.Write(d); err != nil {
		t.Fatal(err)
	}
	sw := ioutil2.NewSectionWriter(&writerAt{f}, 7, 5)
	if _, err := sw.Write([]byte("milix")); err != nil {
		t.Fatal(err)
	}
	pos, err = gotoEnd(f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("pos: %d\n", pos)
	if _, err := f.WriteString("trailer\n"); err != nil {
		t.Fatal(err)
	}
}
