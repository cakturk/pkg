package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

type RIFFHdr struct {
	ChunkID   [4]byte // RIFF
	ChunkSize uint32
	Fmt       [4]byte // WAVE
}

func (f *RIFFHdr) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	er.ReadFull(f.ChunkID[:])
	er.ReadFull(p)
	f.ChunkSize = binary.LittleEndian.Uint32(p)
	er.ReadFull(f.Fmt[:])
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
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

func (f *FmtChunk) Pack(w io.Writer) error {
	ew := &errWriter{w: w}
	p := make([]byte, 4)

	ew.write(f.SubChunkID[:])
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
	SampleData   []byte
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

func (l *ListChunk) ChunkSize() int {
	total := 0
	for _, sc := range l.SubChunks {
		total += sc.RawSize()
	}
	return total + 4 // 4 bytes for TypeID: INFO
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
