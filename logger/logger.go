package logger

import (
	"os"
	"sync"
	"time"
)

var mutex sync.Mutex

type Logger struct {
	fileName string   // 日志文件
	mode     string   // 日志记录形式：cover覆盖，append追加
	logArr   []string // 日志内容数组
}

func NewLogger(fname string, mode string) *Logger {
	return &Logger{fileName: fname, mode: mode}
}

// 记录日志
func (this *Logger) Record(content string, category string, level string, autoFlush bool) *Logger {
	contentStr := this.messageFormat(content, category, level, time.Now())
	this.logArr = append(this.logArr, contentStr)
	if autoFlush == true {
		this.WriteFile()
	}
	return this
}

// 日志格式
func (this *Logger) messageFormat(content string, category string, level string, logTime time.Time) string {
	recordTime := logTime.Format("2006-01-02 15:04:05")
	msg := "[" + recordTime + "]" + "[" + category + "]" + "[" + level + "]" + content
	return msg
}

// 刷新日志(写入文件)
func (this *Logger) WriteFile() {
	var logText string
	var fileMode int
	for _, val := range this.logArr {
		logText += val + "\n"
	}
	if this.mode == "cover" {
		fileMode = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	} else {
		fileMode = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}
	op, err := os.OpenFile(this.fileName, fileMode, 0644)
	defer op.Close() //延迟关闭资源
	if err != nil {
		panic(err)
	}
	mutex.Lock()
	op.WriteString(logText)
	mutex.Unlock()

	// 情况数组
	//this.logArr = append(this.logArr[:0],this.logArr[len(this.logArr):])

}

// 删除文件
func (this *Logger) Remove() {
	os.Truncate(this.fileName, 0)
}

// 获取当前时间的日期
func (this *Logger) getCurTimeDate() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
