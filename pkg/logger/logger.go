package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger 全局日志实例
	Logger zerolog.Logger
)

// LogLevel 日志级别
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
)

// Config 日志配置
type Config struct {
	Level      LogLevel
	Output     io.Writer
	TimeFormat string
	Pretty     bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:      InfoLevel,
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
		Pretty:     true,
	}
}

// Init 初始化日志系统
func Init(cfg *Config) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 设置时间格式
	zerolog.TimeFieldFormat = cfg.TimeFormat

	// 设置输出
	output := cfg.Output
	if cfg.Pretty {
		// 使用非结构化的控制台输出格式
		output = zerolog.ConsoleWriter{
			Out:        cfg.Output,
			TimeFormat: "15:04:05",
			NoColor:    false,
			// 自定义格式化函数，输出更简洁的非结构化日志
			FormatLevel: func(i interface{}) string {
				return ""
			},
			FormatTimestamp: func(i interface{}) string {
				return ""
			},
			FormatMessage: func(i interface{}) string {
				return fmt.Sprintf("%s", i)
			},
			FormatFieldName: func(i interface{}) string {
				return ""
			},
			FormatFieldValue: func(i interface{}) string {
				return ""
			},
		}
	}

	// 设置日志级别
	level := parseLogLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// 创建日志实例
	Logger = zerolog.New(output).With().Timestamp().Logger()

	// 更新全局日志
	log.Logger = Logger
}

// parseLogLevel 解析日志级别
func parseLogLevel(level LogLevel) zerolog.Level {
	switch level {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// SetLevel 设置日志级别
func SetLevel(level LogLevel) {
	zerolog.SetGlobalLevel(parseLogLevel(level))
}

// GetLogger 获取日志实例
func GetLogger() *zerolog.Logger {
	return &Logger
}

// Debug 调试日志
func Debug(msg string) {
	Logger.Debug().Msg(msg)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	Logger.Debug().Msgf(format, args...)
}

// Info 信息日志
func Info(msg string) {
	Logger.Info().Msg(msg)
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	Logger.Info().Msgf(format, args...)
}

// Warn 警告日志
func Warn(msg string) {
	Logger.Warn().Msg(msg)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	Logger.Warn().Msgf(format, args...)
}

// Error 错误日志
func Error(msg string) {
	Logger.Error().Msg(msg)
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	Logger.Error().Msgf(format, args...)
}

// Fatal 致命错误日志
func Fatal(msg string) {
	Logger.Fatal().Msg(msg)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	Logger.Fatal().Msgf(format, args...)
}

// WithField 添加字段
func WithField(key string, value interface{}) *zerolog.Event {
	return Logger.Info().Interface(key, value)
}

// WithFields 添加多个字段
func WithFields(fields map[string]interface{}) *zerolog.Event {
	event := Logger.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	return event
}
