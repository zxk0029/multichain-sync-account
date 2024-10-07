package scanner

import (
	"context"
	"fmt"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/bigint"
	"github.com/dapplink-labs/multichain-transaction-syncs/synchronizer/wallet-chain-node/wallet"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
)

type BlockchainScanner struct {
	client wallet.WalletServiceClient
	chain  string
}

// NewBlockchainScanner 初始化一个新的区块链扫描器实例
func NewBlockchainScanner(rpc wallet.WalletServiceClient, chain string) (*BlockchainScanner, error) {
	return &BlockchainScanner{client: rpc, chain: chain}, nil
}

// GetBlockByNumber 根据区块号获取区块信息
func (b *BlockchainScanner) GetBlockByNumber(blockNumber *big.Int) (*wallet.BlockInfoResponse, error) {
	block, err := b.client.GetBlockByNumber(context.Background(), &wallet.BlockInfoRequest{
		Chain:  b.chain,
		Height: blockNumber.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %v", err)
	}
	return block, nil
}

// GetBlockByRange 根据范围获取区块信息
func (b *BlockchainScanner) GetBlockByRange(start, end *big.Int) (*wallet.BlockByRangeResponse, error) {
	block, err := b.client.GetBlockByRange(context.Background(), &wallet.BlockByRangeRequest{
		Chain: b.chain,
		Start: start.String(),
		End:   end.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %v", err)
	}
	return block, nil
}

// ScanBlocks 扫描从 lastScannedBlock 开始的区块，每次获取 batchSize 个区块
func (b *BlockchainScanner) ScanBlocks(lastScannedBlock *big.Int, batchSize uint64, txHandler func(*wallet.BlockInfoTransactionList)) error {
	// 获取链上的最新区块号
	latestHeader, err := b.client.GetBlockHeaderByNumber(context.Background(), &wallet.BlockHeaderRequest{Chain: b.chain})
	if err != nil {
		return fmt.Errorf("failed to get latest block header: %v", err)
	}
	if latestHeader == nil {
		return fmt.Errorf("failed to get latest block header: %v", err)
	}
	latestBlockNumber := bigint.StringToBigInt(latestHeader.Number)
	// 如果已扫描区块高等于最新区块高，则不需要继续扫描
	if latestBlockNumber == nil || lastScannedBlock.Cmp(latestBlockNumber) >= 0 {
		log.Info("All blocks are already scanned. latest block number: " + latestBlockNumber.String())
		return nil
	}

	// 开始扫描区块
	for lastScannedBlock.Cmp(latestBlockNumber) < 0 {
		// 计算要扫描的结束区块号
		endBlockNumber := new(big.Int).Add(lastScannedBlock, big.NewInt(int64(batchSize)))
		if endBlockNumber.Cmp(latestBlockNumber) > 0 {
			endBlockNumber = latestBlockNumber
		}
		//blocks, err := b.GetBlockByRange(lastScannedBlock, endBlockNumber)
		//if err != nil {
		//	log.Error("Error retrieving blocks %v-%v: %v", lastScannedBlock, endBlockNumber, err)
		//	continue
		//}
		//for _, block := range blocks.Blocks {
		//	// 处理每个区块中的交易
		//	for _, tx := range block.Transactions {
		//		txHandler(tx)
		//	}
		//}
		// 扫描区块范围内的区块
		for blockNum := new(big.Int).Set(lastScannedBlock); blockNum.Cmp(endBlockNumber) <= 0; blockNum.Add(blockNum, big.NewInt(1)) {
			block, err := b.GetBlockByNumber(blockNum)
			if err != nil {
				log.Error("Error retrieving block %v: %v", blockNum, err)
				continue
			}
			// 处理每个区块中的交易
			for _, tx := range block.Transactions {
				txHandler(tx)
			}
		}
		// 更新已扫描的最后区块号
		lastScannedBlock = endBlockNumber
	}
	log.Info("All blocks are scanned.")
	return nil
}
