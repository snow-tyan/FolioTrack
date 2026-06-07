package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Username     string         `gorm:"uniqueIndex;not null;size:50" json:"username"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Holdings     []Holding      `gorm:"foreignKey:UserID" json:"holdings,omitempty"`
	Combos       []Combo        `gorm:"foreignKey:UserID" json:"combos,omitempty"`
}

type Asset struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Symbol       string         `gorm:"uniqueIndex:idx_symbol_market;not null;size:20" json:"symbol"`
	Market       string         `gorm:"uniqueIndex:idx_symbol_market;not null;size:20" json:"market"` // A-share, HK-stock, US-stock, Fund
	Name         string         `gorm:"not null;size:100" json:"name"`
	CurrentPrice float64        `gorm:"type:decimal(12,4);default:0" json:"currentPrice"`
	PrevClose    float64        `gorm:"type:decimal(12,4);default:0" json:"prevClose"`
	LastUpdated  time.Time      `json:"lastUpdated"`
}

type Holding struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"uniqueIndex:idx_user_asset;not null" json:"userId"`
	AssetID   uint           `gorm:"uniqueIndex:idx_user_asset;not null" json:"assetId"`
	Asset     Asset          `gorm:"foreignKey:AssetID" json:"asset"`
	Quantity  float64        `gorm:"type:decimal(16,4);default:0" json:"quantity"`
	CostPrice float64        `gorm:"type:decimal(16,4);default:0" json:"costPrice"`
	Combos    []Combo        `gorm:"many2many:holding_combos;constraint:OnDelete:CASCADE;" json:"combos"`
}

type Combo struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"not null" json:"userId"`
	Name      string         `gorm:"not null;size:50" json:"name"`
	Color     string         `gorm:"size:20" json:"color"` // e.g. "#409EFF"
	Holdings  []Holding      `gorm:"many2many:holding_combos;constraint:OnDelete:CASCADE;" json:"holdings,omitempty"`
}

// Global DB instance
var DB *gorm.DB

func InitDB(db *gorm.DB) {
	DB = db
}
