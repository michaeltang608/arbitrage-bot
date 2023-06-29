package log

import (
	"fmt"
	"log"
	"os"
)

var (
	format = log.Lshortfile | log.Ltime | log.Ldate
)

type Logger struct {
	logInfo  *log.Logger
	logDebug *log.Logger
	logError *log.Logger
}

func NewLog(prefix string) *Logger {
	return &Logger{
		logInfo:  log.New(os.Stdout, fmt.Sprintf("[Info] - [%s]- ", prefix), format),
		logDebug: log.New(os.Stdout, fmt.Sprintf("[Debug] - [%s]- ", prefix), format),
		logError: log.New(os.Stdout, fmt.Sprintf("[Error] - [%s]- ", prefix), format),
	}
}

func (l *Logger) Debug(format string, v ...any) {

	_ = l.logDebug.Output(2, fmt.Sprintf(format, v...))

}
func (l *Logger) Info(format string, v ...any) {
	_ = l.logInfo.Output(2, fmt.Sprintf(format, v...))
}

// Panic 打印error日志，然后再panic, 不用发送飞书，会有recover统一处理
func (l *Logger) Panic(v ...any) {
	msg := fmt.Sprint(v...)
	_ = l.logError.Output(2, msg)
	panic(msg)
}

func (l *Logger) Error(format string, v ...any) {
	_ = l.logError.Output(2, fmt.Sprintf(format, v...))
}
