package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// 定义压测变量
// 从命令行接收参数
// 定时发送压测结果到主服务器
// 压测处理
// 发送压测结果

type master struct {
	ip   string
	port string
	conn net.Conn
}

var (
	SBCNum     int           // 并发连接数
	QPSNum     int           // 总请求次数
	RTNum      time.Duration // 响应时间
	RTTNum     time.Duration // 平均响应时间
	SuccessNum int           // 成功次数
	FailNum    int           // 失败次数
	SecNum     int

	err   error
	mt    master
	mu    sync.Mutex
	RQNum int    // 最大并发数，由命令行传入
	Url   string // 压测url 由命令行输入
)

func init() {
	if len(os.Args) != 5 {
		log.Fatal("%s 并发数 url ip port", os.Args[0])
	}
	RQNum, _ = strconv.Atoi(os.Args[1])
	Url = os.Args[2]
	mt.ip = os.Args[3]
	mt.port = os.Args[4]
}

func main() {
	// 连接master
	mt.conn, err = net.Dial("tcp", mt.ip+":"+mt.port)
	if err != nil {
		log.Fatal(err)
	}
	defer mt.conn.Close()

	fmt.Println("连接服务器成功...")

	// 每一秒发送数据给主服务器
	go func() {
		for range time.Tick(1 * time.Second) {
			sendToMaster(mt, map[string]interface{}{
				"NetStatus":  "dial",
				"Url":        Url,        // Url
				"SBCNum":     SBCNum,     // 并发连接数
				"QPSNum":     QPSNum,     // 总请求次数
				"RTNum":      RTNum,      // 响应时间
				"SecNum":     SecNum,     // 时间
				"SuccessNum": SuccessNum, // 成功次数
				"FailNum":    FailNum,    // 失败次数
			})
		}
	}()
	// 运行次数
	go func() {
		for range time.Tick(1 * time.Second) {
			SecNum++
		}
	}()
	// 执行请求
	testRequest()
	// 告诉主服务器断开
	sendToMaster(mt, map[string]interface{}{
		"NetStatus":  "close",
		"Url":        Url,        // Url
		"SBCNum":     SBCNum,     // 并发连接数
		"QPSNum":     QPSNum,     // 总请求次数
		"RTNum":      RTNum,      // 响应时间
		"SecNum":     SecNum,     // 时间
		"SuccessNum": SuccessNum, // 成功次数
		"FailNum":    FailNum,    // 失败次数
	})

	time.Sleep(1 * time.Second)
}

// 压力测试
func testRequest() {
	c := make(chan int)
	var tb time.Time
	var el time.Duration
	for i := 0; i < RQNum; i++ {
		SBCNum++
		fmt.Println(SBCNum)
		go func(url string, SBCNum int, RQNum int) {
			tb = time.Now()
			_, err := http.Get(url)
			if err == nil {
				el = time.Since(tb)
				mu.Lock()
				RTNum += el
				SuccessNum++
				mu.Unlock()
			} else {
				mu.Lock()
				FailNum++
				mu.Unlock()
			}
			time.Sleep(1 * time.Second)
			if SBCNum >= RQNum {
				c <- 1
			}
		}(Url, SBCNum, RQNum)
		time.Sleep(45 * time.Millisecond)

	}
	<-c // 阻塞
	fmt.Println("Finish")
	os.Exit(0)
}

// 发送数据到主服务器
func sendToMaster(mt master, data map[string]interface{}) {
	bdata, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}

	_, err = mt.conn.Write(bdata)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	} else {
		fmt.Println("send succuly")
	}
}
