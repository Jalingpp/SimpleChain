package main

// func main() {

// 	//创建节点和客户端
// 	fullnodeList, clientList := InitClientAndNodes(InitP2P())

// 	// 全节点之间互相广播消息
// 	for k, v := range fullnodeList {
// 		message := []byte("Hello, I'm " + v.GetNodeID() + ". My address is " + v.GetAddress())
// 		v.P2P.Broadcast(k, message)
// 	}

// 	// 客户端向四个全节点发送消息
// 	for _, v := range clientList {
// 		clientMessage := []byte("Hello, I'm " + v.GetClientID() + ". My address is " + v.GetAddress())
// 		for _, fn := range fullnodeList {
// 			v.P2P.SendRequest(clientMessage, fn.GetAddress())
// 		}
// 	}

// 	// 保持程序运行
// 	select {}
// }
