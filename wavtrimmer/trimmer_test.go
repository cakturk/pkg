package wavtrimmer

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-audio/wav"
)

func timeDiff(start, end, d time.Duration) time.Duration {
	if start == -1 {
		start = 0
	}
	if end == -1 {
		end = d
	}
	return end - start
}

// fixture wav file is taken from:
// http://www.music.helsinki.fi/tmt/opetus/uusmedia/esim/index-e.html
func TestTrim(t *testing.T) {
	cuttingTests := []struct {
		in    string
		start time.Duration
		end   time.Duration
	}{
		{"testdata/a2002011001-e02-16kHz.wav", 1 * time.Second, 3500 * time.Millisecond},
		{"testdata/a2002011001-e02-16kHz.wav", -1, 12 * time.Second},
		{"testdata/a2002011001-e02-16kHz.wav", 23 * time.Second, -1},
		{"testdata/a2002011001-e02-16kHz.wav", -1, -1},
		{"testdata/a2002011001-e02-16kHz.wav", 16 * time.Second, 42752 * time.Millisecond},
	}
	for _, tt := range cuttingTests {
		in, err := os.Open(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		defer in.Close()
		out, err := os.Create("testdata/cropped.wav")
		if err != nil {
			t.Fatal(err)
		}
		defer out.Close()
		err = Trim(in, tt.start, tt.end, out)
		if err != nil {
			t.Fatalf("Crop(...) failed: %v", err)
		}
		out.Seek(0, io.SeekStart)
		od := wav.NewDecoder(out)
		if !od.IsValidFile() {
			t.Fatal("invalid .wav file")
		}
		odur, err := od.Duration()
		if err != nil {
			t.Fatal("cannot find duration")
		}
		in.Seek(0, io.SeekStart)
		id := wav.NewDecoder(in)
		idur, err := id.Duration()
		if err != nil {
			t.Fatal("cannot find duration")
		}
		// Comparing only seconds part is just precise enough to
		// measure the duration of the part which we cut out
		want := timeDiff(tt.start, tt.end, idur) / time.Second
		got := odur / time.Second
		if want != got {
			t.Errorf("got: %q, want: %q", odur, timeDiff(tt.start, tt.end, idur))
		}
	}
	os.Remove("testdata/cropped.wav")
}

func TestTrim2(t *testing.T) {
	in, err := os.Open("/tmp/sil/mother_talk.wav")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create("/tmp/sil/cropped.wav")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := Trim2(in, 220*time.Millisecond, 500*time.Millisecond, out); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkTrim(b *testing.B) {
	in, err := os.Open("/tmp/sil/001-qq2.wav")
	if err != nil {
		b.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create("testdata/cropped-old.wav")
	if err != nil {
		b.Fatal(err)
	}
	defer out.Close()
	err = Trim(in, 3*time.Hour, 4*time.Hour, out)
	if err != nil {
		b.Fatalf("Crop(...) failed: %v", err)
	}
}

func BenchmarkTrim2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		in, err := os.Open("/tmp/sil/001-qq2.wav")
		if err != nil {
			b.Fatal(err)
		}
		defer in.Close()
		out, err := os.Create("/tmp/sil/cropped.wav")
		if err != nil {
			b.Fatal(err)
		}
		defer out.Close()
		if err := Trim2(in, 3*time.Hour, 4*time.Hour, out); err != nil {
			b.Fatal(err)
		}

	}
}
