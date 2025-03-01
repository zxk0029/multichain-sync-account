package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	dal_wallet_go "github.com/dapplink-labs/multichain-sync-account/protobuf/dal-wallet-go"
)

func init() {
	//SetTestChain(ChainTypeEthereum)
	//SetTestChain(ChainTypeSolana)
	SetTestChain("")
}

// validateAddress 验证地址格式
func validateAddress(t *testing.T, addr string, chainName string, bws *BusinessMiddleWireServices) {
	assert.True(t, bws.accountClient.ValidAddress(addr), "Address should be valid")

	// 链特定的格式验证
	switch chainName {
	case "ethereum":
		assert.True(t, strings.HasPrefix(addr, "0x"), "Ethereum address should start with 0x")
		assert.Equal(t, 42, len(addr), "Ethereum address should be 42 characters long")
	case "solana":
		assert.Equal(t, 44, len(addr), "Solana address should be 44 characters long")
	}
}

// Test_BusinessRegister 测试业务注册功能
func Test_BusinessRegister(t *testing.T) {
	// 遍历所有链配置，但只会测试通过 TestChainType 指定的链
	for chainType, chainConfig := range ChainConfigs {
		if !shouldRunChainTest(t, chainType) {
			continue
		}

		t.Run(fmt.Sprintf("Chain_%s", chainConfig.ChainName), func(t *testing.T) {
			bws, cleanup := setupService(t, chainConfig)
			defer cleanup() // Ensure resources are cleaned up
			ctx := context.Background()

			// 测试正常注册
			t.Run("Success", func(t *testing.T) {
				request := &dal_wallet_go.BusinessRegisterRequest{
					ConsumerToken: "test_token",
					RequestId:     CurrentRequestId,
					NotifyUrl:     NotifyUrl,
				}

				response, err := bws.BusinessRegister(ctx, request)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, dal_wallet_go.ReturnCode_SUCCESS, response.Code)

				// 验证数据库记录
				business, err := bws.db.Business.QueryBusinessByUuid(request.RequestId)
				assert.NoError(t, err)
				assert.NotNil(t, business)
			})
		})
	}
}

// Test_ExportAddresses 测试地址导出功能
func Test_ExportAddresses(t *testing.T) {
	for chainType, chainConfig := range ChainConfigs {
		if !shouldRunChainTest(t, chainType) {
			continue
		}
		t.Logf("Testing with chain: %s", chainConfig.ChainName)
		t.Logf("Using public key: %s", chainConfig.TestPublicKey)
		t.Run(fmt.Sprintf("Chain_%s", chainConfig.ChainName), func(t *testing.T) {
			bws, cleanup := setupService(t, chainConfig)
			defer cleanup() // Ensure resources are cleaned up
			ctx := context.Background()

			t.Run("Success", func(t *testing.T) {
				request := &dal_wallet_go.ExportAddressesRequest{
					ConsumerToken: "test_token",
					RequestId:     CurrentRequestId,
					PublicKeys: []*dal_wallet_go.PublicKey{
						{
							Type:      "eoa",
							PublicKey: chainConfig.TestPublicKey,
						},
					},
				}

				response, err := bws.ExportAddressesByPublicKeys(ctx, request)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, dal_wallet_go.ReturnCode_SUCCESS, response.Code)
				assert.NotEmpty(t, response.Addresses)

				t.Logf("Generated addresses for %s:", chainConfig.ChainName)
				for _, addr := range response.Addresses {
					validateAddress(t, addr.Address, chainConfig.ChainName, bws)
					t.Logf("  - %s (type: %s)", addr.Address, addr.Type)
				}
			})
		})
	}
}

// Test_CreateUnSignTransaction 测试交易创建功能
func Test_CreateUnSignTransaction(t *testing.T) {
	//txTypes := []string{"collection", "hot2cold", "withdraw", "deposit"}
	txTypes := []string{"deposit"}

	for chainType, chainConfig := range ChainConfigs {
		if !shouldRunChainTest(t, chainType) {
			continue
		}

		t.Run(fmt.Sprintf("Chain_%s", chainConfig.ChainName), func(t *testing.T) {
			bws, cleanup := setupService(t, chainConfig)
			defer cleanup() // Ensure resources are cleaned up
			ctx := context.Background()

			// 测试不同类型的交易
			for _, txType := range txTypes {
				t.Run(fmt.Sprintf("TxType_%s", txType), func(t *testing.T) {
					request := &dal_wallet_go.UnSignTransactionRequest{
						ConsumerToken:   "test_token",
						RequestId:       CurrentRequestId,
						ChainId:         chainConfig.ChainId,
						Chain:           chainConfig.ChainName,
						From:            chainConfig.TestAddresses.From,
						To:              chainConfig.TestAddresses.To,
						Value:           "1000000000000000", // 0.001 ETH
						ContractAddress: chainConfig.ContractAddress,
						TxType:          txType,
					}

					response, err := bws.CreateUnSignTransaction(ctx, request)
					assert.NoError(t, err)
					assert.NotNil(t, response)
					assert.Equal(t, dal_wallet_go.ReturnCode_SUCCESS, response.Code)

					t.Logf("Created transaction:")
					t.Logf("  - Type: %s", txType)
					t.Logf("  - Transaction ID: %s", response.TransactionId)
					t.Logf("  - Unsigned TX: %s", response.UnSignTx)
				})
			}
		})
	}
}

// Test_BuildSignedTransaction 测试交易签名功能
func Test_BuildSignedTransaction(t *testing.T) {
	for chainType, chainConfig := range ChainConfigs {
		if !shouldRunChainTest(t, chainType) {
			continue
		}

		t.Run(fmt.Sprintf("Chain_%s", chainConfig.ChainName), func(t *testing.T) {
			bws, cleanup := setupService(t, chainConfig)
			defer cleanup() // Ensure resources are cleaned up
			ctx := context.Background()

			t.Run("Success", func(t *testing.T) {
				request := &dal_wallet_go.SignedTransactionRequest{
					ConsumerToken: "test_token",
					RequestId:     CurrentRequestId,
					Chain:         chainConfig.ChainName,
					ChainId:       chainConfig.ChainId,
					TransactionId: "23bfceac-8c33-4e52-b595-523a1093dcd4",
					Signature:     "ba68c683a88800b25d681ccc5207d8ceadba639868d25e90240516b001627cbd1029aee9f80e1c6c53bf1263833d16764346f4f604b4877275257229bba5b97c01",
					TxType:        "deposit",
				}

				response, err := bws.BuildSignedTransaction(ctx, request)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, dal_wallet_go.ReturnCode_SUCCESS, response.Code)

				t.Logf("Built signed transaction:")
				t.Logf("  - Chain: %s", chainConfig.ChainName)
				t.Logf("  - Transaction ID: %s", request.TransactionId)
				t.Logf("  - Signed TX: %s", response.SignedTx)
			})
		})
	}
}
