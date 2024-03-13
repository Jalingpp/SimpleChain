package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

type Block struct {

	//header
	Height        int    //区块高度
	PrevBlockHash []byte //上一个区块的哈希
	Hash          []byte //当前区块的哈希
	Timestamp     string //时间戳
	TxMHTRoot     []byte //交易Merkle树根

	//body
	Transactions []*Transaction //交易列表
}

func NewBlock(height int, prevBlockHash []byte, transactions []*Transaction) *Block {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	//根据transactions构建默克尔树
	//将transactions转换为哈希值
	txhashes := make([][]byte, 0)
	for _, tx := range transactions {
		txhashes = append(txhashes, tx.TxHash)
	}
	//构建默克尔树
	txMHT := NewMerkleTree(txhashes)
	//计算当前区块的哈希值Hash(PrevBlockHash+Timestamp+TxMHTRoot)
	blockcontent := make([]byte, 0)
	blockcontent = append(blockcontent, prevBlockHash...)
	blockcontent = append(blockcontent, []byte(currentTime)...)
	blockcontent = append(blockcontent, txMHT.GetRootHash()...)
	hash := sha256.Sum256(blockcontent)
	block := &Block{height, prevBlockHash, hash[:], currentTime, txMHT.GetRootHash(), transactions}
	return block
}

type SeBlock struct {
	//header
	Height        int    //区块高度
	PrevBlockHash []byte //上一个区块的哈希
	Hash          []byte //当前区块的哈希
	Timestamp     string //时间戳
	TxMHTRoot     []byte //交易Merkle树根

	//body
	Transactions [][]byte //交易列表
}

func (block *Block) SerializeBlock() ([]byte, error) {
	//将区块序列化
	seblock := &SeBlock{block.Height, block.PrevBlockHash, block.Hash, block.Timestamp, block.TxMHTRoot, make([][]byte, 0)}
	for _, tx := range block.Transactions {
		setx, _ := tx.SerializeTx()
		seblock.Transactions = append(seblock.Transactions, setx)
	}
	jsonBlock, err := json.Marshal(seblock)
	if err != nil {
		fmt.Printf("SerializeBlock error: %v\n", err)
		return nil, err
	}
	return jsonBlock, nil
}

func DeserializeBlock(data []byte) (*Block, error) {
	var seblock SeBlock
	if err := json.Unmarshal(data, &seblock); err != nil {
		fmt.Printf("DeserializeBlock error: %v\n", err)
		return nil, err
	}
	block := &Block{seblock.Height, seblock.PrevBlockHash, seblock.Hash, seblock.Timestamp, seblock.TxMHTRoot, make([]*Transaction, 0)}
	for i := 0; i < len(seblock.Transactions); i++ {
		transaction, _ := DeserializeTx(seblock.Transactions[i])
		block.Transactions = append(block.Transactions, transaction)
	}
	return block, nil
}
