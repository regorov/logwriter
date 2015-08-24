# logwriter
=======
Golang package logwriter automates routine related to logging into files.

## Attention
Package is under development

## Concept
### Hot and Cold logs
There is single *hot* log file. Usually file name is similar to daemon/service name and located in /var/log/servicename/.
There are *cold* log files. In accordance to rules specified by logwriter.Config,
it freezes content of *hot" file by moving content to *cold* files.


## Features
- [X] Using fixed name of file with latest log items
- [X] Support module running mode
  - **Production** - writes into the file only
  - **Debug** - writes into file and os.Stdout
- [X] Support hot file freezing rules:
  - By max file size
  - Every N msec
  - Every midnight
  - When hot log file existed on LogWriter constructed
  - Manually
- [X] File write buffering
  - Buffer size can be specified
  - Flush buffer every N msec
  - Flush buffer manually
- [X] Update configuration on the fly
- [ ] Log items re-ordering before persisting
- [ ] Log items re-ordering before archiving
- [ ] Archive log files compression
- [ ] Archive log cleaning
- [ ] Log files round robin
- [ ] Tracing log items and saving in separate .trc files
- [ ] Able to freeze hot file several times per second

## Tasks
- [ ] Add benchmarks
- [ ] Add tests
- [ ] Add examples


## Examples
Using standard log package
```
package main

import (
  "log"
  "time"
  "github/regorov/logwriter"
)

func main() {
	lw, err := logwriter.NewLogWriter("mywebserver",
	                                 &logwriter.Config{BufferSize: 0, // no buffering
	                                                   FreezeInterval : 1 * time.Hour, // create new log every hour
													   HotMaxSize : 100 * 1024 * 1024 // 100 MB
	                                                   HotPath: "/var/log/myweb",
	                                                   ColdPath: "/var/log/myweb/arch",
	                                                   Mode: logwriter.ProductionMode},
				                     true,
									 nil)
	if err != nil {
		panic(err)
	}

	logger := log.New(lw, "mywebserver", log.Ldate | log.Ltime)
	logger.Println("Module started")


	lw.Close()
	return
}
```

Using github.com/Sirupsen/logrus
```
package main

import (
  "time"
  "github.com/Sirupsen/logrus"
  "github/regorov/logwriter"
)

func errHandler(err error) {

	// send SMS
	return
}

func main() {


	lw, err := logwriter.NewLogWriter("mywebserver",
	                                 &logwriter.Config{BufferSize: 1024 * 1024, // 1 MB
	                                                   BufferFlushInterval : 3*time.Second, // flush buffer every 3 sec
	                                                   FreezeInterval : 1 * time.Hour, // create new log every hour
													   HotMaxSize : 100 * 1024 * 1024 // 100 MB
	                                                   HotPath: "/var/log/myweb",
	                                                   ColdPath: "/var/log/myweb/arch",
	                                                   Mode: logwriter.ProductionMode},
									 false,
									 errHandler))
	if err != nil {
		panic(err)
	}

	var log = logrus.New()
  	log.Out = lw

	log.WithFields(logrus.Fields{
	    "animal": "walrus",
    	"size":   10,
  	}).Info("A group of walrus emerges from the ocean")

	lw.Close()
	return
}
```
## License
MIT
