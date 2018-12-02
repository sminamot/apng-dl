package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const pngHeader = "\x89PNG\r\n\x1a\n"

func main() {
	var o string
	var l int
	flag.StringVar(&o, "o", "./out.png", "output filepath")
	flag.IntVar(&l, "l", -1, "loop count, if negative number, use loop count of origin image")

	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("need to specify target dir")
	}

	res, err := http.Get(flag.Arg(0))
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	buf, _ := ioutil.ReadAll(res.Body)
	f := bytes.NewReader(buf)

	if !isPng(f) {
		fmt.Fprintln(os.Stderr, "target url is not png format")
	}

	var offset int64 = 8
	var chunks []io.Reader
	for {
		var length int32
		err := binary.Read(f, binary.BigEndian, &length)
		if err == io.EOF {
			break
		}
		chunks = append(chunks, io.NewSectionReader(f, offset, int64(length)+12))
		offset, _ = f.Seek(int64(length+8), 1)
	}

	out, err := os.Create(o)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	out.Write([]byte(pngHeader))

	for _, chunk := range chunks {
		header := make([]byte, 8)
		chunk.Read(header)
		io.Copy(out, bytes.NewReader(header))
		if string(header[4:8]) == "acTL" && l >= 0 {
			frames, loops := make([]byte, 4), make([]byte, 4)
			chunk.Read(frames)
			io.Copy(out, bytes.NewReader(frames))

			writeUint32(loops, uint32(l))
			io.Copy(out, bytes.NewReader(loops))

			c := make([]byte, 0, 12)
			c = append(c, header[4:8]...)
			c = append(c, frames...)
			c = append(c, loops...)
			crc := crc32.NewIEEE()
			crc.Write(c)
			cc := make([]byte, 4)
			writeUint32(cc, crc.Sum32())
			io.Copy(out, bytes.NewReader(cc))

			continue
		}
		io.Copy(out, chunk)
	}
}

func isPng(r io.Reader) bool {
	b := make([]byte, 8)
	r.Read(b)

	return string(b) == pngHeader
}

func writeUint32(b []uint8, u uint32) {
	b[0] = uint8(u >> 24)
	b[1] = uint8(u >> 16)
	b[2] = uint8(u >> 8)
	b[3] = uint8(u)
}
