## logwriter
Golang package logwriter automates routine related to logging into files.

## Features
- Using fixed name of file with latest log items
- Support module running mode
  - **Production** - writes into the file only
  - **Debug** - writes into file and os.Stdout
- Supported file switch rules:
  - By file size
  - By duration from latest write
  - Every midnight
  - If log file exists when module started
  - Manual
- Write buffering
- Log items re-ordering before persisting
- Log items re-ordering before archiving
- Archive log files compression
- Archive log cleaning
- Log files round robin

## Examples

## License
MIT
