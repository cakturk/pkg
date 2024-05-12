package cygwin

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
)

// !<socket >53622 s 686A723F-83EEA6E6-82B318B1-9BE411BC
func parse(path string) (int, [4]uint32, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, [4]uint32{}, fmt.Errorf("parse: %w", err)
	}
	var (
		port int
		uuid [4]uint32
	)
	_, err = fmt.Sscanf(
		string(b), "!<socket >%d s %x-%x-%x-%x",
		&port, &uuid[0], &uuid[1], &uuid[2], &uuid[3],
	)
	if err != nil {
		return 0, uuid, fmt.Errorf("parse: %w", err)
	}
	fmt.Printf("content: %q: port: %d %x\n", b, port, uuid[0])
	return port, uuid, nil
}

func handshake(w io.ReadWriter, uuid [4]uint32, pid, uid, gid uint32) error {
	var buf [16]byte
	binary.LittleEndian.PutUint32(buf[:4], uuid[0])
	binary.LittleEndian.PutUint32(buf[4:8], uuid[1])
	binary.LittleEndian.PutUint32(buf[8:12], uuid[2])
	binary.LittleEndian.PutUint32(buf[12:], uuid[3])
	_, err := w.Write(buf[:])
	if err != nil {
		return fmt.Errorf("handshake: keyxchg: %w", err)
	}
	_, err = io.ReadFull(w, buf[:])
	if err != nil {
		return fmt.Errorf("handshake: keyxchg read: %w", err)
	}
	binary.BigEndian.PutUint32(buf[:4], pid)
	binary.BigEndian.PutUint32(buf[4:8], uid)
	binary.BigEndian.PutUint32(buf[8:12], gid)
	_, err = w.Write(buf[:12])
	if err != nil {
		return fmt.Errorf("handshake: process credentials: %w", err)
	}
	_, err = io.ReadFull(w, buf[:12])
	if err != nil {
		return fmt.Errorf("handshake: process credentials read: %w", err)
	}
	return nil
}

func dial(path string) (net.Conn, error) {
	port, uuid, err := parse(path)
	if err != nil {
		return nil, fmt.Errorf("cygwin: %w", err)
	}
	conn, err := net.Dial("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("cygwin: %w", err)
	}
	err = handshake(conn, uuid, 1000, 1001, 1002)
	if err != nil {
		return nil, fmt.Errorf("cygwin: %w", err)
	}
	return conn, nil
}
