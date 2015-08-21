# logwriter
=======
Golang package logwriter automates routine related to logging into files.

## Attention
Package is under development

## Features
- [X] Using fixed name of file with latest log items 
- [X] Support module running mode
  - **Production** - writes into the file only
  - **Debug** - writes into file and os.Stdout
- [X] Support file switching rules:
  - By max file size
  - Every N msec
  - Every midnight
  - If log file exists when module started
  - Manual
- [ ] File write buffering
  - Buffer size can be specified
  - Flush buffer every N msec
  - Flush buffer manually
- [ ] Log items re-ordering before persisting
- [ ] Log items re-ordering before archiving
- [ ] Archive log files compression
- [ ] Archive log cleaning
- [ ] Log files round robin
- [ ] Update configuration on the fly
- [ ] Trace mode
   - 

## Tasks
- [ ] Add benchmarks
- [ ] Add tests
- [ ] Add examples

## Examples

Using standard log package
```
import (
  "log"
  "time"
  "github/regorov/logwriter"
)

func main() {
	lw, err := logwriter.NewLogWriter("myweb", 
	                                 &logwriter.Config{BufferSize: 10 * 1024 * 1024, 
	                                                   SwitchInterval : time.Second,
	                                                   SwitchInterval : 1 * time.Hour,
	                                                   Path: "/var/log/myweb",
	                                                   ArchivePath: "/var/log/myweb/arch", 
	                                                   Mode: logwriter.ProductionMode}) 
	if err != nil {
		panic(err)
	}

	logger := log.New(lw, "myweb", log.Ldate | log.Ltime)
	logger.Println("Module started")
	
	
	lw.Close()
	return
}
```

## License
MIT
