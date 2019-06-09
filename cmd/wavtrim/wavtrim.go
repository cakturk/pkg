package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/cakturk/pkg/wavtrimmer"
)

var (
	start   = flag.Duration("start", 0, "indicates the starting point")
	end     = flag.Duration("end", -1, "indicates where to stop")
	inFile  = flag.String("i", "", "input wav FILE")
	outFile = flag.String("o", "", "put newly cropped portion into this FILE: - indicates stdout")
)

func run() error {
	var err error
	flag.Parse()
	if *inFile == "" {
		return errors.New("no input file specified")
	}
	inf, err := os.Open(*inFile)
	if err != nil {
		return err
	}
	defer inf.Close()
	ouf := os.Stdout
	if *outFile != "" && *outFile != "-" {
		ouf, err = os.Open(*inFile)
		if err != nil {
			return err
		}
	}
	defer ouf.Close()
	return wavtrimmer.Trim(inf, *start, *end, ouf)
}

func main() {
	log.SetFlags(log.Lshortfile)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
