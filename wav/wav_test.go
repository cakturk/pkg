package wav

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"unsafe"
)

func TestUnpackRIFF(t *testing.T) {
	f, err := os.Open("smp.wav")
	if err != nil {
		t.Fatal(err)
	}

	var riff RIFFHdr
	if err := riff.Unpack(f); err != nil {
		t.Fatal(err)
	}
	t.Errorf("%#+v", riff)
}

func TestPackRIFF(t *testing.T) {
	riff := RIFFHdr{
		ChunkID:   [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize: 0xabcd,
		Fmt:       [4]byte{'W', 'A', 'V', 'E'},
	}
	w := &bytes.Buffer{}
	if err := riff.Pack(w); err != nil {
		t.Fatal(err)
	}
	t.Errorf("%#+v", w.Bytes())
}

func TestUnpackFmt(t *testing.T) {
	f, err := os.Open("smp.wav")
	if err != nil {
		t.Fatal(err)
	}
	skip := int64(unsafe.Sizeof(RIFFHdr{}))
	if skip != 12 {
		t.Errorf("got: %d, want: %d", skip, 12)
	}
	if _, err := f.Seek(skip, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	var fmtCk FmtChunk
	if err := fmtCk.Unpack(f); err != nil {
		t.Fatal(err)
	}
	t.Errorf("%#+v", fmtCk)
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

func TestUnpackData(t *testing.T) {
	f, err := os.Open("smp.wav")
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
	t.Errorf("%#+v", dck)
}

func TestUnpackList(t *testing.T) {
	f, err := os.Open("smp.wav")
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
	// var ick InfoChunk
	// if err := ick.Unpack(f); err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("%s:\t%s", ick.ID, ick.Text)
	// if err := ick.Unpack(f); err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("%s:\t%s", ick.ID, ick.Text)
	// if err := ick.Unpack(f); err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("%s:\t%s", ick.ID, ick.Text)
	// if err := ick.Unpack(f); err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("%s:\t%s", ick.ID, ick.Text)
	// if err := ick.Unpack(f); err != nil {
	// 	t.Fatal(err)
	// }
	// t.Errorf("%s", ick.Text)
}
