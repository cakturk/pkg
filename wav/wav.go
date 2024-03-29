package wav

// http://soundfile.sapp.org/doc/WaveFormat
// https://ccrma.stanford.edu/courses/422-winter-2014/projects/WaveFormat/
import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/siddontang/go/ioutil2"
)

var (
	RIFF = [4]byte{'R', 'I', 'F', 'F'}
	WAVE = [4]byte{'W', 'A', 'V', 'E'}
	LIST = [4]byte{'L', 'I', 'S', 'T'}
	INFO = [4]byte{'I', 'N', 'F', 'O'}
	FMT  = [4]byte{'f', 'm', 't', ' '}
	DATA = [4]byte{'d', 'a', 't', 'a'}

	// Copied from go-audio
	// List of wav chunk names
	// See http://bwfmetaedit.sourceforge.net/listinfo.html
	IART    = [4]byte{'I', 'A', 'R', 'T'} // artist
	ICMT    = [4]byte{'I', 'C', 'M', 'T'} // comments
	ICOP    = [4]byte{'I', 'C', 'O', 'P'} // copyright
	ICRD    = [4]byte{'I', 'C', 'R', 'D'} // creationDate
	IENG    = [4]byte{'I', 'E', 'N', 'G'} // engineer
	ITCH    = [4]byte{'I', 'T', 'C', 'H'} // technician
	IGNR    = [4]byte{'I', 'G', 'N', 'R'} // genre
	IKEY    = [4]byte{'I', 'K', 'E', 'Y'} // keywords
	IMED    = [4]byte{'I', 'M', 'E', 'D'} // medium
	INAM    = [4]byte{'I', 'N', 'A', 'M'} // title
	IPRD    = [4]byte{'I', 'P', 'R', 'D'} // product
	ISBJ    = [4]byte{'I', 'S', 'B', 'J'} // subject
	ISFT    = [4]byte{'I', 'S', 'F', 'T'} // software
	ISRC    = [4]byte{'I', 'S', 'R', 'C'} // source
	IARL    = [4]byte{'I', 'A', 'R', 'L'} // location
	ITRK    = [4]byte{'I', 'T', 'R', 'K'} // trackNbr
	ITRKBug = [4]byte{'i', 't', 'r', 'k'} // trackNbr
)

type WavFile struct {
	Hdr  RIFFHdr
	Fmt  FmtChunk
	Data DataChunk
	List *ListChunk

	rifWr io.Writer
	fmtWr io.Writer
	datWr io.Writer
}

const (
	RIFFHdrSize      = 12
	FmtChunkSize     = 24
	DataChunkHdrSize = 8 // not including PCM samples
)

func (wf *WavFile) writeHdr(w io.Writer) error {
	if wf.rifWr != nil {
		w = wf.rifWr
	}
	return wf.Hdr.Pack(w)
}

func (wf *WavFile) writeFmt(w io.Writer) error {
	if wf.fmtWr != nil {
		w = wf.fmtWr
	} else if w == nil {
		return nil
	}
	return wf.Fmt.Pack(w)
}

func (wf *WavFile) writeDataHdr(w io.Writer) error {
	if wf.datWr != nil {
		w = wf.datWr
	} else if w == nil {
		return nil
	}
	return wf.Data.Pack(w)
}

func (wf *WavFile) Duration() time.Duration {
	return time.Duration(float64(wf.Data.SubChunkSize) / float64(wf.Fmt.ByteRate) * float64(time.Second))
}

func (wf *WavFile) Encode(w io.WriteSeeker) (int64, error) {
	hdrWr := sectionWriter(w, 0, RIFFHdrSize)

	if err := wf.writeFmt(nil); err != nil {
		return 0, err
	}

	if err := wf.writeDataHdr(w); err != nil {
		return 0, err
	}

	off := wf.Data.size() + RIFFHdrSize + FmtChunkSize
	if wf.List != nil {
		// forward to List chunk
		if _, err := w.Seek(off, io.SeekStart); err != nil {
			return 0, err
		}
		if err := wf.List.Pack(w); err != nil {
			return 0, err
		}
		off += wf.List.size()
	}

	wf.Hdr.ChunkSize = uint32(off) - 8
	if err := wf.writeHdr(hdrWr); err != nil {
		return 0, err
	}
	return int64(wf.Hdr.ChunkSize) + 8, nil
}

func Create(w io.WriteSeeker, sampleRate, nchans, nbits int) (*WavFile, error) {
	blockAlign := uint16(nchans * nbits / 8)
	wf := &WavFile{
		Hdr: RIFFHdr{
			ChunkID:   RIFF,
			ChunkSize: RIFFHdrSize + FmtChunkSize + DataChunkHdrSize - 8,
			Fmt:       WAVE,
		},
		Fmt: FmtChunk{
			SubChunkID:    FMT,
			SubChunkSize:  0x10,
			AudioFormat:   1,
			NumChans:      uint16(nchans),
			SampleRate:    uint32(sampleRate),
			ByteRate:      uint32(sampleRate) * uint32(blockAlign),
			BlockAlign:    blockAlign,
			BitsPerSample: uint16(nbits),
		},
		Data: DataChunk{
			SubChunkID:   DATA,
			SubChunkSize: 0,
		},
		rifWr: sectionWriter(w, 0, RIFFHdrSize),
		fmtWr: sectionWriter(w, RIFFHdrSize, FmtChunkSize),
		datWr: sectionWriter(w, RIFFHdrSize+FmtChunkSize, DataChunkHdrSize),
	}

	// forward to PCM offset: 44
	var off int64 = RIFFHdrSize + FmtChunkSize + DataChunkHdrSize
	if _, err := w.Seek(off, io.SeekStart); err != nil {
		return nil, err
	}
	wf.Data.pcmWr = &pcmWriter{
		Writer:    w,
		chunkSize: &wf.Data.SubChunkSize,
	}
	return wf, nil
}

func Decode(r io.ReadSeeker) (*WavFile, error) {
	w := &WavFile{}
	if err := w.Hdr.Unpack(r); err != nil {
		return nil, err
	}
	if err := w.Fmt.Unpack(r); err != nil {
		return nil, err
	}
	if err := w.Data.Unpack(r); err != nil {
		return nil, err
	}
	var (
		curOff int64
		endOff int64
		err    error
	)
	if curOff, err = r.Seek(0, io.SeekCurrent); err != nil {
		return nil, err
	}

	w.Data.pcmRd = sectionReader(r, curOff, int64(w.Data.SubChunkSize))

	if endOff, err = r.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}
	if curOff+int64(w.Data.SubChunkSize) >= endOff {
		return w, nil // no list chunks available
	}
	curOff = curOff + int64(w.Data.SubChunkSize)
	if _, err := r.Seek(curOff, io.SeekStart); err != nil {
		return nil, err
	}
	lck := &ListChunk{}
	if err := lck.Unpack(r); err != nil {
		return nil, err
	}
	w.List = lck

	return w, nil
}

type RIFFHdr struct {
	ChunkID   [4]byte // RIFF
	ChunkSize uint32
	Fmt       [4]byte // WAVE
}

func (f *RIFFHdr) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	er.ReadFull(f.ChunkID[:])
	if f.ChunkID != RIFF {
		return errors.New("wav: malformed RIFF header")
	}
	er.ReadFull(p)
	f.ChunkSize = binary.LittleEndian.Uint32(p)
	er.ReadFull(f.Fmt[:])
	if f.Fmt != WAVE {
		return errors.New("wav: malformed WAVE header")
	}
	return er.err
}

func (f *RIFFHdr) Pack(w io.Writer) error {
	ew := &errWriter{w: w}
	p := make([]byte, 4)

	ew.write(f.ChunkID[:])
	binary.LittleEndian.PutUint32(p, f.ChunkSize)
	ew.write(p)
	ew.write(f.Fmt[:])
	return ew.err
}

type FmtChunk struct {
	SubChunkID    [4]byte // "fmt "
	SubChunkSize  uint32
	AudioFormat   uint16
	NumChans      uint16
	SampleRate    uint32
	ByteRate      uint32 // avg bytes per sec
	BlockAlign    uint16
	BitsPerSample uint16
}

func (f *FmtChunk) Pack(w io.Writer) error {
	ew := &errWriter{w: w}
	p := make([]byte, 4)

	ew.write(f.SubChunkID[:])
	if string(f.SubChunkID[:]) != "fmt " {
		return errors.New("wav: malformed fmt chunk header")
	}
	binary.LittleEndian.PutUint32(p, f.SubChunkSize)
	ew.write(p)
	binary.LittleEndian.PutUint16(p[:2], f.AudioFormat)
	ew.write(p[:2])
	binary.LittleEndian.PutUint16(p[:2], f.NumChans)
	ew.write(p[:2])
	binary.LittleEndian.PutUint32(p, f.SampleRate)
	ew.write(p)
	binary.LittleEndian.PutUint32(p, f.ByteRate)
	ew.write(p)
	binary.LittleEndian.PutUint16(p[:2], f.BlockAlign)
	ew.write(p[:2])
	binary.LittleEndian.PutUint16(p[:2], f.BitsPerSample)
	ew.write(p[:2])

	return ew.err
}

func (f *FmtChunk) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	er.ReadFull(f.SubChunkID[:])
	er.ReadFull(p)
	f.SubChunkSize = binary.LittleEndian.Uint32(p)

	er.ReadFull(p[:2]) // AudioFormat
	f.AudioFormat = binary.LittleEndian.Uint16(p[:2])

	er.ReadFull(p[:2]) // NumChans
	f.NumChans = binary.LittleEndian.Uint16(p[:2])

	er.ReadFull(p) // SampleRate
	f.SampleRate = binary.LittleEndian.Uint32(p)

	er.ReadFull(p) // ByteRate
	f.ByteRate = binary.LittleEndian.Uint32(p)

	er.ReadFull(p[:2]) // BlockAlign
	f.BlockAlign = binary.LittleEndian.Uint16(p[:2])

	er.ReadFull(p[:2]) // BitsPerSample
	f.BitsPerSample = binary.LittleEndian.Uint16(p[:2])

	return er.err
}

type DataChunk struct {
	SubChunkID   [4]byte // data
	SubChunkSize uint32

	pcmWr io.Writer
	pcmRd io.ReadSeeker
}

func (d *DataChunk) size() int64 {
	return int64(d.SubChunkSize + 8)
}

func (d *DataChunk) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	er.ReadFull(d.SubChunkID[:])
	er.ReadFull(p)
	d.SubChunkSize = binary.LittleEndian.Uint32(p)

	return er.err
}

func (d *DataChunk) Pack(w io.Writer) error {
	ew := &errWriter{w: w}
	p := make([]byte, 4)
	ew.write(d.SubChunkID[:])
	binary.LittleEndian.PutUint32(p, d.SubChunkSize)
	ew.write(p)
	return ew.err
}

func (d *DataChunk) PCMReader() io.ReadSeeker {
	return d.pcmRd
}

func (d *DataChunk) PCMWriter() io.Writer {
	return d.pcmWr
}

type ListChunk struct {
	SubChunkID   [4]byte // LIST
	SubChunkSize uint32  // payload size after this point
	TypeID       [4]byte // INFO

	SubChunks []InfoChunk
}

func (l *ListChunk) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	er.ReadFull(l.SubChunkID[:])
	er.ReadFull(p)
	l.SubChunkSize = binary.LittleEndian.Uint32(p)
	er.ReadFull(l.TypeID[:])

	if string(l.TypeID[:]) != "INFO" {
		return fmt.Errorf("wav: unsupported subchunk id: %v", string(l.SubChunkID[:]))
	}

	readBytes := uint32(0)
	totalBytes := l.SubChunkSize - uint32(len(l.TypeID))
	for readBytes < totalBytes {
		var ic InfoChunk
		if err := ic.Unpack(r); err != nil {
			return fmt.Errorf("list.Unpack: %w", err)
		}
		l.SubChunks = append(l.SubChunks, ic)
		readBytes += uint32(ic.RawSize())
	}
	return er.err
}

func (l *ListChunk) Pack(w io.Writer) error {
	ew := &errWriter{w: w}
	p := make([]byte, 4)
	ew.write(l.SubChunkID[:])

	l.SubChunkSize = uint32(l.ChunkSize())
	binary.LittleEndian.PutUint32(p, l.SubChunkSize)
	ew.write(p)

	ew.write(l.TypeID[:])

	for _, sc := range l.SubChunks {
		if err := sc.Pack(w); err != nil {
			return err
		}
	}
	return ew.err
}

func (l *ListChunk) InfoChunk(name [4]byte) string {
	for _, ic := range l.SubChunks {
		if ic.ID == name {
			return string(ic.Text)
		}
	}
	return ""
}

func (l *ListChunk) ChunkSize() int {
	total := 0
	for _, sc := range l.SubChunks {
		total += sc.RawSize()
	}
	return total + 4 // 4 bytes for TypeID: INFO
}

func (l *ListChunk) size() int64 {
	if l == nil {
		return 0
	}
	return int64(l.SubChunkSize + 8)
}

func (l *ListChunk) RawSize() int {
	return l.ChunkSize() + 8 // ID + Size hdr fields
}

type InfoChunk struct {
	ID   [4]byte
	Size uint32
	Text []byte
}

func (i *InfoChunk) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	er.ReadFull(i.ID[:])
	er.ReadFull(p)
	i.Size = binary.LittleEndian.Uint32(p)

	p = make([]byte, i.Size)
	er.ReadFull(p)

	// Throw away terminating NUL.
	if i.Size > 0 && p[i.Size-1] == '\x00' {
		i.Size--
	}
	i.Text = p[:i.Size]

	return er.err
}

func (i *InfoChunk) Pack(w io.Writer) error {
	ew := &errWriter{w: w}
	p := make([]byte, 4)
	ew.write(i.ID[:])
	binary.LittleEndian.PutUint32(p, uint32(len(i.Text)+1))
	ew.write(p)

	i.Text = append(i.Text, '\x00') // NUL terminate
	ew.write(i.Text)

	return ew.err
}

func (i *InfoChunk) RawSize() int {
	// fields + txt size + NUL terminator
	return 8 + len(i.Text) + 1
}

type errReader struct {
	r   io.Reader
	err error
}

func (er *errReader) Read(p []byte) (n int, err error) {
	if er.err != nil {
		if er.err == io.EOF {
			return 0, er.err
		}
		return 0, nil
	}
	n, er.err = er.r.Read(p)
	return n, nil
}

func (er *errReader) ReadFull(buf []byte) {
	_, _ = io.ReadFull(er, buf)
}

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) write(buf []byte) {
	if ew.err != nil {
		return
	}
	_, ew.err = ew.w.Write(buf)
}

type writerAt struct {
	ws io.WriteSeeker
}

func (w *writerAt) WriteAt(p []byte, off int64) (n int, err error) {
	if _, err := w.ws.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}
	n, err = w.ws.Write(p)
	return n, err
}

func sectionWriter(ws io.WriteSeeker, off, size int64) io.Writer {
	var w io.Writer
	switch v := ws.(type) {
	case io.WriterAt:
		w = ioutil2.NewSectionWriter(v, off, size)
	default:
		w = ioutil2.NewSectionWriter(&writerAt{ws}, off, size)
	}
	return w
}

func sectionReader(rs io.ReadSeeker, off, size int64) io.ReadSeeker {
	switch v := rs.(type) {
	case io.ReaderAt:
		return io.NewSectionReader(v, off, size)
	default:
		return nil
	}
}

type pcmWriter struct {
	io.Writer

	chunkSize *uint32
}

func (p *pcmWriter) Write(b []byte) (n int, err error) {
	n, err = p.Writer.Write(b)
	*p.chunkSize += uint32(n)
	return
}
