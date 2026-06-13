package services

import (
	"errors"
	"fmt"
	"foliotrack/models"
	"strings"

	"gorm.io/gorm"
)

const DefaultAccountName = "默认账户"

var SupportedMarkets = []string{"A-share", "Fund", "HK-stock", "US-stock"}

func IsValidMarket(market string) bool {
	for _, supported := range SupportedMarkets {
		if market == supported {
			return true
		}
	}
	return false
}

func EnsureDefaultAccounts(tx *gorm.DB, userID uint) error {
	for _, market := range SupportedMarkets {
		if _, err := EnsureDefaultAccount(tx, userID, market); err != nil {
			return err
		}
	}
	return nil
}

func EnsureDefaultAccount(tx *gorm.DB, userID uint, market string) (*models.Account, error) {
	if !IsValidMarket(market) {
		return nil, fmt.Errorf("不受支持的资产市场类型")
	}

	var account models.Account
	err := tx.Where("user_id = ? AND market = ? AND is_default = ?", userID, market, true).First(&account).Error
	if err == nil {
		return &account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = tx.Where("user_id = ? AND market = ? AND name = ?", userID, market, DefaultAccountName).First(&account).Error
	if err == nil {
		if !account.IsDefault {
			account.IsDefault = true
			if err := tx.Save(&account).Error; err != nil {
				return nil, err
			}
		}
		return &account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	account = models.Account{
		UserID:    userID,
		Market:    market,
		Name:      DefaultAccountName,
		IsDefault: true,
	}
	if err := tx.Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func ResolveAccountForHolding(tx *gorm.DB, userID uint, market string, accountID uint) (*models.Account, error) {
	if accountID == 0 {
		return EnsureDefaultAccount(tx, userID, market)
	}

	var account models.Account
	err := tx.Where("id = ? AND user_id = ? AND market = ?", accountID, userID, market).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("账户不存在或不属于该市场")
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func ResolveAccountByName(tx *gorm.DB, userID uint, market string, accountName string) (*models.Account, error) {
	name := strings.TrimSpace(accountName)
	if name == "" {
		return EnsureDefaultAccount(tx, userID, market)
	}

	var account models.Account
	err := tx.Where("user_id = ? AND market = ? AND name = ?", userID, market, name).First(&account).Error
	if err == nil {
		return &account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	account = models.Account{
		UserID: userID,
		Market: market,
		Name:   name,
	}
	if err := tx.Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func DropLegacyHoldingIndexes(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Holding{}) {
		return nil
	}

	var constraintNames []string
	if err := db.Raw(`
		SELECT DISTINCT kcu.constraint_name
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc
			ON tc.constraint_schema = kcu.constraint_schema
			AND tc.table_name = kcu.table_name
			AND tc.constraint_name = kcu.constraint_name
		WHERE kcu.table_schema = DATABASE()
			AND kcu.table_name = 'holdings'
			AND tc.constraint_type = 'FOREIGN KEY'
			AND kcu.column_name IN ('user_id', 'asset_id')
	`).Scan(&constraintNames).Error; err != nil {
		return err
	}
	for _, constraintName := range constraintNames {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE holdings DROP FOREIGN KEY `%s`", mysqlIdentifier(constraintName))).Error; err != nil {
			return err
		}
	}

	var count int64
	err := db.Raw(`
		SELECT COUNT(1)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
			AND table_name = 'holdings'
			AND index_name = 'idx_user_asset'
	`).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return db.Exec("DROP INDEX idx_user_asset ON holdings").Error
	}
	return nil
}

func PrepareHoldingAccountColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Holding{}) {
		return nil
	}
	if db.Migrator().HasColumn(&models.Holding{}, "account_id") {
		return nil
	}
	return db.Exec("ALTER TABLE holdings ADD COLUMN account_id bigint unsigned NULL").Error
}

func mysqlIdentifier(value string) string {
	return strings.ReplaceAll(value, "`", "``")
}

func BackfillDefaultAccounts(db *gorm.DB) error {
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, user := range users {
			if err := EnsureDefaultAccounts(tx, user.ID); err != nil {
				return err
			}

			if !tx.Migrator().HasTable(&models.Holding{}) || !tx.Migrator().HasColumn(&models.Holding{}, "account_id") {
				continue
			}

			var holdings []models.Holding
			if err := tx.Preload("Asset").Where("user_id = ? AND (account_id IS NULL OR account_id = 0)", user.ID).Find(&holdings).Error; err != nil {
				return err
			}

			for _, holding := range holdings {
				account, err := EnsureDefaultAccount(tx, user.ID, holding.Asset.Market)
				if err != nil {
					return err
				}
				if err := tx.Model(&models.Holding{}).Where("id = ?", holding.ID).Update("account_id", account.ID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
