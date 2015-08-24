package logwriter_test

import (
	"bytes"
	"fmt"
	"github.com/regorov/logwriter"
	"log"
	"os"
	"sync"
	"testing"
	"time"
	_ "time"
)

// Replace value with smth close to your typical log item
var TypicalLogItem []byte = bytes.Repeat([]byte("R"), 256)

type dummy struct {
}

func (d *dummy) Write(p []byte) (int, error) {
	return len(p), nil
}

type dummyMutex struct {
	mu sync.Mutex
}

func (d *dummyMutex) Write(p []byte) (n int, err error) {
	d.mu.Lock()
	n = len(p)
	d.mu.Unlock()
	return n, nil
}

// Dummy write
func BenchmarkDummyWriter(b *testing.B) {

	d := &dummy{}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			d.Write(TypicalLogItem)
		}
	})

	return
}

// Dummy write with Mutex
func BenchmarkDummyWriterMutex(b *testing.B) {

	d := &dummyMutex{mu: sync.Mutex{}}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			d.Write(TypicalLogItem)
		}
	})

	return
}

func TestLogWriter_Write(t *testing.T) {

	lw, err := logwriter.NewLogWriter("writer",
		&logwriter.Config{BufferSize: 1024 * 1024,
			BufferFlushInterval: 1 * time.Microsecond,
			HotMaxSize:          4 * 1024 * 1024,
			ColdPath:            "coldlog/", Mode: logwriter.ProductionMode},
		false, nil)

	if err != nil {
		t.Fatal(err)
	}

	lw.Write([]byte("test1\n"))
	time.Sleep(3 * time.Second)

	lw.Write([]byte("test2\n"))
	if err := lw.FreezeHotFile(); err != nil {
		t.Fatal(err)
	}

	if err := lw.Close(); err != nil {
		t.Fatal(err)
	}

	return
}

// Channel making speed
func BenchmarkMakeChan(b *testing.B) {

	//a := make([]chan struct{}, 0)
	type resp struct {
		n   int
		err error
	}
	type req struct {
		p   []byte
		out chan resp
	}

	w := make(chan req)

	go func() {
		for {
			val := <-w
			val.out <- resp{len(val.p), nil}
		}

	}()

	b.ResetTimer()

	buf := []byte(`logwriter2015/08/17 00:33:00 Test call 48117\n`)

	for i := 0; i < b.N; i++ {
		r := req{p: buf, out: make(chan resp)}
		w <- r
		<-r.out
	}

	return
}

// Direct non-buffered file write
func BenchmarkFileWriteDirect(b *testing.B) {

	f, err := os.OpenFile("filewritedirect.log", os.O_CREATE|os.O_RDWR, 0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%s - %d\n", time.Now().String(), i)
		f.Write([]byte(s))
	}

	f.Close()

	return
}

func BenchmarkFileWriteBuffered(b *testing.B) {

	lw, err := logwriter.NewLogWriter("filewritebuffered",
		&logwriter.Config{BufferSize: 1024 * 1024,
			//BufferFlushInterval: 200 * time.Millisecond,
			HotMaxSize: 4 * 1024 * 1024,
			ColdPath:   "", Mode: logwriter.ProductionMode}, true, nil)

	if err != nil {
		b.Fatal(err)
	}

	lw.Write([]byte("test1\n"))
	lw.Write([]byte("test2\n"))
	b.ResetTimer()
	/*for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%s - %d\n", time.Now().String(), i)
		_, err := lw.Write([]byte(s))
		if err != nil {
			b.Fatal(err)
		}
	}
	*/

	if err := lw.Close(); err != nil {
		b.Fatal(err)
	}

	return
}

// Write into the file by standard log package using log.Output()
/*func BenchmarkStandardLogToFileUsingOutput(b *testing.B) {

	f, err := os.OpenFile("standardlog-output.log", os.O_CREATE|os.O_RDWR, 0)
	if err != nil {
		b.Fatal(err)
	}

	// no prefix and date format to reduce extra work
	//l := log.New(f, "", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//	l.Output(2, TypicalLogItem)
	}

	f.Close()

	return
}*/

// Write into the file by standard log package
func BenchmarkStandardLogToFileUsingPrint(b *testing.B) {

	f, err := os.OpenFile("standardlog-print.log", os.O_CREATE|os.O_RDWR, 0)
	if err != nil {
		b.Fatal(err)
	}

	// no prefix and date format to reduce extra work
	l := log.New(f, "", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Print(TypicalLogItem)
	}

	f.Close()

	return
}

//
func BenchmarkLogWrite(b *testing.B) {

	lw, err := logwriter.NewLogWriter("logwriter",
		&logwriter.Config{BufferSize: 2000000,
			BufferFlushInterval: 200 * time.Millisecond,
			ColdPath:            "", Mode: logwriter.ProductionMode},
		true,
		nil)

	if err != nil {
		b.Fatal(err)
	}

	l := log.New(lw, "logwriter ", log.Ltime|log.Lmicroseconds)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Print("Test call ", i)
	}

	if err := lw.Close(); err != nil {
		b.Fatal(err)
	}

	return
}

//
func BenchmarkLogWriteParallel(b *testing.B) {

	lw, err := logwriter.NewLogWriter("logwriter-par",
		&logwriter.Config{BufferSize: 10 * 1024 * 1024,
			HotPath: "", ColdPath: "",
			Mode: logwriter.ProductionMode},
		true,
		nil)
	if err != nil {
		b.Fatal(err)
	}

	l := log.New(lw, "logwriter ", log.Ldate|log.Ltime|log.Lmicroseconds)

	b.ResetTimer()

	k := 0
	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine has its own bytes.Buffer.
		k++
		i := 0
		for pb.Next() {
			i++
			l.Println(" Test call ", k, "-", i)
		}
	})

	if err := lw.Close(); err != nil {
		b.Fatal(err)
	}

	return
}
