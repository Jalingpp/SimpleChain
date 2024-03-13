package main

import (
	"bufio"
	"fmt"
	"os"
	"simplechain/network"
	"simplechain/nodes"
	"strings"
)

func main() {

	batchsize := 10

	//创建节点和客户端
	_, clientList := InitClientAndNodes(InitP2P(), batchsize)

	//客户端读取消息并发给主节点
	for _, v := range clientList {
		v.SendRequestToPrimaryNode()
	}

	// 保持程序运行
	select {}
}

// 初始化一个空的P2P网络
func InitP2P() *network.P2P {
	p2p := network.NewP2P("tcp")
	return p2p
}

// 初始化客户端和全节点
func InitClientAndNodes(p2p *network.P2P, batchsize int) (map[string]*nodes.Fullnode, map[string]*nodes.Client) {
	fullnodeList := make(map[string]*nodes.Fullnode)
	clientList := make(map[string]*nodes.Client)

	// 打开文件
	file, err := os.Open("config")
	if err != nil {
		fmt.Println("Error:", err)
		return nil, nil
	}
	defer file.Close()

	// 创建一个带缓冲的读取器
	reader := bufio.NewScanner(file)

	// 逐行读取文件内容并输出
	for reader.Scan() {
		line := reader.Text() // 获取当前行的字符串
		lines := strings.Split(line, ",")
		if lines[0] == "fullnode" {
			fullnode := nodes.NewFullnode(lines[1], lines[2], p2p, batchsize)
			fullnodeList[lines[1]] = fullnode
			if p2p.PrimaryNodeID == "" {
				p2p.SetPrimaryNode(lines[1])
			}
		} else if lines[0] == "client" {
			client := nodes.NewClient(lines[1], lines[2], p2p)
			clientList[lines[1]] = client
		}
	}

	// 检查是否发生了读取错误
	if err := reader.Err(); err != nil {
		fmt.Println("Error:", err)
	}

	return fullnodeList, clientList
}
