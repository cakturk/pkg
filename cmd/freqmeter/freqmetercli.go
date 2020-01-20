package main

import (
	"fmt"
	"time"

	"github.com/cakturk/pkg/freq"
)

func main() {
	var fc freq.Counter
	fc.Start()
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		fc.Cycle()
	}
	fc.Stop()
	fmt.Printf("loop roughly run at %s\n", fc.Frequency())
}
