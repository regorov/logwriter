package logwriter_test

import (
	"github.com/regorov/logwriter"
	"log"
	"os"
	"testing"
	"time"
)

func BenchmarkWriteWitoutMutex(b *testing.B) {

	f, err := os.OpenFile("withoutmutex.log", os.O_CREATE|os.O_RDWR, 0)
	if err != nil {
		b.Fatal(err)
	}

	l := log.New(f, "logwriter", log.Ldate|log.Ltime)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Printf("Test call %d", i)
	}

	f.Close()

	return
}

func BenchmarkWrite(b *testing.B) {

	lw, err := logwriter.NewLogWriter("withmutex", logwriter.LogWriterConfig{MaxDuration: 200 * time.Millisecond, ArchivePath: ""}) //maxSize: 1024 * 1024})
	if err != nil {
		b.Fatal(err)
	}

	l := log.New(lw, "logwriter", log.Ldate|log.Ltime)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Printf("Test call %d", i)
	}

	return
}
