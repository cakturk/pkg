package main

import (
	"fmt"
	"testing"
)

func TestUnpack(t *testing.T) {
	p := []byte{2, 3, 4, 5, 6}
	len := len(p)
	i := 0

	for i+1 < len {
		fmt.Printf("i: %d, len: %d, %v\n", i, len, p[i:i+2])
		i += 2
	}
}
