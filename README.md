# logwriter
Golang package logwriter automates routine related to logging into files.

[![GoDoc](https://godoc.org/github.com/regorov/logwriter?status.svg)](https://godoc.org/github.com/regorov/logwriter)
[![Build Status](https://drone.io/github.com/regorov/logwriter/status.png)](https://drone.io/github.com/regorov/logwriter/latest)
[![Coverage Status](https://coveralls.io/repos/regorov/logwriter/badge.svg?branch=master&service=github)](https://coveralls.io/github/regorov/logwriter?branch=master)

Initial version finished. Stabilization, testing and benchmarkign are going.

## Concepts
#### Hot and Cold Log Files
There is a single **hot** log file. Usually file name is similar to daemon/service name and located in */var/log/servicename/*. There are **cold** log files. In accordance to rules specified by logwriter.Config,
logwriter freezes content of **hot** file by moving content to new **cold** file.

#### Using sync.Mutex
If you don't need buffering (logwriter.Config.BufferSize==0) you can believe that file write executes synchronously.

#### Stop! It's not a *unix way
Oh nooo. Not everyone develops Facebook (c) or smth similar daily :)

## Features
- [X] Folders for hot and cold log files configurable
- [X] Using fixed name of file with latest log items
- [X] Support module running mode
  - **Production** - writes into the file only
  - **Debug** - writes into file and os.Stdout
- [X] Support hot file freezing rules:
  - By max file size
  - Every time.Duration
  - Every midnight
  - Manually
  - Freeze when your appication starts
- [X] File write buffering
  - Configurable buffer size
  - Flush buffer every time.Duration
  - Flush buffer manually
- [X] Update configuration on the fly
- [X] Cold log files compression
- [ ] Log items re-ordering before persisting
- [ ] Log items re-ordering on freezing stage
- [ ] Cold files cleaning
- [ ] Cold log files round robin
- [ ] Tracing option. Saving some of log items in separate .trc files
- [ ] Ability to freeze hot file several times per second

## Tasks
- [ ] Add benchmarks
- [ ] Add tests
- [ ] Add examples


## Examples
Using standard log package
```Go
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

```

Using github.com/Sirupsen/logrus
```Go
package main

import (
  "time"
  "github.com/Sirupsen/logrus"
  "github.com/regorov/logwriter"
)

func errHandler(err error) {

	// send SMS or Smth
	return
}

func main() {

	lw, err := logwriter.NewLogWriter("mywebserver",
	                                  &logwriter.Config{
									      BufferSize: 1024 * 1024, // 1 MB
	                                      BufferFlushInterval : 3 * time.Second, // flush buffer every 3 sec
	                                      FreezeInterval : 1 * time.Hour, // create new log every hour
							              HotMaxSize : 100 * 1024 * 1024, // or when hot file size over 100 MB
	                                      HotPath: "/var/log/myweb",
	                                      ColdPath: "/var/log/myweb/arch",
	                                      Mode: logwriter.ProductionMode,
										},
					                  false, // do not freeze hot file if exists
					                  errHandler))
	if err != nil {
		// Error handling
	}

	var log = logrus.New()
  	log.Out = lw

	log.WithFields(logrus.Fields{"animal": "walrus",
        	                     "size":   10,
  	}).Info("A group of walrus emerges from the ocean")

	if err := lw.Close(); err != nil {
        // Error handling
    }

	return
}
```
## License
MIT
