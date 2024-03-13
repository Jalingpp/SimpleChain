package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"simplechain/storage"
)

type Transaction struct {
	TxID    int    //交易ID
	Content []byte //交易内容(序列化后的Request)
	TxHash  []byte //交易哈希
	Sender  string //交易发送者
}

func NewTransaction(txid int, content []byte) *Transaction {
	hash := sha256.Sum256(content)
	//解析content，获得发送者
	r := new(storage.Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	tx := &Transaction{txid, content, hash[:], r.ClientAddr}
	return tx
}

func (tx *Transaction) SerializeTx() ([]byte, error) {
	jsonTx, err := json.Marshal(tx)
	if err != nil {
		fmt.Printf("SerializeTx error: %v\n", err)
		return nil, err
	}
	return jsonTx, nil
}

func DeserializeTx(data []byte) (*Transaction, error) {
	tx := &Transaction{}
	err := json.Unmarshal(data, tx)
	if err != nil {
		fmt.Printf("DeserializeTx error: %v\n", err)
		return nil, err
	}
	return tx, nil
}
