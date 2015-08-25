// Package logwriter offers a rich log file writing tools.
//
// There is single 'hot' log file per LogWriter.
// Usually file name is similar to servicename name and located in /var/log/servicename.
// All log items goes into 'hot' file.
//
// There are "cold" log files. In accordance to rules specified by Config,
// logwiter freezes content of 'hot' file by moving content to 'cold' files.
package logwriter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	//	"sync/atomic"
	"time"
)

// It is allowed to change default values listed above. Change it before calling NewLogWriter().
// It's even safe to change if you already have running LogWriter instance.
var (
	// HotFileExtension holds extension for 'hot' log file.
	HotFileExtension = "log"

	// ColdFileExtension holds extension for 'cold' log files.
	ColdFileExtension = "log"

	// TraceFileExtension holds extension for trace files. (Not implemented yet)
	TraceFileExtension = "trc"
)

// RunningMode represents application running mode
type RunningMode int

// Supported running mode options
const (
	// DebugMode orders to write log items into "hot" file and os.Stdout
	DebugMode RunningMode = 0

	// ProductionMode orders to wrire log items to the "hot" file only
	ProductionMode RunningMode = 1
)

// Size helpers
const (
	KB = 1024
	MB = 1024 * 1204
	GB = 1024 * 1024 * 1024
)

// Config holds parameters of LogWriter instance.
type Config struct {
	// Current running mode
	Mode RunningMode

	// Output buffer size. Buffering disabled if value == 0
	BufferSize int

	// Flush buffer to disk every BufferFlushInterval (works if BufferSize > 0)
	BufferFlushInterval time.Duration

	// Freeze hot file when size reaches HotMaxSize (value in bytes)
	HotMaxSize int64

	// Freeze hot file every FreezeInterval if value > 0
	FreezeInterval time.Duration

	// Freeze hot file at midnight
	FreezeAtMidnight bool

	// Folder where to open/create hot log file
	HotPath string

	// Folder where to copy cold file (frozen hot file)
	ColdPath string
}

// LogWriter wraps io.Writer to automate routine with log files.
type LogWriter struct {
	w io.Writer
	sync.RWMutex

	// instance configration
	config Config

	// buffer with capacity config.BufferSize
	buffer []byte

	// buffer allocated
	bufferLen int

	// hot and cold file name prefix
	uid string

	// hot file handle
	f *os.File

	// hot file current size
	filelen int64

	// function to sync call in case of i/o error
	errHandler func(error)

	// request to stop all active timers
	stopTimersSignal chan bool

	// timers are stopped notification
	done chan bool

	// error raised in background
	err error

	// reference to func
	coldFileNameFormatter func(string, string, time.Duration) string

	// save public variable HotFileExtension to prevent racing
	hotFileExtension string

	// save public variable CotFileExtension to prevent racing
	coldFileExtension string
}

// NewLogWriter creates new LogWriter, opens/creates hot file "%uid%.log". Hot file
// freezes immediately if freezeExisting is true and non-empty file size > 0.
func NewLogWriter(uid string, cfg *Config, freezeExisting bool, errHanldler func(error)) (*LogWriter, error) {

	lw := &LogWriter{
		uid:                   uid,
		RWMutex:               sync.RWMutex{},
		stopTimersSignal:      make(chan bool),
		done:                  make(chan bool),
		errHandler:            errHanldler,
		coldFileNameFormatter: defaultColdNameFormatter,
		hotFileExtension:      HotFileExtension,
		coldFileExtension:     ColdFileExtension}

	if cfg != nil {
		lw.config = *cfg
	}

	if lw.config.BufferSize > 0 {
		lw.buffer = make([]byte, cfg.BufferSize)

		// Not allow to have cold file size more than specified. Because buffer flushes when it's full
		if lw.config.HotMaxSize > 0 && (lw.config.HotMaxSize-int64(lw.config.BufferSize) > 0) {
			lw.config.HotMaxSize -= int64(lw.config.BufferSize)

		}
	}

	if err := lw.initHotFile(); err != nil {
		return nil, err
	}

	if freezeExisting && lw.filelen > 0 {
		// non-empty hot log file found and must be frozen
		if err := lw.freeze(false); err != nil {
			return nil, err
		}
	}

	lw.startTimers()

	return lw, nil
}

// SetColdNameFormatter replaces default 'cold' file name generator.
// Default format is "$uid-20060102-150405[.00000].log" implemented by
// function defaultColdNameFormatter().
func (lw *LogWriter) SetColdNameFormatter(f func(string, string, time.Duration) string) {
	lw.Lock()
	lw.coldFileNameFormatter = f
	lw.Unlock()
	return
}

// SetErrorFunc assigns callback function to be called when BACKGROUND i/o fails. See running() as instance.
// logwriter public functions return error withoug calling specified function.
// Please be carefull, specified user function calls synchronously!
func (lw *LogWriter) SetErrorFunc(f func(error)) {
	lw.Lock()
	lw.errHandler = f
	lw.Unlock()
	return
}

// Close stops timers, flushes buffers and closes hot file. Please call this function
// at the end of your program.
func (lw *LogWriter) Close() error {

	lw.stopTimers()

	lw.Lock()
	err := lw.close()
	lw.Unlock()
	return err
}

func (lw *LogWriter) close() error {
	if err := lw.flush(false); err != nil {
		return err
	}
	return lw.f.Close()
}

// SetConfig updates LogWriter config parameters. Func stops timers, flushes buffer,
// applies new Config, recreate buffer if need, starts timers.
func (lw *LogWriter) SetConfig(cfg *Config) error {

	lw.stopTimers()

	lw.Lock()

	if err := lw.flush(false); err != nil {
		lw.startTimers()
		lw.Unlock()
		return err
	}

	if cfg == nil {
		lw.setConfig(&Config{})
	} else {
		lw.setConfig(cfg)
	}

	lw.startTimers()
	lw.Unlock()

	return nil
}

func (lw *LogWriter) setConfig(cfg *Config) {

	oldMode := lw.config.Mode
	oldBufferSize := lw.config.BufferSize

	lw.config = *cfg

	if oldMode != cfg.Mode {
		lw.setMode(cfg.Mode)
	}

	// recreate buffer if required
	if oldBufferSize != cfg.BufferSize {
		if cfg.BufferSize > 0 {
			lw.buffer = make([]byte, cfg.BufferSize)
		} else {
			lw.buffer = nil
		}
	}

	return
}

// SetMode changes LogWriter running mode. Default value is ProductionMode
// Default value can be overwritten in NewLogWriter() or changed later by SetMode
func (lw *LogWriter) SetMode(mode RunningMode) {
	lw.Lock()
	lw.setMode(mode)
	lw.Unlock()
	return
}

func (lw *LogWriter) setMode(mode RunningMode) {

	// There is no check lw.mode == mode
	// because initHotFile() retrives new *os.File handle and should be reassigned in lw.w

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

// FlushBuffer flushes buffer if buffering enabled and buffer is not empty
func (lw *LogWriter) FlushBuffer() error {
	return lw.flushBuffer(false)
}

func (lw *LogWriter) flushBuffer(byTimer bool) error {
	lw.Lock()
	err := lw.flush(byTimer)
	lw.Unlock()
	return err
}

func (lw *LogWriter) flush(byTimer bool) error {

	if lw.config.BufferSize == 0 || lw.bufferLen == 0 {
		return nil
	}

	n, err := lw.w.Write(lw.buffer[:lw.bufferLen])

	if err != nil {
		if byTimer && lw.errHandler != nil {
			lw.errHandler(err)
			return nil
		}
		return err
	}

	lw.filelen += int64(n)
	lw.bufferLen = 0

	return nil
}

// runner triggers time based actions
func (lw *LogWriter) runner(cfg Config) {

	bufferFlushTimer := time.NewTimer(cfg.BufferFlushInterval)
	midnightTimer := time.NewTimer(time.Second)
	fileFreezeTimer := time.NewTimer(cfg.FreezeInterval)

	// All non required Timers are stopped. It allows to use single select{} operator
	// May be separate runners will be more efficient. Benchmarking required
	if cfg.BufferFlushInterval == 0 {
		bufferFlushTimer.Stop()
	}

	if !cfg.FreezeAtMidnight {
		midnightTimer.Stop()
	}

	if cfg.FreezeInterval == 0 {
		fileFreezeTimer.Stop()
	}

	// variables required for midnight passing identification
	// comparing date of last triggering with current
	now := time.Now()
	prev := now

	for {
		select {
		case _ = <-lw.stopTimersSignal:
			// stop all timers and exit
			bufferFlushTimer.Stop()
			fileFreezeTimer.Stop()
			midnightTimer.Stop()
			lw.done <- true
			return
		case _ = <-bufferFlushTimer.C:
			lw.flushBuffer(true)

			// Reset timer to compensate i/o time
			_ = bufferFlushTimer.Reset(cfg.BufferFlushInterval)
			break
		case _ = <-fileFreezeTimer.C:
			lw.freezeHotFile(true)

			// Reset timer to compensate i/o time
			_ = fileFreezeTimer.Reset(cfg.FreezeInterval)

			if bufferFlushTimer != nil {
				_ = bufferFlushTimer.Reset(cfg.BufferFlushInterval)
			}

			break
		case now = <-midnightTimer.C:
			if prev.Day() != now.Day() {
				prev = now

				lw.freezeHotFile(true)

				if cfg.FreezeInterval != 0 {
					_ = fileFreezeTimer.Reset(cfg.FreezeInterval)
				}

				if cfg.BufferFlushInterval != 0 {
					_ = bufferFlushTimer.Reset(cfg.BufferFlushInterval)
				}
			}
			break

		}
	}
}

// FreezeHotFile freezes hot file. Freeze steps: flush buffer, close file, rename hot file to
// temporary file in the same folder, rename/move temp file to cold file (async), create new hot file.
func (lw *LogWriter) FreezeHotFile() error {
	return lw.freezeHotFile(false)
}

func (lw *LogWriter) freezeHotFile(byTimer bool) error {
	lw.Lock()
	err := lw.flush(byTimer)
	if err != nil {
		lw.Unlock()
		return err
	}

	err = lw.freeze(byTimer)

	if lw.config.FreezeInterval != 0 {
		// TODO:  Reset timer in running()
	}

	lw.Unlock()

	return err

}

func (lw *LogWriter) freeze(byTimer bool) error {

	if lw.filelen == 0 {
		// nothing to do if file is empty
		return nil
	}

	if lw.f != nil {
		if err := lw.f.Close(); err != nil {
			return err
		}
	} else {
		return nil // TODO: Error
	}

	coldName := lw.coldFileNameFormatter(lw.uid, lw.coldFileExtension, lw.config.FreezeInterval)
	coldFullName := filepath.Join(lw.config.HotPath, coldName)

	// rename hot file. Keep cold file in the same folder (it is faster)
	if err := os.Rename(lw.f.Name(), coldFullName); err != nil {
		return err
	}

	archFullName := filepath.Join(lw.config.ColdPath, coldName)

	// move cold file into config.ColdPath (could be copy to another disk + delete)
	// that's why another routine
	go func(t, a string, errf func(error)) {
		if err := os.Rename(t, a); err != nil {
			if errf != nil {
				errf(err)
			} // TODO: what to do if errf() not specified
		}
	}(coldFullName, archFullName, lw.errHandler)

	return lw.initHotFile()
}

// Write 'overrides' the underlying io.Writer's Write method.
func (lw *LogWriter) Write(p []byte) (n int, err error) {

	lp := len(p)
	if lp == 0 {
		return 0, nil
	}

	lw.Lock()

	if lw.config.BufferSize > 0 {

		// if buffering enabled
		if lp+lw.bufferLen < lw.config.BufferSize {
			// and there is space in the buffer to append
			copy(lw.buffer[lw.bufferLen:], p)
			lw.bufferLen += lp
			lw.Unlock()
			return lp, nil
		}

		// if no space in the buffer do flush buffer
		n, err = lw.w.Write(lw.buffer[:lw.bufferLen])

		if err == nil {
			// copy p[] to the beginning of buffer
			lw.bufferLen = copy(lw.buffer[0:], p)
		} else {
			// complaince with http://golang.org/pkg/io/#Writer
			n = 0
		}
	} else {
		// if no buffering
		n, err = lw.w.Write(p)
	}

	if err != nil {
		lw.Unlock()
		return n, err
	}

	lw.filelen += int64(n)

	if lw.config.HotMaxSize > 0 && (lw.config.HotMaxSize < lw.filelen) {
		err = lw.freeze(false)
	}

	lw.Unlock()
	return n, err
}

// openHotFile opens/creates hot log file "%uid%.log"
func (lw *LogWriter) initHotFile() (err error) {

	lw.f, err = os.OpenFile(
		filepath.Join(lw.config.HotPath, fmt.Sprintf("%s.%s", lw.uid, lw.hotFileExtension)),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666)

	if err != nil {
		return err
	}

	fstat, err := lw.f.Stat()
	if err != nil {
		return err
	}

	lw.filelen = fstat.Size()
	fmt.Println("len", lw.filelen)

	// register lw.f in io.MultiWriter()
	lw.setMode(lw.config.Mode)

	return nil
}

func (lw *LogWriter) startTimers() {

	if (lw.config.BufferSize > 0 && lw.config.BufferFlushInterval != 0) || lw.config.FreezeAtMidnight ||
		lw.config.FreezeInterval != 0 {
		cfg := lw.config
		go lw.runner(cfg)
	}
	return
}

// stopTimers stop timers for triggering actions flush buffer and freeze hot file
func (lw *LogWriter) stopTimers() {

	lw.RLock()
	if (lw.config.BufferSize > 0 && lw.config.BufferFlushInterval != 0) || lw.config.FreezeAtMidnight || lw.config.FreezeInterval != 0 {
		lw.RUnlock()
		lw.stopTimersSignal <- true
		<-lw.done
		return
	}
	lw.RUnlock()
	return
}
func defaultColdNameFormatter(uid, ext string, d time.Duration) string {

	tformat := "20060102-150405-.000000"

	// if d (actually config.FreezeInterval) less than 1 second then file name is extended by microseconds
	// to ensure uniqueness of file names
	if d < time.Second && d > 0 {
		tformat += "-.000000"
	}

	return fmt.Sprintf("%s-%s.%s", uid, time.Now().Format(tformat), ext)
}
