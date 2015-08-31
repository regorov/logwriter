package main

import (
	"github.com/regorov/logwriter"
	"log"
	"time"
)

func main() {
	cfg := &logwriter.Config{
		BufferSize:       0,                  // no buffering
		FreezeInterval:   1 * time.Hour,      // freeze log file every hour
		HotMaxSize:       100 * logwriter.MB, // 100 MB max file size
		CompressColdFile: true,               // compress cold file
		HotPath:          "/var/log/mywebserver",
		ColdPath:         "/var/log/mywebserver/arch",
		Mode:             logwriter.ProductionMode, // write to file only
	}

	lw, err := logwriter.NewLogWriter("mywebserver",
		cfg,
		true, // freeze hot file if exists
		nil)

	if err != nil {
		panic(err)
	}

	logger := log.New(lw, "mywebserver", log.Ldate|log.Ltime)
	logger.Println("Module started")

	if err := lw.Close(); err != nil {
		// Error handling
	}

	return
}
