// Package logwriter automates routine related to log file generation.
package logwriter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Log file switching rules
const (
	SwitchByDuration = 1 << iota
	SwitchBySize
	SwitchAtMidnight
	SwitchByStart
)

// RunningMode specifies application running mode
type RunningMode int

const (
	DebugMode      RunningMode = 0 // Writes into file and os.Stdout
	ProductionMode RunningMode = 1 // Writes into file
)

// Postfixes of environment variables
const (
	EnvDeploymentType = "_deployment_type"
	EnvLoggingLevel   = "_logging_level"
)

const logFileExtension = ".log"

// LogWriter is a special writer helping to resolve logging routines
type LogWriter struct {
	io.Writer
	sync.RWMutex
	LogWriterConfig
	uid          string
	f            *os.File
	midnigth     time.Time // Time of first succefull Write call
	total        int64     // Total # of bytes transferred
	fileName     string    // имя файла
	switchTime   time.Time
	tmpFullName  string
	archFullName string
	tmpName      string
}

type LogWriterConfig struct {
	Mode        RunningMode
	Path        string        // каталог для текущего лог файла
	MaxSize     int64         // макс размер лог файла в байтах
	MaxDuration time.Duration // макс длительность накоплений данных
	ArchivePath string        // каталог для архивных файлов
}

// NewLogWriter creates new LogWriter, opens log file
func NewLogWriter(uid string, lwc LogWriterConfig) (lw *LogWriter, err error) {

	lw = &LogWriter{
		uid:             uid,
		LogWriterConfig: lwc,
		RWMutex:         sync.RWMutex{}}

	if err := lw.createLogFile(); err != nil {
		return nil, err
	}

	return lw, nil
}

func (lw *LogWriter) SetConfig(mode RunningMode) (err error) {
	return lw.setMode(mode)
}

func (lw *LogWriter) SetMode(mode RunningMode) (err error) {
	return lw.setMode(mode)
}

func (lw *LogWriter) setMode(mode RunningMode) (err error) {

	lw.Lock()
	if lw.Mode == mode {
		lw.Unlock()
		return nil
	}

	if mode == DebugMode {
		if lw.f != nil {
			lw.Writer = io.MultiWriter(lw.f, os.Stdout)
		} else {
			lw.Writer = os.Stdout
		}
	} else if mode == ProductionMode {
		if lw.f != nil {
			fstat, err := lw.f.Stat()
			if err == nil {
				lw.Writer = lw.f
				lw.total = fstat.Size()
			}
		} else {
			lw.Writer = os.Stderr
		}
	}

	lw.Unlock()
	return err
}

// Write 'overrides' the underlying io.Writer's Write method.
// This is the one that will be called by io.Copy(). We simply
// use it to keep track of byte counts and then forward the call.
func (lw *LogWriter) Write(p []byte) (n int, err error) {

	lw.Lock()

	n, err = lw.Writer.Write(p)

	if err != nil {
		lw.Unlock()
		panic(err)
		return n, err
	}

	lw.total += int64(n)

	t := time.Now()

	if lw.MaxSize > 0 && lw.MaxSize < lw.total { // если активен режим разбиения файлов по размеру и размер файла превысил лимит

		err = lw.switchFile()

	} else if lw.MaxDuration != 0 {
		// если активен режим разбиения файлов по периоду времени

		if lw.switchTime.IsZero() {
			lw.switchTime = t.Add(lw.MaxDuration)
		} else if t.After(lw.switchTime) {
			err = lw.switchFile()
		}
	} else if t.After(lw.midnigth) {
		err = lw.switchFile()
	}

	lw.Unlock()
	if err != nil {
		panic(err)
	}

	return n, err

}

func (lw *LogWriter) SwitchFile() (err error) {

	lw.Lock()
	err = lw.switchFile()
	lw.Unlock()
	return err
}

func (lw *LogWriter) switchFile() error {

	// закрываем файл если открыт
	if lw.f != nil {
		if err := lw.f.Close(); err != nil {
			panic(err)
			return err
		}
	}

	lw.tmpName = fmt.Sprintf("%s-%d", lw.uid, time.Now().UnixNano()) // Format("2006-01-02T15-04-05-.000000")
	lw.tmpFullName = filepath.Join(lw.Path, lw.tmpName)

	if err := os.Rename(lw.f.Name(), lw.tmpFullName); err != nil {
		return err
	}

	// копируем файл в другом потоке
	lw.archFullName = filepath.Join(lw.ArchivePath, lw.tmpName+logFileExtension)

	go func(t, a string) {
		if err := os.Rename(t, a); err != nil {
			panic(err)
			lw.Write([]byte("\nLog file archiving error!\n" + err.Error()))
		}
	}(lw.tmpFullName, lw.archFullName)

	return lw.createLogFile()
}

func (lw *LogWriter) createLogFile() (err error) {

	lw.f, err = os.OpenFile(filepath.Join(lw.Path, lw.uid+logFileExtension), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err == nil {
		lw.total = 0
		lw.midnigth = time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
		lw.switchTime = time.Time{}
	}

	lw.Writer = lw.f

	return err
}

/*
func (lw *LogWriter) applyEnvConfig() error {

	// getting value of OS environmment variable uid_deployment_type
	deploymentType := strings.ToLower(os.Getenv(lw.uid + EnvDeploymentType))

	if deploymentType == "production" {
		//lw.Formatter = &logrus.JSONFormatter{}

		if lw.f != nil {
			fstat, err := lw.f.Stat()
			if err != nil {
				return err
			}
			lw.Writer = lw.f
		} else {
			lw.Writer = os.Stderr
		}

	} else {
		// The TextFormatter is default, you don't actually have to do this.
		//lw.Formatter = &logrus.TextFormatter{}
		lw.Writer = io.MultiWriter(lw.f, os.Stdout)
	}

	val := os.Getenv(lw.uid + EnvLoggingLevel)

	if len(val) == 0 {
		l.Level = logrus.DebugLevel
	} else {
		n, _ := strconv.Atoi(val)
		l.Level = logrus.Level(n)
	}

	l.WithFields(logrus.Fields{
		l.uid + EnvDeploymentType: deploymentType,
		l.uid + EnvLoggingLevel:   l.Level.String(),
	}).Info("Logger configuration updated by")

	return nil
}
*/
/*
type WriteCounter struct {
	io.Writer
	total      int64     // Total # of bytes transferred
	created    time.Time // Time of first succefull Write call
	logWrapper *LogWriter
}

func NewWriteCounter(startFrom int64, l *LogWriter) *WriteCounter {
	return &WriteCounter{total: startFrom, created: time.Now(), logWrapper: l}
}

func (l *LogWriter) UseFile(name, path string) *LogWriter {
	return l.useFile(name, path)
}
*/

func (l *LogWriter) renameFile() error {

	return nil
}

func (l *LogWriter) readyToArchive(total int64) bool {

	// TODO:
	// проверять время с последней записи
	// а так же
	return false
}

/*
func (l *LogWriter) useFile(name, path string) *LogWriter {

	if name == l.fileName && path == l.path {
		return l
	}

	if l.f != nil {
		if l.archiveLogFile(); l.Error != nil {
			return l
		}
	}

	if l.f, l.Error = os.OpenFile(filepath.Join(path, name), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); l.Error != nil {
		return l
	}

	l.applyEnvConfig()

	return l

}
*/

//os.Stdout, factorlog.NewStdFormatter("%{Date} %{Time} [%{SEV}] %{Message}")),
// создаем новый Logger
//l.Log = factorlog.New(io.MultiWriter(l.File, os.Stdout),
/*
	t := time.Now()

	if len(fileName) == 0 {
		logFileName = fmt.Sprintf("%s-%04d-%02d-%02d.log",
			fileNamePrefix, t.Year(), t.Month(), t.Day())
	} else {
		logFileName = fileName[0]
	}

	l.File, err = os.OpenFile(filepath.Join(path, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	if st, err := l.File.Stat(); err == nil {
		l.fileSize = st.Size()
	} else {
		return nil, err
	}
*/

/*
func (l *Logger) UseFile(fname string) error {
	return l.useFile()
}

func (l *Logger) useFile(fname string) error {

	l.File, err = os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	if st, err := l.File.Stat(); err == nil {
		l.fileSize = st.Size()
	} else {
		return nil, err
	}

	// L.Log.SetOutput()

	return nil
}

func (l *Logger) AddFile() error {

	return nil
}*/
