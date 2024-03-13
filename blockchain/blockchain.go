package blockchain

type Blockchain struct {
	CurrentHeight int
	Chain         []*Block
}

func NewBlockchain() *Blockchain {
	chain := make([]*Block, 0)
	return &Blockchain{0, chain}
}

// 获取区块链最新的区块
func (blockchain *Blockchain) GetLastBlock() *Block {
	if len(blockchain.Chain) == 0 {
		return nil
	}
	return blockchain.Chain[len(blockchain.Chain)-1]
}

// 添加区块,返回最新区块高度
func (blockchain *Blockchain) AddBlock(block *Block) int {
	blockchain.Chain = append(blockchain.Chain, block)
	blockchain.CurrentHeight++
	return blockchain.CurrentHeight
}

// 根据块高获取区块指针，创世区块块高为0
func (blockchain *Blockchain) GetBlockByHeight(height int) *Block {
	return blockchain.Chain[height]
}
