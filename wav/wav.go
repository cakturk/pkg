package wav

import (
	"encoding/binary"
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

	_, _ = io.ReadFull(er, f.ChunkID[:])
	_, _ = io.ReadFull(er, p)
	f.ChunkSize = binary.LittleEndian.Uint32(p)

	_, _ = io.ReadFull(er, f.Fmt[:])

	return er.err
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

func (f *FmtChunk) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	_, _ = io.ReadFull(er, f.SubChunkID[:])
	_, _ = io.ReadFull(er, p)
	f.SubChunkSize = binary.LittleEndian.Uint32(p)

	_, _ = io.ReadFull(er, p[:2]) // AudioFormat
	f.AudioFormat = binary.LittleEndian.Uint16(p[:2])

	_, _ = io.ReadFull(er, p[:2]) // NumChans
	f.NumChans = binary.LittleEndian.Uint16(p[:2])

	_, _ = io.ReadFull(er, p) // SampleRate
	f.SampleRate = binary.LittleEndian.Uint32(p)

	_, _ = io.ReadFull(er, p) // ByteRate
	f.ByteRate = binary.LittleEndian.Uint32(p)

	_, _ = io.ReadFull(er, p[:2]) // BlockAlign
	f.BlockAlign = binary.LittleEndian.Uint16(p[:2])

	_, _ = io.ReadFull(er, p[:2]) // BitsPerSample
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

	_, _ = io.ReadFull(er, d.SubChunkID[:])
	_, _ = io.ReadFull(er, p)
	d.SubChunkSize = binary.LittleEndian.Uint32(p)

	return er.err
}

type ListChunk struct {
	SubChunkID   [4]byte // LIST
	SubChunkSize uint32
	TypeID       [4]byte // INFO
}

func (l *ListChunk) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	_, _ = io.ReadFull(er, l.SubChunkID[:])
	_, _ = io.ReadFull(er, p)
	l.SubChunkSize = binary.LittleEndian.Uint32(p)
	_, _ = io.ReadFull(er, l.TypeID[:])
	return er.err
}

type InfoChunk struct {
	ID   [4]byte
	Size uint32
	Text []byte
}

func (i *InfoChunk) Unpack(r io.Reader) error {
	er := &errReader{r: r}
	p := make([]byte, 4)

	_, _ = io.ReadFull(er, i.ID[:])
	_, _ = io.ReadFull(er, p)
	i.Size = binary.LittleEndian.Uint32(p)

	p = make([]byte, i.Size)
	_, _ = io.ReadFull(er, p)

	// Throw away terminating NUL.
	if i.Size > 0 && p[i.Size-1] == '\x00' {
		i.Size--
	}
	i.Text = p[:i.Size]

	return er.err
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
