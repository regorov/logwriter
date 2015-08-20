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
TBD

## License
MIT
