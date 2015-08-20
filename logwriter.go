// Package logwriter automates log file writing routines.
package logwriter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// Default extension for log files. Change it if required.
	LogFileExtension string = ".log"

	// Default extension for trace files. Change it if required.
	TraceFileExtension string = ".trc"
)

// RunningMode specifies application running mode
type RunningMode int

const (
	// Writes into file and os.Stdout
	DebugMode RunningMode = 0

	// Writes into file
	ProductionMode RunningMode = 1
)

// Config stores LogWriter parameters
type Config struct {
	// Current running mode
	Mode RunningMode

	// Output buffer size
	BufferSize int

	// Buffer flush to HDD every
	BufferFlushInterval time.Duration

	// Create new log file if existing size is over
	SwitchSize int64

	// Create new log file every SwitchInterval (after the 1st log item arrival)
	SwitchInterval time.Duration

	// Create new log file at midnight
	SwitchAtMidnight bool

	// Where to create active log file
	Path string

	// Where to copy log file after 'switching'
	ArchivePath string
}


// LogWriter wraps io.Writer to automate routine with log files
type LogWriter struct {
	w io.Writer
	sync.RWMutex

	// LW instance configration
	config Config

	buf []byte
	buftotal int 

	uid string
	f   *os.File

	// coming midnight
	midnigth time.Time

	// active log file length
	total int64

	// active log file name (usually $uid.log)
	fileName string

	// time of next log file swithing. If IsZero() == true, feature not used
	switchTime time.Time
}


// NewLogWriter creates new LogWriter and main log file
func NewLogWriter(uid string, cfg *Config) (lw *LogWriter, err error) {

	lw = &LogWriter{
		uid:     uid,
		RWMutex: sync.RWMutex{}}

	if cfg != nil {
		lw.config = *cfg
	}

	if lw.config.BufferSize > 0 {
		// reserve double len
		lw.buf = make([]byte, cfg.BufferSize*2)
	}

	if err := lw.createLogFile(); err != nil {
		return nil, err
	}

	return lw, nil
}

// SetConfig updates LogWriter config parameters
func (lw *LogWriter)SetConfig(cfg *Config) {
	
	lw.Lock()
	if cfg == nil {
		lw.setConfig(&Config{})
	} else {
		lw.setConfig(cfg)
	}
	lw.Unlock()
	
	return
}


func (lw *LogWriter)setConfig(cfg *Config) {
	
	oldmode := lw.config.Mode
	lw.config = *cfg
	
	if  oldmode != cfg.Mode {
		lw.setMode(cfg.Mode)
	}
		
	return
}

// Close flushes file buffer and closes log file
func (lw *LogWriter) Close() (err error) {
	lw.Lock()
	err = lw.close()
	lw.Unlock()
	return
}

func (lw *LogWriter) close() (err error) {

	if lw.f == nil {
		return nil
	}

	// TODO: flushbuffer

	return lw.f.Close()
}

func (lw *LogWriter) SetMode(mode RunningMode) {
	lw.Lock()
	lw.setMode(mode)
	lw.Unlock()
}

func (lw *LogWriter) setMode(mode RunningMode) {

	
	if mode == DebugMode {
		if lw.f != nil {
			lw.w = io.MultiWriter(lw.f, os.Stdout)
		} else {
			lw.w = os.Stdout
		}
	} else if mode == ProductionMode {
		if lw.f != nil {
			lw.w = lw.f
		} else {
			lw.w = os.Stderr
		}
	}
	
	lw.config.Mode = mode


	return
}

// Write 'overrides' the underlying io.Writer's Write method.
func (lw *LogWriter) Write(p []byte) (n int, err error) {

	// TODO: buffered i/o
	if len(p) == 0 {
		return 0, nil
	}
	
	lw.Lock()
	
	
	if len(lw.buf) > 0 {
		if 	len(p) + lw.buftotal < len(lw.buf)/2 {
			copy(lw.buf[lw.buftotal:], p)
			lw.buftotal += len(p)
			lw.Unlock()
			return len(p), nil
		} else {
			p = lw.buf[:lw.buftotal]
			lw.buftotal = 0
		}
	} 
	
	 
	
	n, err = lw.w.Write(p)

	if err != nil {
		lw.Unlock()
		return n, err
	}	

	lw.total += int64(n)
		
	doSwitch := (lw.config.SwitchSize > 0 && lw.config.SwitchSize < lw.total) ||
		(lw.config.SwitchInterval != 0 && !lw.switchTime.IsZero() && time.Now().After(lw.switchTime))

	if !doSwitch {
		if lw.switchTime.IsZero() {
			lw.switchTime = time.Now().Add(lw.config.SwitchInterval)
		}

		doSwitch = lw.config.SwitchAtMidnight && time.Now().After(lw.midnigth)
	}

	if doSwitch {
		err = lw.switchFile()
	}
	
	
	lw.Unlock()

	return n, err
}

// ForceSwitchFile immediately archives active log file
func (lw *LogWriter) ForceSwitchFile() (err error) {
	lw.Lock()
	err = lw.switchFile()
	lw.Unlock()
	return err
}

//
func (lw *LogWriter) switchFile() error {

	// close() file if it's open
	if lw.f != nil {
		if err := lw.f.Close(); err != nil {
			return err
		}
	}

	tmpName := fmt.Sprintf("%s-%d", lw.uid, time.Now().UnixNano()) // Format("2006-01-02T15-04-05-.000000")
	tmpFullName := filepath.Join(lw.config.Path, tmpName)

	if err := os.Rename(lw.f.Name(), tmpFullName); err != nil {
		return err
	}

	archFullName := filepath.Join(lw.config.ArchivePath, tmpName+LogFileExtension)

	// rename (probably copy file) in parallel routine
	go func(t, a string) {
		if err := os.Rename(t, a); err != nil {
			lw.Write([]byte("\nLog file archiving error!\n" + err.Error()))
		}
	}(tmpFullName, archFullName)

	return lw.createLogFile()
}

// Creates active log file "$uid.log"
func (lw *LogWriter) createLogFile() (err error) {

	lw.f, err = os.OpenFile(filepath.Join(lw.config.Path, lw.uid+LogFileExtension), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		return err
	}

	fstat, err := lw.f.Stat()
	if err != nil {
		return err
	}

	lw.total = fstat.Size()
	lw.midnigth = time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
	lw.switchTime = time.Time{}

	lw.setMode(lw.config.Mode)

	return nil

}
