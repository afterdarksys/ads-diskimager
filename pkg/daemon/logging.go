package daemon

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"strings"
)

// LogLevel represents logging severity
type LogLevel int

const (
	// LogLevelDebug for debug messages
	LogLevelDebug LogLevel = iota
	// LogLevelInfo for informational messages
	LogLevelInfo
	// LogLevelWarn for warning messages
	LogLevelWarn
	// LogLevelError for error messages
	LogLevelError
	// LogLevelFatal for fatal messages
	LogLevelFatal
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
	Close() error
}

// DaemonLogger implements Logger with syslog/journald support
type DaemonLogger struct {
	syslogWriter *syslog.Writer
	stdLogger    *log.Logger
	minLevel     LogLevel
	useSyslog    bool
	identifier   string
}

// NewLogger creates a new daemon logger
func NewLogger(identifier string, useSyslog bool, minLevel LogLevel) (*DaemonLogger, error) {
	logger := &DaemonLogger{
		identifier: identifier,
		useSyslog:  useSyslog,
		minLevel:   minLevel,
	}

	if useSyslog {
		// Try to connect to syslog/journald
		writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, identifier)
		if err != nil {
			// Fallback to stderr if syslog is not available
			logger.useSyslog = false
			logger.stdLogger = log.New(os.Stderr, fmt.Sprintf("[%s] ", identifier), log.LstdFlags)
			return logger, fmt.Errorf("failed to connect to syslog, using stderr: %w", err)
		}
		logger.syslogWriter = writer
	} else {
		// Use standard logger to stderr
		logger.stdLogger = log.New(os.Stderr, fmt.Sprintf("[%s] ", identifier), log.LstdFlags)
	}

	return logger, nil
}

// NewConsoleLogger creates a logger that writes to stdout
func NewConsoleLogger(identifier string, minLevel LogLevel) *DaemonLogger {
	return &DaemonLogger{
		identifier: identifier,
		useSyslog:  false,
		minLevel:   minLevel,
		stdLogger:  log.New(os.Stdout, fmt.Sprintf("[%s] ", identifier), log.LstdFlags),
	}
}

// Debug logs a debug message
func (l *DaemonLogger) Debug(msg string) {
	if l.minLevel > LogLevelDebug {
		return
	}
	l.log(LogLevelDebug, msg)
}

// Info logs an informational message
func (l *DaemonLogger) Info(msg string) {
	if l.minLevel > LogLevelInfo {
		return
	}
	l.log(LogLevelInfo, msg)
}

// Warn logs a warning message
func (l *DaemonLogger) Warn(msg string) {
	if l.minLevel > LogLevelWarn {
		return
	}
	l.log(LogLevelWarn, msg)
}

// Error logs an error message
func (l *DaemonLogger) Error(msg string) {
	if l.minLevel > LogLevelError {
		return
	}
	l.log(LogLevelError, msg)
}

// Fatal logs a fatal message and exits
func (l *DaemonLogger) Fatal(msg string) {
	l.log(LogLevelFatal, msg)
	os.Exit(1)
}

// log performs the actual logging
func (l *DaemonLogger) log(level LogLevel, msg string) {
	formattedMsg := fmt.Sprintf("[%s] %s", level.String(), msg)

	if l.useSyslog && l.syslogWriter != nil {
		// Write to syslog with appropriate priority
		switch level {
		case LogLevelDebug:
			l.syslogWriter.Debug(msg)
		case LogLevelInfo:
			l.syslogWriter.Info(msg)
		case LogLevelWarn:
			l.syslogWriter.Warning(msg)
		case LogLevelError:
			l.syslogWriter.Err(msg)
		case LogLevelFatal:
			l.syslogWriter.Crit(msg)
		}
	} else if l.stdLogger != nil {
		// Write to standard logger
		l.stdLogger.Println(formattedMsg)
	}
}

// Close closes the logger
func (l *DaemonLogger) Close() error {
	if l.syslogWriter != nil {
		return l.syslogWriter.Close()
	}
	return nil
}

// SetOutput sets the output destination for non-syslog logging
func (l *DaemonLogger) SetOutput(w io.Writer) {
	if l.stdLogger != nil {
		l.stdLogger.SetOutput(w)
	}
}

// ParseLogLevel parses a log level string
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

// MultiLogger writes to multiple loggers
type MultiLogger struct {
	loggers []Logger
}

// NewMultiLogger creates a logger that writes to multiple destinations
func NewMultiLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{
		loggers: loggers,
	}
}

// Debug logs to all loggers
func (m *MultiLogger) Debug(msg string) {
	for _, l := range m.loggers {
		l.Debug(msg)
	}
}

// Info logs to all loggers
func (m *MultiLogger) Info(msg string) {
	for _, l := range m.loggers {
		l.Info(msg)
	}
}

// Warn logs to all loggers
func (m *MultiLogger) Warn(msg string) {
	for _, l := range m.loggers {
		l.Warn(msg)
	}
}

// Error logs to all loggers
func (m *MultiLogger) Error(msg string) {
	for _, l := range m.loggers {
		l.Error(msg)
	}
}

// Fatal logs to all loggers and exits
func (m *MultiLogger) Fatal(msg string) {
	for _, l := range m.loggers {
		l.Error(msg) // Use Error instead of Fatal to log to all
	}
	os.Exit(1)
}

// Close closes all loggers
func (m *MultiLogger) Close() error {
	var errs []string
	for _, l := range m.loggers {
		if err := l.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing loggers: %s", strings.Join(errs, "; "))
	}
	return nil
}
