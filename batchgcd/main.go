package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ncw/gmp"
	"github.com/therealmik/batchgcd"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"encoding/base64"
	"encoding/binary"
	"math/big"
)

const (
	MODULI_BASE = 10 // Hex
	GCCOUNT     = 250000
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var algorithmName = flag.String("algorithm", "smoothparts", "mulaccum|pairwise|smoothparts|smoothparts_lowmem")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetOutput(os.Stderr)
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("No files specified")
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var f func([]*gmp.Int, chan<- batchgcd.Collision)

	switch *algorithmName {
	case "pairwise":
		f = batchgcd.BasicPairwiseGCD
	case "mulaccum":
		f = batchgcd.MulAccumGCD
	case "smoothparts":
		f = batchgcd.SmoothPartsGCD
	case "smoothparts_lowmem":
		doLowMem()
		return
	default:
		log.Fatal("Invalid algorithm: ", *algorithmName)
	}

	moduli := make([]*gmp.Int, 0)
	for _, filename := range flag.Args() {
		log.Print("Loading moduli from ", filename)
		moduli = loadModuli(moduli, filename)
	}

	ch := make(chan batchgcd.Collision, 256)
	log.Print("Executing...")
	go f(moduli, ch)

	for compromised := range uniqifyCollisions(ch) {
		if !compromised.Test() {
			log.Fatal("Test failed on ", compromised)
		}
		log.Print(compromised)

		// fmt.Println(compromised.Csv())
	}
	log.Print("Finished.")
}

func doLowMem() {
	moduli := make(chan *gmp.Int, 1)
	collisions := make(chan batchgcd.Collision, 1)

	log.Print("Executing...")
	go batchgcd.LowMemSmoothPartsGCD(moduli, collisions)

	for _, filename := range flag.Args() {
		log.Print("Reading moduli from ", filename)
		readModuli(moduli, filename)
		log.Print("Done reading moduli from ", filename)
	}
	close(moduli)

	for compromised := range uniqifyCollisions(collisions) {
		if !compromised.Test() {
			log.Fatal("Test failed on ", compromised)
		}
		fmt.Println(compromised.Csv())
	}
	log.Print("Finished.")
}

func loadModuli(moduli []*gmp.Int, filename string) []*gmp.Int {
	ch := make(chan *gmp.Int, 1)
	go func() {
		readModuli(ch, filename)
		close(ch)
	}()
	for m := range ch {
		moduli = append(moduli, m)
	}
	return moduli
}

func readModuli(ch chan *gmp.Int, filename string) {
	fp, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	defer runtime.GC()

	var count uint64
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(fp)
	log.Print("Reading ")
	for scanner.Scan() {
		count += 1
		if count%GCCOUNT == 0 {
			log.Print("Moduli read: ", count)
			runtime.GC()
		}
		m := new(gmp.Int)

		splitModuli := strings.SplitN(scanner.Text(), ",", 2)
		s := splitModuli[0] // Accept CSV moduli, so long as modulus is first column

		// Dedupe
		if _, ok := seen[s]; ok {
			continue
		} else {
			seen[s] = struct{}{}
		}
		data, _ := base64.StdEncoding.DecodeString(s)

		i := 0
		chunk, i := read_chunk(data, i)

		chunk, i = read_chunk(data, i)
		// e := unpack_bigint(chunk)

		chunk, i = read_chunk(data, i)
		n := chunk

		m.SetBytes(n)
		ch <- m
	}
}

func read_chunk(buffer []byte, i int) ([]byte, int) {
	length, start := read_int(buffer, i)
	s := buffer[start:start+length]
	return s, start + length;
}

func unpack_bigint(buffer []byte) (*big.Int) {
	b := big.NewInt(0)
	b.SetBytes(buffer)
	return b
}

func read_int(data []byte, start int) (int, int) {
	end := start + 4
	b := data[start:end]
	noBytes := binary.BigEndian.Uint32(b)
	return int(noBytes) , end;
}

func uniqifyCollisions(in <-chan batchgcd.Collision) chan batchgcd.Collision {
	out := make(chan batchgcd.Collision)
	go uniqifyProc(in, out)
	return out
}

func uniqifyProc(in <-chan batchgcd.Collision, out chan<- batchgcd.Collision) {
	seen := make(map[string]struct{})
	for c := range in {
		s := c.String()
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out <- c
	}
	close(out)
}
