package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

type fpga struct {
	rw io.ReadWriteCloser
}

func (f *fpga) startSample() error {
	return sendCmd(f.rw, "start sample")
}

func (f *fpga) debug(n int) error {
	return sendCmd(f.rw, "debug "+strconv.Itoa(n))
}

func (f *fpga) readHdr(n int) (uint32, error) {
	return sendAndReadNumBytes(f.rw, "read"+strconv.Itoa(n))
}

func main() {
	conn, err := net.Dial("tcp", "192.168.1.124:7")
	if err != nil {
		log.Fatal(err)
	}
	// if err := conn.SetDeadline(200 * time.Millisecond); err != nil {
	// 	log.Fatalf("failed set deadline: %v", err)
	// }

	must(sendCmd(conn, "start sample"))
	must(sendCmd(conn, "debug 2"))

	must(sendCmd(conn, "enable 0"))
	must(sendCmd(conn, "enable 1"))
	must(sendCmd(conn, "enable 2"))
	must(sendCmd(conn, "enable 3"))
	must(sendCmd(conn, "enable 4"))
	must(sendCmd(conn, "enable 5"))
	must(sendCmd(conn, "enable 6"))
	must(sendCmd(conn, "enable 7"))

	time.Sleep(1 * time.Second)
	// readHdr(conn)
	// return
	i := 0
	for i < 300 {
		readN(conn, 1, 0)
		readN(conn, 1, 1)
		readN(conn, 1, 2)
		readN(conn, 1, 3)
		readN(conn, 1, 4)
		readN(conn, 1, 5)
		readN(conn, 1, 6)
		readN(conn, 1, 7)
		i++
	}
	must(sendCmd(conn, "disable 0"))
	must(sendCmd(conn, "disable 1"))
	must(sendCmd(conn, "disable 2"))
	must(sendCmd(conn, "disable 3"))
	must(sendCmd(conn, "disable 4"))
	must(sendCmd(conn, "disable 5"))
	must(sendCmd(conn, "disable 6"))
	must(sendCmd(conn, "disable 7"))
	return

	// time.Sleep(3 * time.Second)
	must(sendCmd(conn, "read 0"))

	p, err := readData(conn)
	if err != nil {
		return
	}
	q, err := readData(conn)
	if err != nil {
		return
	}
	p = append(p, q...)
	r, err := readData(conn)
	if err != nil {
		return
	}
	p = append(p, r...)
	time.Sleep(500 * time.Millisecond)
	r, err = readData(conn)
	if err != nil {
		return
	}
	p = append(p, r...)
	unpack(0, p)
	fmt.Printf("read %d bytes\n", len(p))
}

func enable(conn io.ReadWriter, n int) error {
	ns := strconv.Itoa(n)
	return sendCmd(conn, "enable "+ns)
}

func readN(r io.ReadWriter, count, nc int) {
	total := 0
	p := make([]byte, 32768*100)
	ns := strconv.Itoa(nc)
	i := 0
	for i < count {
		// must(sendCmd(r, "read 1"))
		cnt, err := sendAndReadNumBytes(r, "read "+ns)
		// fmt.Fprintf(os.Stderr, "total bytes: %d\n", cnt)
		if cnt == 0 {
			fmt.Fprintf(os.Stderr, "failed to read bytes, sleeping...\n")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("hdr bytes: %d\n", cnt)
		n, err := io.ReadFull(r, p[:cnt])
		// n, err := r.Read(p[:cnt])
		if err != nil {
			log.Fatalf("readN: %d, err: %v", n, err)
		}
		// fmt.Printf("read %d bytes\n", n)
		total += n
		unpack(nc, p[:cnt])
		i++
	}
	// fmt.Printf("total bytes: %d\n", total)
}

// func ReadFull(r Reader, buf []byte) (n int, err error) {
// 	return ReadAtLeast(r, buf, len(buf))
// }

// func ReadAtLeast(r Reader, buf []byte, min int) (n int, err error) {
// 	if len(buf) < min {
// 		return 0, ErrShortBuffer
// 	}
// 	for n < min && err == nil {
// 		var nn int
// 		nn, err = r.Read(buf[n:])
// 		n += nn
// 	}
// 	if n >= min {
// 		err = nil
// 	} else if n > 0 && err == EOF {
// 		err = ErrUnexpectedEOF
// 	}
// 	return
// }

func sendAndReadNumBytes(rw io.ReadWriter, cmd string) (uint32, error) {
	b := []byte(cmd)
	b = append(b, []byte{0x0d, 0x00}...)
	_, err := rw.Write(b)
	if err != nil {
		return 0, err
	}

	p := make([]byte, 8)
	n, err := io.ReadFull(rw, p)
	if err != nil {
		return 0, err
	}
	if string(p[:4]) != "_OK_" {
		fmt.Printf("%#v, %#v\n", p[:4], "_OK_")
		return 0, fmt.Errorf("cmd: %d bytes, %s: %s", n, "", p)
	}
	// fmt.Printf("raw cnt: %#v\n", p[4:])
	u := binary.LittleEndian.Uint32(p[4:])
	return u, nil
}

func sendRawCmd(rw io.ReadWriter, cmd string) error {
	b := []byte(cmd)
	b = append(b, []byte{0x0d, 0x00}...)
	_, err := rw.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func readHdr(r io.ReadWriter) {
	must(sendRawCmd(r, "read 1"))
	p := make([]byte, 8)

	_, err := io.ReadFull(r, p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("hdr: %#v\n", p)
}

func validHdr(r io.Reader) error {
	ack := make([]byte, 4)
	n, err := io.ReadFull(r, ack)
	if err != nil {
		return err
	}
	if string(ack[:4]) != "_OK_" {
		fmt.Printf("%#v, %#v\n", ack[:4], "_OK_")

		return fmt.Errorf("cmd: %d bytes, %s: %s", n, "", ack)
	}
	return nil
}

// var prev = uint16(0)
var prev [8]uint16

func unpack(n int, p []byte) {
	i := 0
	s := len(p) - len(p)%2
	for i < s-1 {
		q := p[i : i+2]
		u := binary.LittleEndian.Uint16(q)
		fmt.Fprintf(os.Stderr, "val: %d, %#v\n", u, q)
		if prev[n] > 0 && u-prev[n] > 1 {
			fmt.Fprintf(os.Stdout, "prev: %d, val: %d, %#v\n", prev[n], u, q)
		}
		prev[n] = u
		i += 2
	}
}

func readData(r io.Reader) ([]byte, error) {
	p := make([]byte, 8192)
	n, err := r.Read(p)
	fmt.Printf("read len: %d\n", n)

	if err != nil {
		return nil, err
	}
	return p[:n], nil
}

func sendCmd(rw io.ReadWriter, cmd string) error {
	b := []byte(cmd)
	b = append(b, []byte{0x0d, 0x00}...)
	_, err := rw.Write(b)
	if err != nil {
		return err
	}

	ack := make([]byte, 4)
	n, err := io.ReadFull(rw, ack)
	if err != nil {
		return err
	}
	if string(ack[:4]) != "_OK_" {
		fmt.Printf("%#v, %#v\n", ack[:4], "_OK_")

		return fmt.Errorf("cmd: %d bytes, %s: %s", n, cmd, ack)
	}
	return nil
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
