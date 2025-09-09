/*
Refer to: nimtechnology.com
Maintainers: Nim
*/
package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Global logger instance
var Log *zap.Logger
var SugaredLog *zap.SugaredLogger

// Mutex for file lock to prevent concurrent file access
var fileLock sync.Mutex

// InitLogger initializes the logger with file rotation and timestamped logs.
func InitLogger() {
	// level
	level := zap.InfoLevel
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		level = zap.DebugLevel
	case "WARN":
		level = zap.WarnLevel
	case "ERROR":
		level = zap.ErrorLevel
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.CallerKey = "caller"

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(level),
	)

	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	SugaredLog = Log.Sugar()
}

// TimestampedLumberjackWriter wraps lumberjack.Logger and adds timestamp-based file rotation.
type TimestampedLumberjackWriter struct {
	*lumberjack.Logger
	baseFilename string
}

// NewTimestampedLumberjackWriter creates a new TimestampedLumberjackWriter.
func NewTimestampedLumberjackWriter(filename string, maxSize, maxBackups, maxAge int, compress bool) *TimestampedLumberjackWriter {
	return &TimestampedLumberjackWriter{
		Logger: &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		},
		baseFilename: filename,
	}
}

// Write writes to the original log file and rotates it with a timestamp when it's closed.
func (t *TimestampedLumberjackWriter) Write(p []byte) (n int, err error) {
	fileLock.Lock() // Lock the file access
	defer fileLock.Unlock()

	n, err = t.Logger.Write(p)
	if err != nil {
		return n, err
	}

	// Create timestamped rotated filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	rotatedFilename := fmt.Sprintf("%s.%s.gz", t.baseFilename, timestamp)

	// Attempt to compress and rename the log file with retry mechanism
	err = t.compressAndRename(rotatedFilename)
	return n, err
}

// compressAndRename handles the renaming and compressing of the log file.
func (t *TimestampedLumberjackWriter) compressAndRename(newFilename string) error {
	var err error
	for i := 0; i < 3; i++ {
		err = t.tryCompressAndRename(newFilename)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * 1) // Retry after a second
	}
	return err
}

// tryCompressAndRename performs the actual compression and renaming.
func (t *TimestampedLumberjackWriter) tryCompressAndRename(newFilename string) error {
	// Open the current log file
	currentFile, err := os.Open(t.baseFilename)
	if err != nil {
		return err
	}
	defer currentFile.Close()

	// Create a temporary file
	tmpFile, err := os.Create(newFilename + ".tmp")
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	// Rename the current log file to temporary file
	err = os.Rename(t.baseFilename, tmpFile.Name())
	if err != nil {
		return err
	}

	// Compress the temporary file
	err = compressFile(tmpFile.Name())
	if err != nil {
		return err
	}

	// Finally, rename the temporary file to the desired final filename
	err = os.Rename(tmpFile.Name(), newFilename)
	return err
}

// compressFile compresses the log file to gzip format.
func compressFile(filename string) error {
	// Open the renamed log file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gzip file with a .gz extension
	compressedFile, err := os.Create(filename + ".gz")
	if err != nil {
		return err
	}
	defer compressedFile.Close()

	// Compress the log file
	gzWriter := gzip.NewWriter(compressedFile)
	defer gzWriter.Close()

	// Copy the contents of the original file into the gzip file
	_, err = io.Copy(gzWriter, file)
	return err
}

// Wrapper functions for Infof, Debugf, Warnf, and Errorf

func Infof(format string, args ...interface{}) {
	SugaredLog.Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	SugaredLog.Debugf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	SugaredLog.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	SugaredLog.Errorf(format, args...)
}

// Function to log with structured logging
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// ZapWriter implements io.Writer, forwarding logs to a Zap logger.
type ZapWriter struct {
	logger *zap.Logger
	level  zapcore.Level
}

// NewZapWriter creates a new ZapWriter instance.
func NewZapWriter(logger *zap.Logger, level zapcore.Level) *ZapWriter {
	return &ZapWriter{logger: logger, level: level}
}

// Write satisfies io.Writer interface, forwarding output to the Zap logger.
func (z *ZapWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	switch z.level {
	case zapcore.DebugLevel:
		z.logger.Debug(msg)
	case zapcore.InfoLevel:
		z.logger.Info(msg)
	case zapcore.WarnLevel:
		z.logger.Warn(msg)
	case zapcore.ErrorLevel:
		z.logger.Error(msg)
	default:
		z.logger.Info(msg)
	}
	return len(p), nil
}
