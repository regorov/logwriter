package logwriter_test

import (
	"log"
	"os"
	"testing"
	_ "time"

	"github.com/regorov/logwriter"
)

// Replace value with smth close to your typical log item
var TypicalLogItem string = "logwriter2015/08/17 00:33:00 Test call 48117\n"

// Direct non-buffered file write
func BenchmarkDirectFileWrite(b *testing.B) {

	f, err := os.OpenFile("directfilewrite.log", os.O_CREATE|os.O_RDWR, 0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Write([]byte(TypicalLogItem))
	}

	f.Close()

	return
}

// Write into the file by standard log package using log.Output()
func BenchmarkStandardLogToFileUsingOutput(b *testing.B) {

	f, err := os.OpenFile("standardlog-output.log", os.O_CREATE|os.O_RDWR, 0)
	if err != nil {
		b.Fatal(err)
	}

	// no prefix and date format to reduce extra work
	l := log.New(f, "", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Output(2, TypicalLogItem)
	}

	f.Close()

	return
}

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

	lw, err := logwriter.NewLogWriter("logwriter", &logwriter.Config{BufferSize: 1024*1024, ArchivePath: "", Mode: logwriter.ProductionMode}) //maxSize: 1024 * 1024})
	if err != nil {
		b.Fatal(err)
	}

	l := log.New(lw, "", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Print(TypicalLogItem)
	}

	if err := lw.Close(); err != nil {
		b.Fatal(err)
	}

	return
}
