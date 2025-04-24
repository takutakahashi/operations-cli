package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogLevel はログレベルを表します
type LogLevel int

const (
	// ログレベルの定義
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String はLogLevelを文字列に変換します
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger はロギングインターフェースを定義します
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
	Println(v ...interface{})
	Printf(format string, v ...interface{})
	SetLevel(level LogLevel)
	GetLevel() LogLevel
}

// BaseLogger は基本的なロガーの実装です
type BaseLogger struct {
	logger *log.Logger
	writer io.Writer
	level  LogLevel
}

// NewBaseLogger は新しいBaseLoggerを作成します
func NewBaseLogger(w io.Writer) *BaseLogger {
	return &BaseLogger{
		logger: log.New(os.Stderr, "", log.LstdFlags),
		writer: w,
		level:  INFO, // デフォルトはINFOレベル
	}
}

func (l *BaseLogger) log(level LogLevel, format string, v ...interface{}) {
	if level >= l.level {
		msg := fmt.Sprintf(format, v...)
		l.logger.Printf("[%s] %s", level.String(), msg)
	}
}

func (l *BaseLogger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

func (l *BaseLogger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

func (l *BaseLogger) Warn(format string, v ...interface{}) {
	l.log(WARN, format, v...)
}

func (l *BaseLogger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

func (l *BaseLogger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *BaseLogger) GetLevel() LogLevel {
	return l.level
}

func (l *BaseLogger) Println(v ...interface{}) {
	l.logger.Println(v...)
}

func (l *BaseLogger) Printf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

// FileLogger はファイルにログを出力するロガーの実装です
type FileLogger struct {
	*BaseLogger
	file *os.File
}

// NewFileLogger は新しいFileLoggerを作成します
func NewFileLogger(logDir string) (*FileLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("mcp-server_%s.log", timestamp))

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileLogger{
		BaseLogger: NewBaseLogger(file),
		file:       file,
	}, nil
}

func (l *FileLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// StdoutLogger は標準出力にログを出力するロガーの実装です
type StdoutLogger struct {
	*BaseLogger
}

// NewStdoutLogger は新しいStdoutLoggerを作成します
func NewStdoutLogger() *StdoutLogger {
	return &StdoutLogger{
		BaseLogger: NewBaseLogger(os.Stderr),
	}
}

// MultiLogger は複数のWriterにログを出力するロガーの実装です
type MultiLogger struct {
	*BaseLogger
}

// NewMultiLogger は新しいMultiLoggerを作成します
func NewMultiLogger(writers ...io.Writer) *MultiLogger {
	return &MultiLogger{
		BaseLogger: NewBaseLogger(io.MultiWriter(writers...)),
	}
}

// NullLogger はログを出力しないロガーの実装です
type NullLogger struct {
	*BaseLogger
}

// NewNullLogger は新しいNullLoggerを作成します
func NewNullLogger() *NullLogger {
	return &NullLogger{
		BaseLogger: NewBaseLogger(io.Discard),
	}
}

// LoggerConfig はロガーの設定を管理します
type LoggerConfig struct {
	Level   LogLevel
	Outputs []LogOutput
	LogDir  string
}

// LogOutput はログの出力先を表します
type LogOutput int

const (
	OutputStdout LogOutput = iota
	OutputFile
)

// NewLoggerFromEnv は環境変数からロガーを作成します
func NewLoggerFromEnv() Logger {
	config := LoggerConfig{
		Level:   INFO,
		Outputs: []LogOutput{OutputStdout},
	}

	// ログレベルの設定
	if level := os.Getenv("OM_LOG_LEVEL"); level != "" {
		switch level {
		case "DEBUG":
			config.Level = DEBUG
		case "INFO":
			config.Level = INFO
		case "WARN":
			config.Level = WARN
		case "ERROR":
			config.Level = ERROR
		}
	}

	// 出力先の設定
	if output := os.Getenv("OM_LOG_OUTPUT"); output != "" {
		config.Outputs = []LogOutput{}
		outputs := strings.Split(output, ",")
		for _, o := range outputs {
			switch strings.TrimSpace(o) {
			case "stdout":
				config.Outputs = append(config.Outputs, OutputStdout)
			case "file":
				config.Outputs = append(config.Outputs, OutputFile)
			}
		}
	}

	// ログディレクトリの設定
	if logDir := os.Getenv("OM_LOG_DIR"); logDir != "" {
		config.LogDir = logDir
	}

	return NewLoggerFromConfig(config)
}

// NewLoggerFromConfig は設定からロガーを作成します
func NewLoggerFromConfig(config LoggerConfig) Logger {
	var writers []io.Writer

	for _, output := range config.Outputs {
		switch output {
		case OutputStdout:
			writers = append(writers, os.Stdout)
		case OutputFile:
			if config.LogDir != "" {
				if fileLogger, err := NewFileLogger(config.LogDir); err == nil {
					writers = append(writers, fileLogger.writer)
				}
			}
		}
	}

	// 出力先が設定されていない場合はNullLoggerを返す
	if len(writers) == 0 {
		logger := NewNullLogger()
		logger.SetLevel(config.Level)
		return logger
	}

	// 出力先が1つの場合
	if len(writers) == 1 {
		var logger Logger
		if writers[0] == os.Stdout {
			logger = NewStdoutLogger()
		} else {
			logger = NewBaseLogger(writers[0])
		}
		logger.SetLevel(config.Level)
		return logger
	}

	// 複数の出力先がある場合
	logger := NewMultiLogger(writers...)
	logger.SetLevel(config.Level)
	return logger
}
