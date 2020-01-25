package wavtrimmer

import (
	"errors"
	"fmt"
	"io"
	"time"

	cwav "github.com/cakturk/pkg/wav"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// Trim function cuts out the part between the time from `start` to the `stop`
// out of a wav file
func Trim(r io.ReadSeeker, start time.Duration, end time.Duration, w io.WriteSeeker) error {
	dec := wav.NewDecoder(r)
	if !dec.IsValidFile() {
		return errors.New("cutter: invalid file format")
	}
	dec.ReadInfo()
	t, err := dec.Duration()
	if err != nil {
		return err
	}
	if start == -1 {
		start = 0
	}
	if end == -1 {
		end = t
	}
	if t < start || t < end {
		return fmt.Errorf("cutter: start: %s or end: %s not between 0 - %s", start, end, t)
	}
	if start >= end {
		return fmt.Errorf("cutter: start: %s earlier than end: %s", start, end)
	}
	enc := wav.NewEncoder(
		w,
		int(dec.SampleRate),
		int(dec.BitDepth),
		int(dec.NumChans),
		int(dec.WavAudioFormat),
	)
	buf := &audio.IntBuffer{
		Data: make([]int, 32*1024),
		Format: &audio.Format{
			NumChannels: int(dec.NumChans),
			SampleRate:  int(dec.SampleRate),
		},
		SourceBitDepth: int(dec.BitDepth),
	}
	return cutOut(start, end, buf, dec, enc)
}

func howMany(d time.Duration, dec *wav.Decoder, buf *audio.IntBuffer) (n int, remainder int) {
	bytesInSample := dec.BitDepth / 8
	bytesToSkip := uint64(dec.AvgBytesPerSec) * uint64(d/time.Second)
	nrSamples := int(bytesToSkip / uint64(bytesInSample))
	return nrSamples / len(buf.Data), nrSamples % len(buf.Data)
}

func discard(to time.Duration, buf *audio.IntBuffer, dec *wav.Decoder) error {
	var err error
	n, remainder := howMany(to, dec, buf)
	for n > 0 {
		_, err = dec.PCMBuffer(buf)
		if err != nil {
			return err
		}
		n--
	}
	buf.Data = buf.Data[:remainder]
	_, err = dec.PCMBuffer(buf)
	return err
}

func cutOut(
	start, end time.Duration,
	buf *audio.IntBuffer,
	dec *wav.Decoder,
	enc *wav.Encoder,
) error {
	var err error
	err = discard(start, buf, dec)
	if err != nil {
		return err
	}
	// grow buf.Data to its capacity by slicing it again
	buf.Data = buf.Data[:cap(buf.Data)]
	n, remainder := howMany(end-start, dec, buf)
	for n > 0 {
		_, err = dec.PCMBuffer(buf)
		if err != nil {
			return err
		}
		err = enc.Write(buf)
		if err != nil {
			return err
		}
		n--
	}
	buf.Data = buf.Data[:remainder]
	_, err = dec.PCMBuffer(buf)
	if err != nil {
		return err
	}
	err = enc.Write(buf)
	if err != nil {
		return err
	}
	return enc.Close()
}

func durationReader(src *cwav.WavFile, start, end time.Duration) (io.Reader, error) {
	off := int64(float64(src.Fmt.ByteRate) * float64(start) / float64(time.Second))
	if off%2 != 0 {
		off++
	}
	r := src.Data.PCMReader()
	if _, err := r.Seek(off, io.SeekCurrent); err != nil {
		return nil, err
	}
	count := int64(float64(src.Fmt.ByteRate) * float64(end-start) / float64(time.Second))
	if count%2 != 0 {
		count--
	}
	return io.LimitReader(r, count), nil
}

// Trim2 function cuts out the part between the time from `start` to the `stop`
// out of a wav file
func Trim2(r io.ReadSeeker, start time.Duration, end time.Duration, w io.WriteSeeker) error {
	wavSrc, err := cwav.Decode(r)
	if err != nil {
		return err
	}
	src := wavSrc.Data.PCMReader()
	if src == nil {
		return errors.New("trim: nil PCM reader")
	}

	dur := wavSrc.Duration()
	if start == -1 {
		start = 0
	}
	if end == -1 {
		end = dur
	}
	if dur < start || dur < end {
		return fmt.Errorf("trim: start: %s or end: %s not between 0 - %s", start, end, dur)
	}
	if start >= end {
		return fmt.Errorf("trim: start: %s earlier than end: %s", start, end)
	}

	srcDr, err := durationReader(wavSrc, start, end)
	if err != nil {
		return err
	}

	wavDst, err := cwav.Create(w, int(wavSrc.Fmt.SampleRate))
	if err != nil {
		return err
	}
	dst := wavDst.Data.PCMWriter()
	if dst == nil {
		return errors.New("trim: nil PCM writer")
	}

	p := make([]byte, 128*1024)
	if _, err := io.CopyBuffer(dst, srcDr, p); err != nil {
		return err
	}
	if _, err := wavDst.Encode(w); err != nil {
		return err
	}
	return nil
}
