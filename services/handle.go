package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"

	"github.com/dapplink-labs/multichain-sync-account/common/json2"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/database/dynamic"
	dal_wallet_go "github.com/dapplink-labs/multichain-sync-account/protobuf/dal-wallet-go"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient/chain-account/account"
	"gorm.io/gorm"
)

const (
	Network = "mainnet"
)

var (
	EthGasLimit   uint64 = 60000
	TokenGasLimit uint64 = 120000
	Min1Gwei      uint64 = 1000000000
	//maxFeePerGas                = "135177480"
	//maxPriorityFeePerGas        = "535177480"
)

func (bws *BusinessMiddleWireServices) BusinessRegister(ctx context.Context, request *dal_wallet_go.BusinessRegisterRequest) (*dal_wallet_go.BusinessRegisterResponse, error) {
	if request.RequestId == "" || request.NotifyUrl == "" {
		return &dal_wallet_go.BusinessRegisterResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "invalid params",
		}, nil
	}

	// 1. 检查业务是否已存在
	existingBusiness, err := bws.db.Business.QueryBusinessByUuid(request.RequestId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("query business fail", "err", err)
		return &dal_wallet_go.BusinessRegisterResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "database error",
		}, nil
	}

	// 2. 如果业务不存在，创建新业务
	if existingBusiness == nil {
		business := &database.Business{
			GUID:        uuid.New(),
			BusinessUid: request.RequestId,
			NotifyUrl:   request.NotifyUrl,
			Timestamp:   uint64(time.Now().Unix()),
		}
		if err := bws.db.Business.StoreBusiness(business); err != nil {
			log.Error("store business fail", "err", err)
			return &dal_wallet_go.BusinessRegisterResponse{
				Code: dal_wallet_go.ReturnCode_ERROR,
				Msg:  "store db fail",
			}, nil
		}
	}

	// 3. 创建或更新业务链关系和相关表
	if err := dynamic.CreateTableFromTemplate(request.RequestId, bws.accountClient.ChainName, bws.db); err != nil {
		log.Error("create tables fail", "err", err, "chain", bws.accountClient.ChainName)
		return &dal_wallet_go.BusinessRegisterResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  fmt.Sprintf("failed to create tables for chain %s", bws.accountClient.ChainName),
		}, nil
	}

	return &dal_wallet_go.BusinessRegisterResponse{
		Code: dal_wallet_go.ReturnCode_SUCCESS,
		Msg:  "config business success",
	}, nil
}

// ExportAddressesByPublicKeys todo change to tx
// ExportAddressesByPublicKeys
func (bws *BusinessMiddleWireServices) ExportAddressesByPublicKeys(ctx context.Context, request *dal_wallet_go.ExportAddressesRequest) (*dal_wallet_go.ExportAddressesResponse, error) {
	var (
		retAddressess []*dal_wallet_go.Address
		dbAddresses   []*database.Addresses
		balances      []*database.Balances
	)

	for _, value := range request.PublicKeys {
		address := bws.accountClient.ExportAddressByPubKey("", value.PublicKey)
		item := &dal_wallet_go.Address{
			Type:    value.Type,
			Address: address,
		}
		parseAddressType, err := database.ParseAddressType(value.Type)
		if err != nil {
			log.Error("handle ParseAddressType fail", "type", value.Type, "err", err)
			return nil, err
		}
		_, _, balance := bws.accountClient.GetAccount(address)

		dbAddress := &database.Addresses{
			GUID:        uuid.New(),
			Address:     address,
			AddressType: parseAddressType,
			PublicKey:   value.PublicKey,
			Timestamp:   uint64(time.Now().Unix()),
		}
		dbAddresses = append(dbAddresses, dbAddress)

		balanceItem := &database.Balances{
			GUID:         uuid.New(),
			Address:      address,
			TokenAddress: common.Address{}.String(),
			AddressType:  parseAddressType,
			Balance:      big.NewInt(int64(balance)),
			LockBalance:  big.NewInt(0),
			Timestamp:    uint64(time.Now().Unix()),
		}
		balances = append(balances, balanceItem)

		retAddressess = append(retAddressess, item)
	}
	err := bws.db.Addresses.StoreAddresses(request.RequestId, bws.accountClient.ChainName, dbAddresses)
	if err != nil {
		return &dal_wallet_go.ExportAddressesResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "store address to db fail",
		}, nil
	}
	err = bws.db.Balances.StoreBalances(request.RequestId, bws.accountClient.ChainName, balances)
	if err != nil {
		return &dal_wallet_go.ExportAddressesResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "store balance to db fail",
		}, nil
	}
	return &dal_wallet_go.ExportAddressesResponse{
		Code:      dal_wallet_go.ReturnCode_SUCCESS,
		Msg:       "generate address success",
		Addresses: retAddressess,
	}, nil
}

func (bws *BusinessMiddleWireServices) CreateUnSignTransaction(ctx context.Context, request *dal_wallet_go.UnSignTransactionRequest) (*dal_wallet_go.UnSignTransactionResponse, error) {
	response := &dal_wallet_go.UnSignTransactionResponse{
		Code:     dal_wallet_go.ReturnCode_ERROR,
		UnSignTx: "0x00",
	}

	if err := validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	transactionType, err := database.ParseTransactionType(request.TxType)
	if err != nil {
		return nil, fmt.Errorf("invalid request TxType: %w", err)
	}

	amountBig, ok := new(big.Int).SetString(request.Value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount value: %s", request.Value)
	}
	guid := uuid.New()

	nonceStr, err := bws.getAccountNonce(ctx, request.Chain, request.From)
	if err != nil {
		return nil, fmt.Errorf("get account nonce failed: %w", err)
	}

	feeInfo, err := bws.getFeeInfo(ctx, request.Chain, request.From)
	if err != nil {
		return nil, fmt.Errorf("get fee info failed: %w", err)
	}
	gasLimit, contractAddress := bws.getGasAndContractInfo(request.ContractAddress)

	switch transactionType {
	case database.TxTypeDeposit:
		err := bws.StoreDeposits(ctx, request, guid, amountBig, gasLimit, feeInfo, transactionType)
		if err != nil {
			return nil, fmt.Errorf("store deposit failed: %w", err)
		}
	case database.TxTypeWithdraw:
		if err := bws.storeWithdraw(request, guid, amountBig, gasLimit, feeInfo, transactionType); err != nil {
			return nil, fmt.Errorf("store withdraw failed: %w", err)
		}
	case database.TxTypeCollection, database.TxTypeHot2Cold, database.TxTypeCold2Hot:
		if err := bws.storeInternal(request, guid, amountBig, gasLimit, feeInfo, transactionType); err != nil {
			return nil, fmt.Errorf("store internal failed: %w", err)
		}
	default:
		response.Msg = "Unsupported transaction type"
		response.UnSignTx = "0x00"
		return response, nil
	}

	// 构建交易请求
	var base64Str string
	if config, ok := database.ChainTokenTypes[strings.ToLower(request.Chain)]; ok && config.IsEVM {
		// EVM 链使用 EIP-1559 交易格式
		nonce, err := strconv.ParseUint(nonceStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid nonce value: %w", err)
		}

		dynamicFeeTxReq := Eip1559DynamicFeeTx{
			ChainId:              request.ChainId,
			Nonce:                nonce,
			FromAddress:          request.From,
			ToAddress:            request.To,
			GasLimit:             gasLimit,
			MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
			MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
			Amount:               request.Value,
			ContractAddress:      contractAddress,
		}
		data := json2.ToJSON(dynamicFeeTxReq)
		base64Str = base64.StdEncoding.EncodeToString(data)
	} else {
		// 非 EVM 链使用各自的交易格式
		txReq := map[string]interface{}{
			"chain":           request.Chain,
			"from":            request.From,
			"to":              request.To,
			"amount":          request.Value,
			"nonce":           nonceStr, // 使用原始 nonce 字符串
			"contractAddress": contractAddress,
			"gasLimit":        gasLimit,
		}
		data := json2.ToJSON(txReq)
		base64Str = base64.StdEncoding.EncodeToString(data)
	}

	unsignTx := &account.UnSignTransactionRequest{
		Chain:    request.Chain,
		Network:  Network,
		Base64Tx: base64Str,
	}

	log.Info("BusinessMiddleWireServices CreateUnSignTransaction unsignTx", json2.ToJSONString(unsignTx))
	returnTx, err := bws.accountClient.AccountRpClient.CreateUnSignTransaction(ctx, unsignTx)
	log.Info("BusinessMiddleWireServices CreateUnSignTransaction returnTx", json2.ToJSONString(returnTx))
	if err != nil {
		log.Error("create un sign transaction fail", "err", err)
		return nil, fmt.Errorf("create unsigned transaction failed: %w", err)
	}

	response.Code = dal_wallet_go.ReturnCode_SUCCESS
	response.Msg = "submit withdraw and build un sign tranaction success"
	response.TransactionId = guid.String()
	response.UnSignTx = returnTx.UnSignTx
	return response, nil
}

func (bws *BusinessMiddleWireServices) BuildSignedTransaction(ctx context.Context, request *dal_wallet_go.SignedTransactionRequest) (*dal_wallet_go.SignedTransactionResponse, error) {
	response := &dal_wallet_go.SignedTransactionResponse{
		Code: dal_wallet_go.ReturnCode_ERROR,
	}
	// 1. Get transaction from database based on type
	var (
		fromAddress          string
		toAddress            string
		amount               string
		tokenAddress         string
		gasLimit             uint64
		maxFeePerGas         string
		maxPriorityFeePerGas string
	)

	transactionType, err := database.ParseTransactionType(request.TxType)
	if err != nil {
		return nil, fmt.Errorf("invalid request TxType: %w", err)
	}

	switch transactionType {
	case database.TxTypeDeposit:
		tx, err := bws.db.Deposits.QueryDepositsById(request.RequestId, bws.accountClient.ChainName, request.TransactionId)
		if err != nil {
			return nil, fmt.Errorf("query deposit failed: %w", err)
		}
		if tx == nil {
			response.Msg = "Deposit transaction not found"
			return response, nil
		}
		fromAddress = tx.FromAddress
		toAddress = tx.ToAddress
		amount = tx.Amount.String()
		tokenAddress = tx.TokenAddress
		gasLimit = tx.GasLimit
		maxFeePerGas = tx.MaxFeePerGas
		maxPriorityFeePerGas = tx.MaxPriorityFeePerGas

	case database.TxTypeWithdraw:
		tx, err := bws.db.Withdraws.QueryWithdrawsById(request.RequestId, bws.accountClient.ChainName, request.TransactionId)
		if err != nil {
			return nil, fmt.Errorf("query withdraw failed: %w", err)
		}
		if tx == nil {
			response.Msg = "Withdraw transaction not found"
			return response, nil
		}
		fromAddress = tx.FromAddress
		toAddress = tx.ToAddress
		amount = tx.Amount.String()
		tokenAddress = tx.TokenAddress
		gasLimit = tx.GasLimit
		maxFeePerGas = tx.MaxFeePerGas
		maxPriorityFeePerGas = tx.MaxPriorityFeePerGas

	case database.TxTypeCollection, database.TxTypeHot2Cold, database.TxTypeCold2Hot:
		tx, err := bws.db.Internals.QueryInternalsById(request.RequestId, bws.accountClient.ChainName, request.TransactionId)
		if err != nil {
			return nil, fmt.Errorf("query internal failed: %w", err)
		}
		if tx == nil {
			response.Msg = "Internal transaction not found"
			return response, nil
		}
		fromAddress = tx.FromAddress
		toAddress = tx.ToAddress
		amount = tx.Amount.String()
		tokenAddress = tx.TokenAddress
		gasLimit = tx.GasLimit
		maxFeePerGas = tx.MaxFeePerGas
		maxPriorityFeePerGas = tx.MaxPriorityFeePerGas

	default:
		response.Msg = "Unsupported transaction type"
		response.SignedTx = "0x00"
		return response, nil
	}

	// 2. Get current nonce
	nonceStr, err := bws.getAccountNonce(ctx, request.Chain, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("get account nonce failed: %w", err)
	}

	// 3. Build transaction data
	var base64Str string
	if config, ok := database.ChainTokenTypes[strings.ToLower(request.Chain)]; ok && config.IsEVM {
		// Convert nonce for EVM chains
		nonce, err := strconv.ParseUint(nonceStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid nonce value: %w", err)
		}

		// Build EIP-1559 transaction
		dynamicFeeTx := Eip1559DynamicFeeTx{
			ChainId:              request.ChainId,
			Nonce:                nonce,
			FromAddress:          fromAddress,
			ToAddress:            toAddress,
			GasLimit:             gasLimit,
			MaxFeePerGas:         maxFeePerGas,
			MaxPriorityFeePerGas: maxPriorityFeePerGas,
			Amount:               amount,
			ContractAddress:      tokenAddress,
		}
		data := json2.ToJSON(dynamicFeeTx)
		base64Str = base64.StdEncoding.EncodeToString(data)
	} else {
		// Non-EVM chains use their own transaction format
		txReq := map[string]interface{}{
			"chain":           request.Chain,
			"from":            fromAddress,
			"to":              toAddress,
			"amount":          amount,
			"nonce":           nonceStr, // Use original nonce string
			"contractAddress": tokenAddress,
			"gasLimit":        gasLimit,
		}
		data := json2.ToJSON(txReq)
		base64Str = base64.StdEncoding.EncodeToString(data)
	}

	// 4. Build signed transaction
	signedTxReq := &account.SignedTransactionRequest{
		Chain:     request.Chain,
		Network:   Network,
		Signature: request.Signature,
		Base64Tx:  base64Str,
	}

	log.Info("BuildSignedTransaction request", "base64Tx", base64Str)
	returnTx, err := bws.accountClient.AccountRpClient.BuildSignedTransaction(ctx, signedTxReq)
	log.Info("BuildSignedTransaction response", "returnTx", json2.ToJSONString(returnTx))
	if err != nil {
		return nil, fmt.Errorf("build signed transaction failed: %w", err)
	}

	// 5. Update transaction status in database
	var updateErr error
	switch transactionType {
	case database.TxTypeDeposit:
		updateErr = bws.db.Deposits.UpdateDepositById(request.RequestId, bws.accountClient.ChainName, request.TransactionId, returnTx.SignedTx, database.TxStatusSigned)
	case database.TxTypeWithdraw:
		updateErr = bws.db.Withdraws.UpdateWithdrawById(request.RequestId, bws.accountClient.ChainName, request.TransactionId, returnTx.SignedTx, database.TxStatusSigned)
	case database.TxTypeCollection, database.TxTypeHot2Cold, database.TxTypeCold2Hot:
		updateErr = bws.db.Internals.UpdateInternalById(request.RequestId, bws.accountClient.ChainName, request.TransactionId, returnTx.SignedTx, database.TxStatusSigned)
	default:
		response.Msg = "Unsupported transaction type"
		response.SignedTx = "0x00"
		return response, nil
	}

	if updateErr != nil {
		return nil, fmt.Errorf("update transaction status failed: %w", updateErr)
	}

	response.SignedTx = returnTx.SignedTx
	response.Msg = "build signed tx success"
	response.Code = dal_wallet_go.ReturnCode_SUCCESS
	return response, nil
}

func (bws *BusinessMiddleWireServices) SetTokenAddress(ctx context.Context, request *dal_wallet_go.SetTokenAddressRequest) (*dal_wallet_go.SetTokenAddressResponse, error) {
	var (
		tokenList []database.Tokens
	)
	for _, value := range request.TokenList {
		CollectAmountBigInt, _ := new(big.Int).SetString(value.CollectAmount, 10)
		ColdAmountBigInt, _ := new(big.Int).SetString(value.ColdAmount, 10)
		token := database.Tokens{
			GUID:          uuid.New(),
			TokenAddress:  value.Address,
			Decimals:      uint8(value.Decimals),
			TokenName:     value.TokenName,
			CollectAmount: CollectAmountBigInt,
			ColdAmount:    ColdAmountBigInt,
			Timestamp:     uint64(time.Now().Unix()),
		}
		tokenList = append(tokenList, token)
	}
	err := bws.db.Tokens.StoreTokens(request.RequestId, bws.accountClient.ChainName, tokenList)
	if err != nil {
		log.Error("set token address fail", "err", err)
		return nil, err
	}
	return &dal_wallet_go.SetTokenAddressResponse{
		Code: 1,
		Msg:  "set token address success",
	}, nil
}

// FeeInfo 结构体用于存储解析后的费用信息
type FeeInfo struct {
	GasPrice       *big.Int // 基础 gas 价格
	GasTipCap      *big.Int // 小费上限
	Multiplier     int64    // 倍数
	MultipliedTip  *big.Int // 小费 * 倍数
	MaxPriorityFee *big.Int // 小费 * 倍数 * 2 (最大上限)
}

// ParseFastFee 解析 FastFee 字符串并计算相关费用
func ParseFastFee(fastFee string) (*FeeInfo, error) {
	// 1. 按 "|" 分割字符串
	parts := strings.Split(fastFee, "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid fast fee format: %s", fastFee)
	}

	// 2. 解析 GasPrice (baseFee)
	gasPrice := new(big.Int)
	if _, ok := gasPrice.SetString(parts[0], 10); !ok {
		return nil, fmt.Errorf("invalid gas price: %s", parts[0])
	}

	// 3. 解析 GasTipCap
	gasTipCap := new(big.Int)
	if _, ok := gasTipCap.SetString(parts[1], 10); !ok {
		return nil, fmt.Errorf("invalid gas tip cap: %s", parts[1])
	}

	// 4. 解析倍数（去掉 "*" 前缀）
	multiplierStr := strings.TrimPrefix(parts[2], "*")
	multiplier, err := strconv.ParseInt(multiplierStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid multiplier: %s", parts[2])
	}

	// 5. 计算 MultipliedTip (小费 * 倍数)
	multipliedTip := new(big.Int).Mul(
		gasTipCap,
		big.NewInt(multiplier),
	)
	// 设置最小小费阈值 (1 Gwei)
	//minTipCap := big.NewInt(int64(Min1Gwei))
	//if multipliedTip.Cmp(minTipCap) < 0 {
	//	multipliedTip = minTipCap
	//}

	// 6. 计算 MaxPriorityFee (baseFee + 小费*倍数*2)
	maxPriorityFee := new(big.Int).Mul(
		multipliedTip,
		big.NewInt(2),
	)
	// 加上 baseFee
	maxPriorityFee.Add(maxPriorityFee, gasPrice)

	return &FeeInfo{
		GasPrice:       gasPrice,
		GasTipCap:      gasTipCap,
		Multiplier:     multiplier,
		MultipliedTip:  multipliedTip,
		MaxPriorityFee: maxPriorityFee,
	}, nil
}

func validateRequest(request *dal_wallet_go.UnSignTransactionRequest) error {
	if request == nil {
		return errors.New("request cannot be nil")
	}
	if request.From == "" {
		return errors.New("from address cannot be empty")
	}
	if request.To == "" {
		return errors.New("to address cannot be empty")
	}
	if request.Value == "" {
		return errors.New("value cannot be empty")
	}
	return nil
}

func determineTokenType(chainName, contractAddress string) database.TokenType {
	if contractAddress == "0x00" {
		contractAddress = common.Address{}.String()
	}
	isNative := contractAddress == database.GetNativeAddress(chainName)
	return database.GetTokenType(chainName, isNative)
}

func (bws *BusinessMiddleWireServices) getAccountNonce(ctx context.Context, chain, address string) (string, error) {
	accountReq := &account.AccountRequest{
		Chain:           chain,
		Network:         Network,
		Address:         address,
		ContractAddress: "0x00",
	}

	accountInfo, err := bws.accountClient.AccountRpClient.GetAccount(ctx, accountReq)
	if err != nil {
		return "", fmt.Errorf("get account info failed: %w", err)
	}

	return accountInfo.Sequence, nil
}

// TODO Solana链需要传入构建后的交易，才能获取交易费用。需要在上游服务（wallet-chain-account）进行处理
func (bws *BusinessMiddleWireServices) getFeeInfo(ctx context.Context, chain, address string) (*FeeInfo, error) {
	accountFeeReq := &account.FeeRequest{
		Chain:   chain,
		Network: Network,
		RawTx:   "",
		Address: address,
	}

	feeResponse, err := bws.accountClient.AccountRpClient.GetFee(ctx, accountFeeReq)
	if err != nil {
		return nil, fmt.Errorf("get fee failed: %w", err)
	}

	return ParseFastFee(feeResponse.FastFee)
}

func (bws *BusinessMiddleWireServices) getGasAndContractInfo(contractAddress string) (uint64, string) {
	if contractAddress == "0x00" {
		return EthGasLimit, "0x00"
	}
	return TokenGasLimit, contractAddress
}

func (bws *BusinessMiddleWireServices) storeWithdraw(request *dal_wallet_go.UnSignTransactionRequest,
	transactionId uuid.UUID, amountBig *big.Int, gasLimit uint64, feeInfo *FeeInfo, transactionType database.TransactionType) error {

	withdraw := &database.Withdraws{
		GUID:                 transactionId,
		Timestamp:            uint64(time.Now().Unix()),
		Status:               database.TxStatusCreateUnsigned,
		BlockHash:            common.Hash{},
		BlockNumber:          big.NewInt(1),
		TxHash:               common.Hash{},
		TxType:               transactionType,
		FromAddress:          request.From,
		ToAddress:            request.To,
		Amount:               amountBig,
		GasLimit:             gasLimit,
		MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
		TokenType:            determineTokenType(request.Chain, request.ContractAddress),
		TokenAddress:         request.ContractAddress,
		TokenId:              request.TokenId,
		TokenMeta:            request.TokenMeta,
		TxSignHex:            "",
	}

	return bws.db.Withdraws.StoreWithdraw(request.RequestId, bws.accountClient.ChainName, withdraw)
}

// 辅助方法：存储内部交易
func (bws *BusinessMiddleWireServices) storeInternal(request *dal_wallet_go.UnSignTransactionRequest,
	transactionId uuid.UUID, amountBig *big.Int, gasLimit uint64, feeInfo *FeeInfo, transactionType database.TransactionType) error {

	internal := &database.Internals{
		GUID:                 transactionId,
		Timestamp:            uint64(time.Now().Unix()),
		Status:               database.TxStatusCreateUnsigned,
		BlockHash:            common.Hash{},
		BlockNumber:          big.NewInt(1),
		TxHash:               common.Hash{},
		TxType:               transactionType,
		FromAddress:          request.From,
		ToAddress:            request.To,
		Amount:               amountBig,
		GasLimit:             gasLimit,
		MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
		TokenType:            determineTokenType(request.Chain, request.ContractAddress),
		TokenAddress:         request.ContractAddress,
		TokenId:              request.TokenId,
		TokenMeta:            request.TokenMeta,
		TxSignHex:            "",
	}

	return bws.db.Internals.StoreInternal(request.RequestId, bws.accountClient.ChainName, internal)
}

func (bws *BusinessMiddleWireServices) StoreDeposits(ctx context.Context,
	depositsRequest *dal_wallet_go.UnSignTransactionRequest, transactionId uuid.UUID, amountBig *big.Int,
	gasLimit uint64, feeInfo *FeeInfo, transactionType database.TransactionType) error {
	fmt.Printf("StoreDeposits - Chain: %s, ContractAddress: %s\n",
		depositsRequest.Chain, depositsRequest.ContractAddress)
	dbDeposit := &database.Deposits{
		GUID:                 transactionId,
		Timestamp:            uint64(time.Now().Unix()),
		Status:               database.TxStatusCreateUnsigned,
		Confirms:             0,
		BlockHash:            common.Hash{},
		BlockNumber:          big.NewInt(1),
		TxHash:               common.Hash{},
		TxType:               transactionType,
		FromAddress:          depositsRequest.From,
		ToAddress:            depositsRequest.To,
		Amount:               amountBig,
		GasLimit:             gasLimit,
		MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
		TokenType:            determineTokenType(depositsRequest.Chain, depositsRequest.ContractAddress),
		TokenAddress:         depositsRequest.ContractAddress,
		TokenId:              depositsRequest.TokenId,
		TokenMeta:            depositsRequest.TokenMeta,
		TxSignHex:            "",
	}

	return bws.db.Deposits.StoreDeposits(depositsRequest.RequestId, bws.accountClient.ChainName, []*database.Deposits{dbDeposit})
}
