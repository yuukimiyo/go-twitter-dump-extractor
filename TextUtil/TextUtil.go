// package TextUtil
package main

import (
	"fmt"
	"flag"
	"github.com/golang/glog"
	"bytes"
	"compress/zlib"
	"io"
)

func init() {
	flag.Parse()
}

func Deflate(plainString string) {
	glog.V(3).Infof("input: " + plainString)

	zr, err := compress(bytes.NewBufferString("hello"))
	if err != nil {
		panic(err)
	}

	//display compressed data
	b := zr.Bytes()
	fmt.Printf("%d bytes: %v\n", len(b), b)
}

func compressStringToZlib(r io.Reader) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zw := zlib.NewWriter(buf)
	defer zw.Close()

	if _, err := io.Copy(zw, r); err != nil {
		return buf, err
	}
	return buf, nil
}

func extract(zr io.Reader) (io.Reader, error) {
	return zlib.NewReader(zr)
}

func main() {
	glog.V(3).Infof("start")

	Deflate("test string")
}
