package consensus

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"simplechain/network"
	"simplechain/storage"
	"simplechain/utils"
	"strconv"
	"sync"
)

type Pbft struct {
	NodeID     string //节点ID
	Addr       string //节点网络监听地址
	RsaPrivKey []byte //RSA私钥
	RsaPubKey  []byte //RSA公钥

	P2P         *network.P2P //一个P2P网络
	SequenceIDL int          //当前已提交消息的自增序号(低水位线)

	Lock sync.Mutex //锁
	//临时消息池，消息摘要对应消息本体
	MessagePool map[string]*storage.Request
	//存放收到的prepare数量(至少需要收到并确认2f个)，根据摘要来对应
	PrePareConfirmCount map[string]map[string]bool
	//存放收到的commit数量（至少需要收到并确认2f+1个），根据摘要来对应
	CommitConfirmCount map[string]map[string]bool
	//该笔消息是否已进行Commit广播
	IsCommitBordcast map[string]bool
	//该笔消息是否已对客户端进行Reply
	IsReply map[string]bool
	//暂存完成共识待commit的消息
	MessageToCommit map[int]storage.Commit

	MessageCommitted []storage.Message //本地消息池（模拟持久化层），只有确认提交成功后才会存入此池
	Loger            *log.Logger       //日志对象
}

func NewPBFT(nodeID string, addr string, privkey []byte, pubkey []byte, p2p *network.P2P) *Pbft {
	p := new(Pbft)
	p.NodeID = nodeID
	p.Addr = addr
	p.RsaPrivKey = privkey
	p.RsaPubKey = pubkey
	p.P2P = p2p
	p.SequenceIDL = 0
	p.MessagePool = make(map[string]*storage.Request)
	p.PrePareConfirmCount = make(map[string]map[string]bool)
	p.CommitConfirmCount = make(map[string]map[string]bool)
	p.IsCommitBordcast = make(map[string]bool)
	p.IsReply = make(map[string]bool)
	p.MessageCommitted = make([]storage.Message, 0)
	p.MessageToCommit = make(map[int]storage.Commit)
	return p
}

func (p *Pbft) HandleRequest(data []byte) {
	//切割消息，根据消息命令调用不同的功能
	cmd, content := storage.SplitMessage(data) //拆解消息类别，content是序列化后的Request、PrePrepare、Prepare或Commit
	switch storage.Command(cmd) {
	case storage.CRequest:
		p.HandleClientRequest(content)
	case storage.CPrePrepare:
		p.HandlePrePrepare(content)
	case storage.CPrepare:
		p.HandlePrepare(content)
	case storage.CCommit:
		p.HandleCommit(content)
	}
}

// 处理客户端发来的请求
func (p *Pbft) HandleClientRequest(content []byte) {
	logFile, logerr := os.OpenFile("./logout/"+p.NodeID+"_log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if logerr != nil {
		fmt.Println("open log file failed, err:", logerr)
	}
	defer logFile.Close()
	// 创建日志对象
	p.Loger = log.New(logFile, "", log.Lshortfile)

	// fmt.Println("节点", p.NodeID, "已接收到客户端发来的request")
	p.Loger.Println("节点", p.NodeID, "已接收到客户端发来的request")
	//使用json解析出Request结构体（反序列化得到request）
	r := new(storage.Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	//获取消息摘要
	digest := storage.GetDigest(*r)
	// fmt.Println("节点", p.NodeID, "已将request存入临时消息池")
	p.Loger.Println("节点", p.NodeID, "已将request存入临时消息池")
	//存入临时消息池
	p.MessagePool[digest] = r
	//主节点对消息摘要进行签名
	digestByte, _ := hex.DecodeString(digest)
	signInfo := utils.RsaSignWithSha256(digestByte, p.RsaPrivKey)
	//拼接成PrePrepare，准备发往follower节点
	pp := storage.PrePrepare{RequestMessage: *r, Digest: digest, SequenceID: r.ID, Sign: signInfo}
	//将PrePrepare序列化
	b, err := json.Marshal(pp)
	if err != nil {
		log.Panic(err)
	}
	// fmt.Println("节点", p.NodeID, "正在向其他节点进行进行PrePrepare广播")
	p.Loger.Println("节点", p.NodeID, "正在向其他节点进行进行PrePrepare广播")
	//给序列化后的PrePrepare消息添加消息类别
	message := storage.JointMessage(storage.CPrePrepare, b)
	//进行PrePrepare广播
	p.P2P.Broadcast(p.NodeID, message)
	// fmt.Println("节点", p.NodeID, " PrePrepare广播完成")
	p.Loger.Println("节点", p.NodeID, " PrePrepare广播完成")
}

// 序号累加
func (p *Pbft) SequenceIDLAdd() {
	p.Lock.Lock()
	p.SequenceIDL++
	p.Lock.Unlock()
}

// 处理预准备消息
func (p *Pbft) HandlePrePrepare(content []byte) {
	logFile, logerr := os.OpenFile("./logout/"+p.NodeID+"_log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if logerr != nil {
		fmt.Println("open log file failed, err:", logerr)
	}
	defer logFile.Close()
	// 创建日志对象
	p.Loger = log.New(logFile, "", log.Lshortfile)

	// fmt.Println("节点", p.NodeID, "已接收到主节点发来的PrePrepare")
	p.Loger.Println("节点", p.NodeID, "已接收到主节点发来的PrePrepare")
	// 反序列化得到PrePrepare结构体
	pp := new(storage.PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}
	//获取主节点的公钥，用于数字签名验证
	primaryNodePubKey := p.P2P.GetPrimaryPubkey()
	digestByte, _ := hex.DecodeString(pp.Digest)
	if digest := storage.GetDigest(pp.RequestMessage); digest != pp.Digest {
		// fmt.Println("信息摘要对不上,拒绝进行prepare广播")
		p.Loger.Println("信息摘要对不上,拒绝进行prepare广播")
	} else if !utils.RsaVerySignWithSha256(digestByte, pp.Sign, primaryNodePubKey) {
		// fmt.Println("主节点签名验证失败,拒绝进行prepare广播")
		p.Loger.Println("主节点签名验证失败,拒绝进行prepare广播")
	} else {
		//将信息存入临时消息池
		// fmt.Println("节点", p.NodeID, "已将消息存入临时节点池")
		p.Loger.Println("节点", p.NodeID, "已将消息存入临时节点池")
		p.MessagePool[pp.Digest] = &pp.RequestMessage
		//节点使用私钥对其签名
		sign := utils.RsaSignWithSha256(digestByte, p.RsaPrivKey)
		//拼接成Prepare
		pre := storage.Prepare{Digest: pp.Digest, SequenceID: pp.SequenceID, NodeID: p.NodeID, Sign: sign}
		//将Prepare序列化
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Panic(err)
		}
		//进行准备阶段的广播
		// fmt.Println("节点", p.NodeID, "正在进行Prepare广播")
		p.Loger.Println("节点", p.NodeID, "正在进行Prepare广播")
		//为序列化后的Prepare消息添加消息类别
		p.P2P.Broadcast(p.NodeID, storage.JointMessage(storage.CPrepare, bPre))
		// fmt.Println("节点", p.NodeID, " Prepare广播完成")
		p.Loger.Println("节点", p.NodeID, " Prepare广播完成")
	}
}

// 处理准备消息
func (p *Pbft) HandlePrepare(content []byte) {
	logFile, logerr := os.OpenFile("./logout/"+p.NodeID+"_log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if logerr != nil {
		fmt.Println("open log file failed, err:", logerr)
	}
	defer logFile.Close()
	// 创建日志对象
	p.Loger = log.New(logFile, "", log.Lshortfile)

	//反序列化得到Prepare结构体
	pre := new(storage.Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Panic(err)
	}
	// fmt.Println("节点", p.NodeID, "已接收到节点", pre.NodeID, "发来的Prepare")
	p.Loger.Println("节点", p.NodeID, "已接收到节点", pre.NodeID, "发来的Prepare")
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.P2P.GetNodePubkey(pre.NodeID)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if _, ok := p.MessagePool[pre.Digest]; !ok {
		// fmt.Println("当前临时消息池无此摘要,拒绝执行commit广播")
		p.Loger.Println("当前临时消息池无此摘要,拒绝执行commit广播")
	} else if !utils.RsaVerySignWithSha256(digestByte, pre.Sign, MessageNodePubKey) {
		// fmt.Println("节点签名验证失败,拒绝执行commit广播")
		p.Loger.Println("节点签名验证失败,拒绝执行commit广播")
	} else {
		p.SetPrePareConfirmMap(pre.Digest, pre.NodeID, true)
		count := 0
		for range p.PrePareConfirmCount[pre.Digest] {
			count++
		}
		//因为主节点不会发送Prepare，所以不包含自己
		specifiedCount := 0
		if p.NodeID == p.P2P.GetPrimaryID() {
			specifiedCount = len(p.P2P.NodeTable) / 3 * 2
		} else {
			specifiedCount = (len(p.P2P.NodeTable) / 3 * 2) - 1
		}
		//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
		//获取消息源节点的公钥，用于数字签名验证
		if count >= specifiedCount && !p.IsCommitBordcast[pre.Digest] {
			// fmt.Println("节点", p.NodeID, "已收到至少2f个节点(包括本地节点)发来的Prepare信息")
			p.Loger.Println("节点", p.NodeID, "已收到至少2f个节点(包括本地节点)发来的Prepare信息")
			//节点使用私钥对其签名
			sign := utils.RsaSignWithSha256(digestByte, p.RsaPrivKey)
			//构建Commit结构体
			c := storage.Commit{Digest: pre.Digest, SequenceID: pre.SequenceID, NodeID: p.NodeID, Sign: sign}
			//将Commit序列化
			bc, err := json.Marshal(c)
			if err != nil {
				log.Panic(err)
			}
			//进行提交信息的广播
			// fmt.Println("节点", p.NodeID, "正在进行commit广播")
			p.Loger.Println("节点", p.NodeID, "正在进行commit广播")
			//将序列化后的Commit添加消息类别后广播
			p.P2P.Broadcast(p.NodeID, storage.JointMessage(storage.CCommit, bc))
			p.IsCommitBordcast[pre.Digest] = true
			// fmt.Println("节点", p.NodeID, "commit广播完成")
			p.Loger.Println("节点", p.NodeID, "commit广播完成")
		}
	}
}

// 为多重映射开辟赋值
func (p *Pbft) SetPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.PrePareConfirmCount[val]; !ok {
		p.PrePareConfirmCount[val] = make(map[string]bool)
	}
	p.PrePareConfirmCount[val][val2] = b
}

// 处理提交确认消息
func (p *Pbft) HandleCommit(content []byte) {
	logFile, logerr := os.OpenFile("./logout/"+p.NodeID+"_log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if logerr != nil {
		fmt.Println("open log file failed, err:", logerr)
	}
	defer logFile.Close()
	// 创建日志对象
	p.Loger = log.New(logFile, "", log.Lshortfile)

	//反序列化得到Commit结构体
	c := new(storage.Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	// fmt.Println("节点", p.NodeID, "已接收到节点", c.NodeID, "发来的Commit")
	p.Loger.Println("节点", p.NodeID, "已接收到节点", c.NodeID, "发来的Commit")
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.P2P.GetNodePubkey(c.NodeID)
	digestByte, _ := hex.DecodeString(c.Digest)
	if _, ok := p.PrePareConfirmCount[c.Digest]; !ok {
		// fmt.Println("当前prepare池无此摘要,拒绝将信息持久化到本地消息池")
		p.Loger.Println("当前prepare池无此摘要,拒绝将信息持久化到本地消息池")
	} else if !utils.RsaVerySignWithSha256(digestByte, c.Sign, MessageNodePubKey) {
		// fmt.Println("节点签名验证失败,拒绝将信息持久化到本地消息池")
		p.Loger.Println("节点签名验证失败,拒绝将信息持久化到本地消息池")
	} else {
		p.SetCommitConfirmMap(c.Digest, c.NodeID, true)
		count := 0
		for range p.CommitConfirmCount[c.Digest] {
			count++
		}
		//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！
		if count >= len(p.P2P.NodeTable)/3*2 && !p.IsReply[c.Digest] && p.IsCommitBordcast[c.Digest] {
			// fmt.Println("节点", p.NodeID, "已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息")
			p.Loger.Println("节点", p.NodeID, "已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息")

			//判断是否是当前最小序号
			if c.SequenceID == p.SequenceIDL {
				//将消息放入待提交池中（用于上链时获取摘要）
				p.MessageToCommit[c.SequenceID] = *c
				//将消息信息，提交到本地消息池中！
				p.MessageCommitted = append(p.MessageCommitted, p.MessagePool[c.Digest].Message) //Message中包含区块高度和序列化后的区块（见Fullnode.go中的函数BlockToRequest）
				info := p.NodeID + "节点已将msgid:" + strconv.Itoa(p.MessagePool[c.Digest].ID) + "存入本地消息池中,消息内容为：" + string(p.MessagePool[c.Digest].Content)
				p.Loger.Println(info)
				//只将回复位置为true，不实际执行回复，实际回复在Fullnode中将区块拆解为交易后，依次回复每笔交易
				//p.Loger.Println("节点", p.NodeID, "正在reply客户端")
				//p.P2P.SendRequest([]byte(info), p.MessagePool[c.Digest].ClientAddr)
				p.IsReply[c.Digest] = true
				//p.Loger.Println("节点", p.NodeID, "reply完毕")
				p.SequenceIDLAdd()
				for {
					if _, ok := p.MessageToCommit[p.SequenceIDL]; ok {
						p.MessageCommitted = append(p.MessageCommitted, p.MessagePool[p.MessageToCommit[p.SequenceIDL].Digest].Message)
						info := p.NodeID + "节点已将msgid:" + strconv.Itoa(p.MessageToCommit[p.SequenceIDL].SequenceID) + "存入本地消息池中,消息内容为：" + string(p.MessagePool[p.MessageToCommit[p.SequenceIDL].Digest].Content)
						p.Loger.Println(info)
						//只将回复位置为true，不实际执行回复，实际回复在Fullnode中将区块拆解为交易后，依次回复每笔交易
						//p.Loger.Println("节点", p.NodeID, "正在reply客户端")
						//p.P2P.SendRequest([]byte(info), p.MessagePool[p.MessageToCommit[p.SequenceIDL].Digest].ClientAddr)
						p.IsReply[p.MessageToCommit[p.SequenceIDL].Digest] = true
						//p.Loger.Println("节点", p.NodeID, "reply完毕")
						p.SequenceIDLAdd()
					} else {
						break
					}
				}
			} else if c.SequenceID > p.SequenceIDL {
				//如果收到的消息序号大于当前最小序号，则将消息存入待commit消息池
				p.MessageToCommit[c.SequenceID] = *c
				// fmt.Println("节点", p.NodeID, "已将消息存入待commit消息池")
				p.Loger.Println("节点", p.NodeID, "已将消息", c.SequenceID, "存入待commit消息池")
			}
		}
	}
}

// 为多重映射开辟赋值
func (p *Pbft) SetCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.CommitConfirmCount[val]; !ok {
		p.CommitConfirmCount[val] = make(map[string]bool)
	}
	p.CommitConfirmCount[val][val2] = b
}
