package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "192.168.1.124:7")
	if err != nil {
		log.Fatal(err)
	}

	must(sendCmd(conn, "start sample"))
	must(sendCmd(conn, "debug 2"))
	must(sendCmd(conn, "enable 0"))
	time.Sleep(1 * time.Second)
	readN(conn, 100)
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
	unpack(p)
	fmt.Printf("read %d bytes\n", len(p))
}

func readN(r io.ReadWriter, count int) {
	total := 0
	p := make([]byte, 8192)
	i := 0
	for i < count {
		must(sendCmd(r, "read 0"))
		n, err := io.ReadFull(r, p)
		if err != nil {
			log.Fatalf("readN: %d, err: %v", n, err)
		}
		total += n
		unpack(p)
		i++
	}
	fmt.Printf("total bytes: %d\n", total)
}

func unpack(p []byte) {
	i := 0
	s := len(p) - len(p)%2
	for i < s-1 {
		q := p[i : i+2]
		u := binary.LittleEndian.Uint16(q)
		fmt.Printf("val: %d, %#v\n", u, q)
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

	ack := make([]byte, 7)
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
