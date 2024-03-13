package nodes

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"simplechain/network"
	"simplechain/storage"
	"simplechain/utils"
	"time"
)

type Client struct {
	ClientID   string       //节点ID
	Addr       string       //节点网络监听地址
	RsaPrivKey []byte       //RSA私钥
	RsaPubKey  []byte       //RSA公钥
	P2P        *network.P2P //当前节点所在的P2P网络
}

func NewClient(clientID string, addr string, p2p *network.P2P) *Client {
	priv, pub := utils.GetKeyPair() //生成rsa公私钥
	p2p.AddClient(clientID, addr)   //将当前节点注册入P2P网络
	p2p.AddPubKey(clientID, pub)    //将当前节点的公钥写入P2P网络
	client := &Client{clientID, addr, priv, pub, p2p}
	go client.CreateClientP2PListen() //启动网络监听
	return client
}

func (client *Client) GetClientID() string {
	return client.ClientID
}

func (client *Client) GetAddress() string {
	return client.Addr
}

// 为全节点创建监听器并持续监听处理消息
func (client *Client) CreateClientP2PListen() {
	addr := client.GetAddress()
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("客户端%s开启P2P监听,地址：%s\n", client.ClientID, addr)
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err := io.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		client.HandleRequest(b)
	}
}

// 处理接收到的请求
func (client *Client) HandleRequest(b []byte) {
	// Code
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Println(currentTime, client.GetClientID(), "recieves:", string(b))
}

// 读取文件中的消息并依次发送给主节点
func (client *Client) SendRequestToPrimaryNode() {
	//读取文件中的消息
	// 打开文件
	file, err := os.Open("./data/testRequest_" + client.ClientID)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	// 创建一个带缓冲的读取器
	reader := bufio.NewScanner(file)

	// 逐行读取文件内容并输出
	for reader.Scan() {
		line := reader.Text() // 获取当前行的字符串
		r := new(storage.Request)
		r.Timestamp = time.Now().UnixNano()
		r.ClientAddr = client.Addr
		r.Message.ID = GetRandom()
		//消息内容就是用户的输入
		r.Message.Content = []byte(line)
		//将request序列化
		br, err := json.Marshal(r)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(br))
		//为序列化后的request添加消息类别
		content := storage.JointMessage(storage.CRequest, br)
		//发送给主节点
		client.P2P.SendRequest(content, client.P2P.NodeTable[client.P2P.GetPrimaryID()])
	}

	// 检查是否发生了读取错误
	if err := reader.Err(); err != nil {
		fmt.Println("Error:", err)
	}

}

// 返回一个十位数的随机数，作为msgid
func GetRandom() int {
	x := big.NewInt(10000000000)
	for {
		result, err := rand.Int(rand.Reader, x)
		if err != nil {
			log.Panic(err)
		}
		if result.Int64() > 1000000000 {
			return int(result.Int64())
		}
	}
}
