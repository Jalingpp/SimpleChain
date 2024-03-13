package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type Transaction struct {
	TxID    int    //交易ID
	Content []byte //交易内容
	TxHash  []byte //交易哈希
}

func NewTransaction(txid int, content []byte) *Transaction {
	hash := sha256.Sum256(content)
	tx := &Transaction{txid, content, hash[:]}
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
