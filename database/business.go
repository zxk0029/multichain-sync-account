package database

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Business struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	BusinessUid string    `json:"business_uid"`
	Timestamp   uint64
}

type BusinessDB interface {
	CreateBusiness(string) error
	QueryBusinessAll() ([]*Business, error)
}

type businessDB struct {
	gorm *gorm.DB
}

func NewBusinessDB(db *gorm.DB) BusinessDB {
	return &businessDB{gorm: db}
}

func (db *businessDB) CreateBusiness(requestId string) error {
	result := db.gorm.Table("business").Create(&Business{BusinessUid: requestId, Timestamp: uint64(time.Now().Unix())})
	if result.Error != nil {
		log.Error("create business batch fail", "Err", result.Error)
		return result.Error
	}
	return nil
}

func (db *businessDB) QueryBusinessAll() ([]*Business, error) {
	var business []*Business
	result := db.gorm.Table("business").Find(&business)
	if result.Error != nil {
		log.Error("query business all fail", "Err", result.Error)
		return nil, result.Error
	}
	return business, nil
}
