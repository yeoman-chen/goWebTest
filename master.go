package main

import (
	"fmt"
	"log"
	"net"
	//"net/http"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

var (
	wg sync.WaitGroup = sync.WaitGroup{} // 等待各个socket连接处理
)
var ip string
var port string
var slaveCount int
var slaves []*slave

type slave struct {
	UserId     string
	SBCNum     int           // 并发连接数
	QPSNum     int           // 总请求次数
	RTNum      time.Duration //响应时间
	RTTNum     time.Duration // 平均响应时间
	SecNum     int           // 时间
	SuccessNum int           // 成功次数
	FailNum    int           // 失败次数
	Url        string
	conn       net.Conn
}

func init() {
	if len(os.Args) != 4 {
		log.Fatal("请输入 ip地址 port 从服务器数量")
	}
	ip = os.Args[1]
	port = os.Args[2]
	slaveCount, _ = strconv.Atoi(os.Args[3])
}

func main() {
	stop_chan := make(chan os.Signal) // 接收系统中断信号
	signal.Notify(stop_chan, os.Interrupt)
	stopNum := 0
	slaveNum := 0
	// 监听端口
	listen, err := net.Listen("tcp", ip+":"+port)
	if err != nil {
		log.Fatal(err)
	}
	//defer listen.Close()
	buf := make([]byte, 128)
	fmt.Println("Running...\n")
	// 每2秒显示运行状态
	go func() {
		for range time.Tick(2 * time.Second) {
			show(slaves)
		}
	}()
	// 检测slave 断开的状态
	go func() {
		<-stop_chan
		//Warn("Get Stop Command. Now Stoping...")
		stopNum++
		if stopNum >= slaveNum {
			if err = listen.Close(); err != nil {
				fmt.Println("Close error")
				log.Println(err)
			}
		}
	}()

	// 获取客户端数据
	for i := 0; i < slaveCount; i++ {
		fmt.Println("6666=====")
		conn, err := listen.Accept()
		fmt.Println("8888=====")
		if err != nil {
			fmt.Println("Accept error")
			log.Println(err)
			continue
		}
		//defer conn.Close()
		n, err := conn.Read(buf)
		tempC := slave{conn: conn, UserId: conn.RemoteAddr().String(), Url: string(buf[:n])}
		wg.Add(1)
		go tempC.run()
		slaves = append(slaves, &tempC)
		slaveNum++
		fmt.Println("=====" + strconv.Itoa(slaveNum))
	}
	wg.Wait() // 等待是否有未处理完socket处理
	time.Sleep(2 * time.Second)
}

// 接收从服务器发来数据
func (s *slave) run() {
	defer wg.Done()
	defer s.conn.Close()
	netStatus := "dail"
	var dv interface{}
	buf := make([]byte, 1024)
	for {
		n, err := s.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("client %s is close!\n", s.conn.RemoteAddr().String())
			}
			//在这里直接退出goroutine，关闭由defer操作完成
			return
		}
		err = json.Unmarshal(buf[:n], &dv)
		if err != nil {
			log.Println(err)
			continue
		}
		s.SBCNum = int(dv.(map[string]interface{})["SBCNum"].(float64))         // 并发连接数
		s.QPSNum = int(dv.(map[string]interface{})["QPSNum"].(float64))         // 总请求次数
		s.RTNum = time.Duration(dv.(map[string]interface{})["RTNum"].(float64)) // 响应时间
		s.SecNum = int(dv.(map[string]interface{})["SecNum"].(float64))
		s.SuccessNum = int(dv.(map[string]interface{})["SuccessNum"].(float64))
		s.FailNum = int(dv.(map[string]interface{})["FailNum"].(float64))
		s.Url = string(dv.(map[string]interface{})["Url"].(string))
		netStatus = string(dv.(map[string]interface{})["Url"].(string))
		if netStatus == "close" {
			return
		}
	}
}

func show(clients []*slave) {
	if len(clients) == 0 {
		return
	}
	total := slave{}
	num := 0
	for _, client := range clients {
		if client.SecNum == 0 {
			continue
		}
		num++
		fmt.Printf("用户id：%s,url: %s,并发数：%d,请求次数：%d,平均响应时间：%s,成功次数：%d,失败次数：%d\n",
			client.UserId,
			client.Url,
			client.SBCNum,
			client.SuccessNum+client.FailNum,
			client.RTNum/(time.Duration(client.SecNum)*time.Second),
			client.SuccessNum,
			client.FailNum)
		total.SBCNum += client.SBCNum
		total.RTNum += client.RTNum / (time.Duration(client.SecNum) * time.Second)
		total.SecNum += client.SecNum
		total.SuccessNum += client.SuccessNum
		total.FailNum += client.FailNum
	}
	if num == 0 {
		return
	}
	fmt.Printf("并发数：%d,请求次数：%d,平均响应时间：%s,成功次数：%d,失败次数：%d\n",
		total.SBCNum,
		total.SuccessNum+total.FailNum,
		total.RTNum/time.Duration(num),
		total.SuccessNum,
		total.FailNum)
	fmt.Println("\n")
}

// 心跳检测
func heartbeat(clients []slave) []slave {
	var tempc []slave
	for _, cleit := range clients {
		_, err := cleit.conn.Write([]byte(""))
		if err == nil {
			tempc = append(tempc, cleit)
		}
	}
	return tempc
}
