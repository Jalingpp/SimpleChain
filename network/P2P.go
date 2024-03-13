package network

import (
	"log"
	"net"
)

type P2P struct {
	NodeTable     map[string]string //全节点地址列表
	ClientTable   map[string]string //客户端地址列表
	PubKeyTable   map[string][]byte //全节点和客户端公钥列表
	NetworkType   string            //网络类型
	PrimaryNodeID string            //主节点
}

func NewP2P(nettype string) *P2P {
	p2p := &P2P{make(map[string]string), make(map[string]string), make(map[string][]byte), nettype, ""}
	return p2p
}

func (p2p *P2P) SetPrimaryNode(nodeID string) {
	p2p.PrimaryNodeID = nodeID
}

func (p2p *P2P) AddFullNode(nodeID string, addr string) {
	p2p.NodeTable[nodeID] = addr
}

func (p2p *P2P) AddPubKey(nodeID string, pubkey []byte) {
	p2p.PubKeyTable[nodeID] = pubkey
}

func (p2p *P2P) AddClient(clientID string, addr string) {
	p2p.ClientTable[clientID] = addr
}

// 广播某个全节点fullnode的消息给列表里其他fullnode
func (p2p *P2P) Broadcast(nodeID string, context []byte) {
	for k, v := range p2p.NodeTable {
		if k != nodeID {
			p2p.SendRequest(context, v)
		}
	}
}

// 发送请求
func (p2p *P2P) SendRequest(context []byte, addr string) {
	conn, err := net.Dial(p2p.NetworkType, addr)
	if err != nil {
		log.Println("connect error", err)
		return
	}

	_, err = conn.Write(context)
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}

// 获取主节点ID
func (p2p *P2P) GetPrimaryID() string {
	if p2p.PrimaryNodeID == "" {
		return ""
	}
	return p2p.PrimaryNodeID
}

// 获取主节点公钥
func (p2p *P2P) GetPrimaryPubkey() []byte {
	if p2p.PrimaryNodeID == "" {
		return nil
	}
	return p2p.PubKeyTable[p2p.PrimaryNodeID]
}

// 获取某个节点的公钥
func (p2p *P2P) GetNodePubkey(nodeID string) []byte {
	return p2p.PubKeyTable[nodeID]
}
