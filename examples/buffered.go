package main

import (
	"bytes"
	"fmt"
	"github.com/regorov/logwriter"
	"os"
	"runtime/pprof"
	"time"
)

func main() {

	f, err := os.Create("cpu.out")
	if err != nil {
		fmt.Println(err)
		return
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	if err := os.Remove("test.log"); err != nil {
		if !os.IsNotExist(err) {
			fmt.Println(err)
			return
		}
	}

	lw, err := logwriter.NewLogWriter("test",
		&logwriter.Config{BufferSize: 2 * logwriter.MB,
			HotMaxSize: 10 * logwriter.MB,
                        CompressColdFile : true,
			ColdPath:   "", Mode: logwriter.ProductionMode}, true, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	buf := append(bytes.Repeat([]byte("R"), 256), '\n')
	fmt.Println("Started")
	t := time.Now()
	for i := 0; i < 1000000; i++ {
		n, err := lw.Write(buf)
		if err != nil {
			fmt.Println(err, n)
			break
		}
	}

	fmt.Println("Write duration: ", time.Now().Sub(t))

	if err := lw.Close(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Full duration: ", time.Now().Sub(t))

	return
}
