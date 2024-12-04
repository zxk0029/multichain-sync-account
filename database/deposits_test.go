package database

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/dapplink-labs/multichain-sync-account/common/json2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
)

func TestDepositsDB_StoreAndQuery(t *testing.T) {
	const (
		CurrentRequestId = 1
	)

	db := SetupDb()
	depositsDB := NewDepositsDB(db.gorm)

	deposit := &Deposits{
		GUID:         uuid.New(),
		BlockHash:    common.HexToHash("0x1"),
		BlockNumber:  big.NewInt(1),
		Hash:         common.HexToHash("0x2"),
		FromAddress:  common.HexToAddress("0x3"),
		ToAddress:    common.HexToAddress("0x4"),
		TokenAddress: common.HexToAddress("0x5"),
		TokenId:      "1",
		TokenMeta:    "meta",
		Fee:          big.NewInt(100),
		Amount:       big.NewInt(1000),
		Confirms:     0,
		Status:       DepositStatusPending,
		Timestamp:    1234567890,
	}

	// Store the deposit
	err := depositsDB.StoreDeposits(strconv.Itoa(CurrentRequestId), []*Deposits{deposit})
	if err != nil {
		t.Fatalf("failed to store deposit: %v", err)
	}

	// Query the deposit
	notifyDeposits, err := depositsDB.QueryNotifyDeposits(strconv.Itoa(CurrentRequestId))
	if err != nil {
		t.Fatalf("failed to query notify deposits: %v", err)
	}
	t.Logf("notifyDeposits %v", json2.ToPrettyJSON(notifyDeposits))

	blockNumber := uint64(10)
	confirms := uint64(5)
	err = depositsDB.UpdateDepositsComfirms(strconv.Itoa(CurrentRequestId), blockNumber, confirms)
	if err != nil {
		t.Fatalf("failed to update deposit confirms: %v", err)
	}

	// Query the deposit
	notifyDepositsV2, err := depositsDB.QueryNotifyDeposits(strconv.Itoa(CurrentRequestId))
	if err != nil {
		t.Fatalf("failed to query notify deposits: %v", err)
	}
	t.Logf("notifyDeposits_v2 %v", json2.ToPrettyJSON(notifyDepositsV2))
}

func TestUpdateDepositsNotifyStatus(t *testing.T) {
	const (
		CurrentRequestId = 1
	)

	db := SetupDb()
	depositsDB := NewDepositsDB(db.gorm)

	deposit := &Deposits{
		GUID:         uuid.New(),
		BlockHash:    common.HexToHash("0x1"),
		BlockNumber:  big.NewInt(1),
		Hash:         common.HexToHash("0x2"),
		FromAddress:  common.HexToAddress("0x3"),
		ToAddress:    common.HexToAddress("0x4"),
		TokenAddress: common.HexToAddress("0x5"),
		TokenId:      "1",
		TokenMeta:    "meta",
		Fee:          big.NewInt(100),
		Amount:       big.NewInt(1000),
		Confirms:     0,
		Status:       DepositStatusPending,
		Timestamp:    1234567890,
	}

	err := depositsDB.StoreDeposits(strconv.Itoa(CurrentRequestId), []*Deposits{deposit})
	if err != nil {
		t.Fatalf("failed to store deposit: %v", err)
	}

	// Query the deposit
	notifyDeposits, err := depositsDB.QueryNotifyDeposits(strconv.Itoa(CurrentRequestId))
	if err != nil {
		t.Fatalf("failed to query notify deposits: %v", err)
	}
	t.Logf("notifyDeposits %v", json2.ToPrettyJSON(notifyDeposits))

	newStatus := DepositStatusWalletDone
	err = depositsDB.UpdateDepositsNotifyStatus(strconv.Itoa(CurrentRequestId), newStatus, []*Deposits{deposit})
	if err != nil {
		t.Fatalf("failed to update deposit notify status: %v", err)
	}

	// Query the deposit
	notifyDepositsV2, err := depositsDB.QueryNotifyDeposits(strconv.Itoa(CurrentRequestId))
	if err != nil {
		t.Fatalf("failed to query notify deposits: %v", err)
	}
	t.Logf("notifyDepositsV2 %v", json2.ToPrettyJSON(notifyDepositsV2))

}
