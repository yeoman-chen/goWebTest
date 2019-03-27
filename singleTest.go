package main

/**
 * 单机测试
 * 压测几个关键参数为：并发连接数、总请求次数、影响时间、平均响应时间、成功次数、失败次数、
 * 日志记录、时间段统计分析等
 */
import (
	"fmt"
	"github.com/yeoman-chen/goCurl"
	"goWebTest/logger"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	LOG_FILE = "D:\\www\\GOPATH\\src\\goWebTest\\app.log"
)

var (
	BeginTime      time.Time         // 开始执行时间
	EndTime        time.Time         // 结束时间
	SecNum         int               // 秒数
	RQNum          int               // 最大运行次数，由命令行输入
	Url            string            // 压测URL,由命令行输入
	UserNum        int               // 用户数量，由命令行输入
	RunNum         int               // 运行次数,控制协程退出
	HttpMethod     string            // 请求方法
	Params         map[string]string // 参数
	ReqDoneTimeArr []string          // 执行时间数组
)

// 响应结构体
type respData struct {
	StatusCode    int   // 响应状态码
	ContentLength int64 // 内容长度
}

var users []User

// User结构体
type User struct {
	UserId     int           // 用户ID
	SBCNum     int           // 并发数
	QPSNum     int           // 总请求次数
	RTNum      time.Duration // 响应时间
	RTTNum     time.Duration // 平均响应时间
	SuccessNum int           // 成功次数
	FailNum    int           // 失败次数
	mu         sync.Mutex    // 锁
}

func init() {
	if len(os.Args) == 1 || len(os.Args) < 5 || os.Args[1] == "--help" {
		log.Fatal(" --help 帮助\n -c 并发数\n -n 总请求数\n Url 请求的Url\n -m 请求方式\n -p 参数\n Example: go run singleTest.go -c 10 -n 100 http://www.baidu.com/ -m get -p a=1&b=2")
	}

	UserNum, _ = strconv.Atoi(os.Args[2])
	RQNum, _ = strconv.Atoi(os.Args[4])
	Url = os.Args[5]
	if len(os.Args) >= 7 {
		HttpMethod = strings.ToLower(os.Args[7])
	} else {
		HttpMethod = "get"
	}
	Params := make(map[string]string)
	if len(os.Args) >= 9 {
		param := strings.Replace(os.Args[9], "\"", "", -1)
		strArr := strings.Split(param, "&")
		for _, str := range strArr {
			data := strings.Split(str, "=")
			if len(data) == 2 {
				Params[data[0]] = data[1]
			}
		}
	}
	users = make([]User, UserNum)
	RunNum = 0
}

func main() {
	fmt.Println("Start Running ...")
	// 清空日志内容
	logger.NewLogger(LOG_FILE, "cover").Remove()
	c := make(chan int)
	go func(c chan int) {
		for range time.Tick(1 * time.Second) {
			SecNum += 1
			showAll(users)
			if RunNum >= RQNum {
				break
			}
		}
		c <- 1
	}(c)
	// 执行请求
	testRequest()
	// 阻塞等待
	<-c
	// 总结
	resultSummary()

	fmt.Println("End Running ...")

}

// 请求测试
func testRequest() {
	BeginTime = time.Now()
	temp := 0
	for i := 0; i < UserNum; i++ {
		if RQNum%UserNum != 0 && i < RQNum%UserNum {
			temp = 1
		} else {
			temp = 0
		}
		users[i].UserId = i
		users[i].QPSNum = RQNum/UserNum + temp
		go users[i].request(HttpMethod, Url, Params)
		time.Sleep(45 * time.Millisecond)
	}
	EndTime = time.Now()
}

// 显示所有的信息
func showAll(us []User) {
	uLen := len(us)
	var SBCNum int            // 并发连接数
	var QPSNum float64 = 0.00 // 每秒请求数
	var RTNum time.Duration   // 响应时间
	var RTTNum float64        // 平均响应时间
	var SuccessNum int        // 成功次数
	var FailNum int           // 失败次数

	for i := 0; i < uLen; i++ {
		SBCNum += us[i].SBCNum
		RTNum += us[i].RTNum
		SuccessNum += us[i].SuccessNum
		FailNum += us[i].FailNum
		//us[i].show()
		reqNum, _ := strconv.ParseFloat(strconv.Itoa(us[i].SBCNum), 64)
		// float to string
		qpsNum, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", reqNum/us[i].RTNum.Seconds()), 64)
		QPSNum += qpsNum
	}
	// fmt.Println(RTNum)
	tepRTNum, _ := strconv.ParseFloat(strconv.Itoa(SuccessNum+FailNum), 64)
	RTTNum = RTNum.Seconds() / tepRTNum
	fmt.Printf("并发数：%d,请求次数：%d, 每秒请求次数：%f,平均响应时间：%fs,成功次数：%d,失败次数：%d\n",
		uLen, SuccessNum+FailNum, QPSNum, RTTNum, SuccessNum, FailNum)
}

func (u *User) request(httpMethod, url string, params map[string]string) {
	var tb time.Time
	var el time.Duration

	for i := 0; i < u.QPSNum; i++ {
		u.SBCNum++
		go func(u *User) {
			tb = time.Now()
			var resp respData
			resp, err := doneRequest(httpMethod, url, params)
			if err == nil && resp.StatusCode == 200 {
				el = time.Since(tb)
				msg := url + " " + httpMethod + " " + strconv.Itoa(resp.StatusCode) + " " + fmt.Sprintf("%s", el)
				logObj := logger.NewLogger(LOG_FILE, "append")
				logObj.Record(msg, "application", "info", true)
				u.mu.Lock()
				ReqDoneTimeArr = append(ReqDoneTimeArr, fmt.Sprintf("%s", el))
				u.SuccessNum++
				u.RTNum += el
				RunNum++
				u.mu.Unlock()
			} else {
				u.mu.Lock()
				u.FailNum++
				RunNum++
				u.mu.Unlock()
			}
		}(u)
		time.Sleep(45 * time.Microsecond)
	}
}

// 执行请求
func doneRequest(method, url string, queries map[string]string) (respData, error) {
	respData := respData{StatusCode: 200, ContentLength: 0}
	req := goCurl.NewRequest()
	resp, err := req.SetQueries(queries).Post(url)
	if err != nil {
		return respData, err
	}
	respData.StatusCode = resp.StatusCode
	return respData, nil
}

// 结果汇总，打印每个阶段完成情况
func resultSummary() {

	sort.Sort(sort.StringSlice(ReqDoneTimeArr))
	for _, val := range ReqDoneTimeArr {
		fmt.Println(val)
	}
	total := len(ReqDoneTimeArr)
	perArr := [...]int{50, 60, 70, 80, 90, 95, 99}
	fmt.Println(" ")
	fmt.Println("在特定时间内服务完成请求的百分比:")

	var arrIndex int
	for _, val := range perArr {
		arrIndex = total * val / 100
		varStr := strconv.Itoa(val)
		fmt.Println(varStr + "%  " + ReqDoneTimeArr[arrIndex])
	}
	fmt.Println(" ")
}

// 显示单个线程状态
func (u *User) show() {
	fmt.Printf("用户id：%d,并发数：%d,请求次数：%d,平均响应时间：%s,成功次数：%d,失败次数：%d\n",
		u.UserId, u.SBCNum, u.SuccessNum+u.FailNum, u.RTNum/(time.Duration(SecNum)*time.Second), u.SuccessNum, u.FailNum)
}
