package xbufio

import (
	"bufio"
	"io"
)

const (
	defaultBufSize = 4096
)

// WriteSeekCloser implements buffering for an io.WriteSeekCloser object.
type WriteSeekCloser struct {
	w io.WriteSeeker
	b *bufio.Writer
}

// NewWriteSeekCloserSize returns a new Writer whose buffer has at least the specified
// size. If the argument io.Writer is already a Writer with large enough
// size, it returns the underlying Writer.
func NewWriteSeekCloserSize(w io.WriteSeeker, size int) *WriteSeekCloser {
	// Is it already a WriterSeeker?
	b, ok := w.(*WriteSeekCloser)
	if ok && b.b.Buffered()+b.b.Available() >= size {
		return b
	}
	if size <= 0 {
		size = defaultBufSize
	}
	return &WriteSeekCloser{
		w: w,
		b: bufio.NewWriterSize(w, defaultBufSize),
	}
}

// NewWriteSeekCloser returns a new Writer whose buffer has the default size.
func NewWriteSeekCloser(w io.WriteSeeker) *WriteSeekCloser {
	return NewWriteSeekCloserSize(w, defaultBufSize)
}

// Write writes the contents of p into the buffer.
func (w *WriteSeekCloser) Write(p []byte) (n int, err error) {
	return w.b.Write(p)
}

// Seek sets the offset for the next Read or Write on file to offset.
func (w *WriteSeekCloser) Seek(offset int64, whence int) (int64, error) {
	err := w.b.Flush()
	if err != nil {
		return 0, err
	}
	return w.w.Seek(offset, whence)
}

// Close closes the File, rendering it unusable for I/O.
func (w *WriteSeekCloser) Close() error {
	err := w.b.Flush()
	if err != nil {
		return err
	}
	c, ok := w.w.(io.Closer)
	if !ok {
		return nil
	}
	return c.Close()
}
