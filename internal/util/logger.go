package util

import (
	"ai-ops/internal/util/errors"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// 日志级别字符串映射
var logLevelNames = map[LogLevel]string{
	LogLevelDebug: "DEBUG",
	LogLevelInfo:  "INFO",
	LogLevelWarn:  "WARN",
	LogLevelError: "ERROR",
}

// 日志级别颜色映射（用于终端输出）
var logLevelColors = map[LogLevel]string{
	LogLevelDebug: "\033[36m", // 青色
	LogLevelInfo:  "\033[32m", // 绿色
	LogLevelWarn:  "\033[33m", // 黄色
	LogLevelError: "\033[31m", // 红色
}

const colorReset = "\033[0m"

// 日志器结构
type Logger struct {
	level       LogLevel
	format      string // "json" 或 "text"
	output      io.Writer
	enableColor bool
	logger      *log.Logger
}

// 全局日志器实例
var DefaultLogger *Logger

// 初始化默认日志器
func init() {
	DefaultLogger = NewLogger(LogLevelInfo, "text", os.Stdout, true)
}

// 创建新的日志器
func NewLogger(level LogLevel, format string, output io.Writer, enableColor bool) *Logger {
	return &Logger{
		level:       level,
		format:      format,
		output:      output,
		enableColor: enableColor,
		logger:      log.New(output, "", 0),
	}
}

// 从字符串解析日志级别
func ParseLogLevel(levelStr string) LogLevel {
	switch strings.ToLower(levelStr) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// 设置输出格式
func (l *Logger) SetFormat(format string) {
	l.format = format
}

// 记录日志
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := logLevelNames[level]

	if l.format == "json" {
		l.logJSON(timestamp, levelName, message, fields)
	} else {
		l.logText(timestamp, levelName, level, message, fields)
	}
}

// 文本格式日志
func (l *Logger) logText(timestamp, levelName string, level LogLevel, message string, fields map[string]interface{}) {
	var output strings.Builder

	// 添加颜色（如果启用）
	if l.enableColor {
		output.WriteString(logLevelColors[level])
	}

	// 基本信息
	output.WriteString(fmt.Sprintf("[%s] %s %s", timestamp, levelName, message))

	// 添加字段
	if len(fields) > 0 {
		output.WriteString(" |")
		for key, value := range fields {
			output.WriteString(fmt.Sprintf(" %s=%v", key, value))
		}
	}

	// 重置颜色
	if l.enableColor {
		output.WriteString(colorReset)
	}

	l.logger.Println(output.String())
}

// JSON格式日志
func (l *Logger) logJSON(timestamp, levelName, message string, fields map[string]interface{}) {
	data := map[string]interface{}{
		"timestamp": timestamp,
		"level":     levelName,
		"message":   message,
	}
	if len(fields) > 0 {
		for k, v := range fields {
			if err, ok := v.(error); ok {
				data[k] = err.Error()
			} else {
				data[k] = v
			}
		}
	}
	b, err := json.Marshal(data)
	if err != nil {
		l.logger.Println(fmt.Sprintf("[%s] %s %s | marshal_error=%v", timestamp, levelName, message, err))
		return
	}
	l.logger.Println(string(b))
}

// Debug级别日志
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message, nil)
}

// Debug级别日志（带字段）
func (l *Logger) Debugw(message string, fields map[string]interface{}) {
	l.log(LogLevelDebug, message, fields)
}

// Info级别日志
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message, nil)
}

// Info 级别日志（带字段）
func (l *Logger) Infow(message string, fields map[string]interface{}) {
	l.log(LogLevelInfo, message, fields)
}

// Warn 级别日志
func (l *Logger) Warn(message string) {
	l.log(LogLevelWarn, message, nil)
}

// Warnw 级别日志（带字段）
func (l *Logger) Warnw(message string, fields map[string]interface{}) {
	l.log(LogLevelWarn, message, fields)
}

// Error级别日志
func (l *Logger) Error(message string) {
	l.log(LogLevelError, message, nil)
}

// Error级别日志（带字段）
func (l *Logger) Errorw(message string, fields map[string]interface{}) {
	l.log(LogLevelError, message, fields)
}

// 记录错误对象
func (l *Logger) LogError(err error, context string) {
	if err == nil {
		return
	}

	fields := map[string]interface{}{
		"context": context,
		"error":   err.Error(),
	}

	if appErr, ok := err.(*errors.AppError); ok {
		fields["error_code"] = appErr.Code
		if appErr.Details != "" {
			fields["details"] = appErr.Details
		}
	}

	l.log(LogLevelError, "发生错误", fields)
}

// 记录错误对象（带额外字段）
func (l *Logger) LogErrorWithFields(err error, context string, extraFields map[string]interface{}) {
	if err == nil {
		return
	}

	fields := map[string]interface{}{
		"context": context,
		"error":   err.Error(),
	}

	// 添加额外字段
	for key, value := range extraFields {
		fields[key] = value
	}

	if appErr, ok := err.(*errors.AppError); ok {
		fields["error_code"] = appErr.Code
		if appErr.Details != "" {
			fields["details"] = appErr.Details
		}
	}

	l.log(LogLevelError, "发生错误", fields)
}

// Debug 全局日志函数（使用默认日志器）
func Debug(message string) {
	DefaultLogger.Debug(message)
}

func Debugw(message string, fields map[string]interface{}) {
	DefaultLogger.Debugw(message, fields)
}

func Info(message string) {
	DefaultLogger.Info(message)
}

func Infow(message string, fields map[string]interface{}) {
	DefaultLogger.Infow(message, fields)
}

func Warn(message string) {
	DefaultLogger.Warn(message)
}

func Warnw(message string, fields map[string]interface{}) {
	DefaultLogger.Warnw(message, fields)
}

func Error(message string) {
	DefaultLogger.Error(message)
}

func Errorw(message string, fields map[string]interface{}) {
	DefaultLogger.Errorw(message, fields)
}

func LogError(err error, context string) {
	DefaultLogger.LogError(err, context)
}

func LogErrorWithFields(err error, context string, extraFields map[string]interface{}) {
	DefaultLogger.LogErrorWithFields(err, context, extraFields)
}

// InitLogger 初始化日志器（根据配置）
func InitLogger(level, format, output, file string) error {
	logLevel := ParseLogLevel(level)

	// 关闭旧的文件句柄（如有）
	if DefaultLogger != nil {
		if f, ok := DefaultLogger.output.(*os.File); ok && f != os.Stdout && f != os.Stderr {
			_ = f.Close()
		}
	}

	var writer io.Writer
	var enableColor bool

	switch output {
	case "stdout":
		writer = os.Stdout
		enableColor = true
	case "stderr":
		writer = os.Stderr
		enableColor = true
	case "file":
		if file == "" {
			return errors.NewError(errors.ErrCodeConfigInvalid, "日志输出为文件时必须指定文件路径")
		}
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return errors.WrapError(errors.ErrCodeConfigInvalid, "无法打开日志文件", err)
		}
		writer = f
		enableColor = false
	default:
		writer = os.Stdout
		enableColor = true
	}

	DefaultLogger = NewLogger(logLevel, format, writer, enableColor)
	return nil
}
