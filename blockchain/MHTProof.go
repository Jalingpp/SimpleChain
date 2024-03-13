package blockchain

import "simplechain/utils"

type ProofPair struct {
	Index int    //0表示左子节点,1表示右子节点
	Hash  []byte //左子节点或右子节点的哈希值
}

func (proofPair *ProofPair) GetSizeOf() uint {
	return utils.SIZEOFINT + uint(len(proofPair.Hash))*utils.SIZEOFBYTE
}

type MHTProof struct {
	isExist       bool        //是否存在，存在则默克尔
	proofPairs    []ProofPair //存在证明的pairs
	isSegExist    bool        //key不存在时判断segment是否存在，存在则根据segment中所有的值构建segment的默克尔树根
	values        []string    //segment中所有的值，用于构建segment的默克尔树根
	segKeys       []string    //所有segment的segKey，在segment不存在时有效
	segRootHashes [][]byte    //所有segment的根哈希，在segment不存在时有效，用于计算segment的默克尔树根
}

func (mhtProof *MHTProof) GetSizeOf() uint {
	ret := 2 * utils.SIZEOFBOOL
	for _, proofPair := range mhtProof.proofPairs {
		ret += proofPair.GetSizeOf()
	}
	for _, value := range mhtProof.values {
		ret += uint(len(value)) * utils.SIZEOFBYTE
	}
	for _, segKey := range mhtProof.segKeys {
		ret += uint(len(segKey)) * utils.SIZEOFBYTE
	}
	for _, hash := range mhtProof.segRootHashes {
		ret += uint(len(hash)) * utils.SIZEOFBYTE
	}
	return ret
}

// NewMHTProof 新建一个MHTProof
func NewMHTProof(isExist bool, proofPairs []ProofPair, isSegExist bool, values []string, segKeys []string, segRootHashes [][]byte) *MHTProof {
	return &MHTProof{isExist, proofPairs, isSegExist, values, segKeys, segRootHashes}
}

func (mhtProof *MHTProof) GetIsExist() bool {
	return mhtProof.isExist
}

func (mhtProof *MHTProof) GetProofPairs() []ProofPair {
	return mhtProof.proofPairs
}

func (mhtProof *MHTProof) GetIsSegExist() bool {
	return mhtProof.isSegExist
}

func (mhtProof *MHTProof) GetValues() []string {
	return mhtProof.values
}

func (mhtProof *MHTProof) GetSegKeys() []string {
	return mhtProof.segKeys
}

func (mhtProof *MHTProof) GetSegRootHashes() [][]byte {
	return mhtProof.segRootHashes
}
