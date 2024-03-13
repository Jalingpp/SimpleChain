package nodes

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"simplechain/blockchain"
	"simplechain/consensus"
	"simplechain/network"
	"simplechain/storage"
	"simplechain/utils"
	"strconv"
	"sync"
	"time"
)

type Fullnode struct {
	NodeID     string //节点ID
	Addr       string //节点网络监听地址
	RsaPrivKey []byte //RSA私钥
	RsaPubKey  []byte //RSA公钥

	MessagePool [][]byte   //接收客户端请求的消息池（队列）
	mpmutex     sync.Mutex //消息池的互斥锁

	P2P  *network.P2P    //当前节点所在的P2P网络
	Pbft *consensus.Pbft //当前节点的共识协议是pbft

	BatchSize    int                    //打包区块的大小上限
	Blockchain   *blockchain.Blockchain //当前节点维护的区块链
	packedNumber int                    //已经打包的区块数量
}

func NewFullnode(nodeID string, addr string, p2p *network.P2P, batchsize int) *Fullnode {
	priv, pub := utils.GetKeyPair()                         //生成rsa公私钥
	messagepool := make([][]byte, 0)                        //创建空消息池
	p2p.AddFullNode(nodeID, addr)                           //将当前节点注册入P2P网络
	p2p.AddPubKey(nodeID, pub)                              //将当前节点的公钥写入P2P网络
	pbft := consensus.NewPBFT(nodeID, addr, priv, pub, p2p) //创建共识协议
	fullnode := &Fullnode{nodeID, addr, priv, pub, messagepool, sync.Mutex{}, p2p, pbft, batchsize, blockchain.NewBlockchain(), 0}
	go fullnode.CreateFullNodeP2PListen() //启动网络监听
	go fullnode.RunConsensus()            //开启共识
	return fullnode
}

func (fullnode *Fullnode) GetNodeID() string {
	return fullnode.NodeID
}

func (fullnode *Fullnode) GetAddress() string {
	return fullnode.Addr
}

// 为全节点创建监听器并持续监听处理消息
func (fullnode *Fullnode) CreateFullNodeP2PListen() {
	logFile, err := os.OpenFile("./logout/"+fullnode.NodeID+"_log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("open log file failed, err:", err)
	}
	defer logFile.Close()
	// 创建日志对象
	fullnode.Pbft.Loger = log.New(logFile, "", log.Lshortfile)
	listen, err := net.Listen("tcp", fullnode.GetAddress())
	if err != nil {
		log.Panic(err)
	}
	// fmt.Printf("全节点%s开启P2P监听,地址：%s\n", fullnode.NodeID, fullnode.Addr)
	fullnode.Pbft.Loger.Println("全节点", fullnode.NodeID, "开启P2P监听,地址：", fullnode.Addr)
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
		//主节点处理客户端请求,非主节点交给pbft处理
		if fullnode.NodeID == fullnode.P2P.GetPrimaryID() {
			//解析消息
			cmd, _ := storage.SplitMessage(b)
			//如果cmd是request,则放入消息池
			if cmd == string(storage.CRequest) {
				fullnode.HandleRequest(b)
			} else {
				fullnode.Pbft.HandleRequest(b)
			}
		} else {
			fullnode.Pbft.HandleRequest(b)
		}
	}
}

// 处理接收到的请求
func (fullnode *Fullnode) HandleRequest(b []byte) {
	logFile, err := os.OpenFile("./logout/"+fullnode.NodeID+"_log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("open log file failed, err:", err)
	}
	defer logFile.Close()
	// 创建日志对象
	fullnode.Pbft.Loger = log.New(logFile, "", log.Lshortfile)
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	// fmt.Println(currentTime, fullnode.GetNodeID()+" recieves", string(b))
	fullnode.Pbft.Loger.Println(currentTime, fullnode.GetNodeID(), "recieves", string(b))
	//将接收到的消息放入消息池
	fullnode.mpmutex.Lock()
	fullnode.MessagePool = append(fullnode.MessagePool, b)
	fullnode.mpmutex.Unlock()
}

// 主节点启动共识例程
func (fullnode *Fullnode) RunConsensus() {
	go fullnode.AddToChain() //异步将区块上链
	for {
		//如果是主节点,则打包区块
		if fullnode.NodeID == fullnode.P2P.GetPrimaryID() {
			var hash []byte
			if fullnode.Blockchain.GetLastBlock() == nil {
				hash = []byte{}
			} else {
				hash = fullnode.Blockchain.GetLastBlock().Hash
			}
			newblock := fullnode.PackBlock(fullnode.packedNumber, hash)
			// fmt.Println("主节点打包区块")
			// fmt.Println("区块高度：", newblock.Height, ", 区块中交易数量：", len(newblock.Transactions))
			fullnode.packedNumber++
			//将区块转换为消息
			request := fullnode.BlockToRequest(newblock)
			//对刚打包的区块进行共识
			fullnode.Pbft.HandleRequest(request)
		}
	}
}

// 打包区块
func (fullnode *Fullnode) PackBlock(height int, prevhash []byte) *blockchain.Block {
	// 在循环中判断 MessagePool 是否为空
	for {
		if len(fullnode.MessagePool) != 0 {
			time.Sleep(time.Second)
			// 从消息池中取出消息
			fullnode.mpmutex.Lock()
			transactions := make([]*blockchain.Transaction, 0)
			for i := 0; i < fullnode.BatchSize; i++ {
				if len(fullnode.MessagePool) != 0 {
					message := fullnode.MessagePool[0]
					fullnode.MessagePool = fullnode.MessagePool[1:]
					// 解析消息
					_, content := storage.SplitMessage(message)
					// 将消息内容转换为交易
					tx := blockchain.NewTransaction(i, content)
					transactions = append(transactions, tx)
				} else {
					break
				}
			}
			fullnode.mpmutex.Unlock()
			return blockchain.NewBlock(height, prevhash, transactions)
		}
	}
}

// 一个同步线程：将共识后的区块上链
func (fullnode *Fullnode) AddToChain() {
	for {
		//判断是否有共识后的区块
		if fullnode.Pbft.SequenceIDL > fullnode.Blockchain.CurrentHeight {
			for i := fullnode.Blockchain.CurrentHeight; i < fullnode.Pbft.SequenceIDL; i++ {
				//将共识后的区块上链
				fullnode.Pbft.Loger.Println("节点", fullnode.NodeID, "将共识后的区块", i, "上链")
				request := fullnode.Pbft.MessagePool[fullnode.Pbft.MessageToCommit[i].Digest]
				block := fullnode.RequestToBlock(request)
				//回复block中的所有客户端
				fullnode.ReplyClient(block)
				blockHeight := fullnode.Blockchain.AddBlock(block)
				fullnode.Pbft.Loger.Println("节点", fullnode.NodeID, "共识后的区块", i, "上链成功,当前区块链高度为", blockHeight)
				// fmt.Println("节点", fullnode.NodeID, "共识后的区块", i, "上链成功,当前区块链高度为", blockHeight)
				// fullnode.PrintBlockInfor(i)
			}
		}
	}
}

// 将区块转换为Request消息（request序列化后再添加消息类别）
func (fullnode *Fullnode) BlockToRequest(block *blockchain.Block) []byte {
	//将区块序列化
	seblock, _ := block.SerializeBlock()
	r := new(storage.Request)
	r.Timestamp = time.Now().UnixNano()
	r.ClientAddr = "" //总的区块不包含客户端，每个交易包含客户端
	r.Message.ID = block.Height
	//消息内容就是用户的输入
	r.Message.Content = seblock
	//将请求序列化
	serequest, err := json.Marshal(r)
	if err != nil {
		log.Panic(err)
	}
	//添加消息类别
	request := storage.JointMessage(storage.CRequest, serequest)
	return request
}

// 将Request消息转换为区块
func (fullnode *Fullnode) RequestToBlock(request *storage.Request) *blockchain.Block {
	seblock := request.Content
	block, _ := blockchain.DeserializeBlock(seblock)
	return block
}

// 输出区块信息
func (fullnode *Fullnode) PrintBlockInfor(blockNo int) {
	block := fullnode.Blockchain.GetBlockByHeight(blockNo)
	if block == nil {
		fmt.Println("区块不存在")
		return
	}
	fmt.Println("区块高度：", block.Height, ", 区块中交易数量：", len(block.Transactions))
}

// 回复客户端
func (fullnode *Fullnode) ReplyClient(block *blockchain.Block) {
	for i := 0; i < len(block.Transactions); i++ {
		tx := block.Transactions[i]
		//回复客户端
		info := fullnode.NodeID + "节点已将msgid:" + strconv.Itoa(tx.TxID) + "存入区块" + strconv.Itoa(block.Height) + "中,消息内容为：" + string(tx.Content)
		fullnode.Pbft.Loger.Println("节点", fullnode.NodeID, "正在reply客户端")
		fullnode.P2P.SendRequest([]byte(info), tx.Sender)
		// fmt.Println("节点", fullnode.NodeID, "reply客户端完成:消息存入区块", block.Height, "中,消息内容为：", string(tx.Content))
	}
}
