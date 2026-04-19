package database

import (
	"time"

	"github.com/shopspring/decimal"
)

type FundSector struct {
	Code       string    `gorm:"primaryKey;type:varchar(50)" json:"code"`
	Name       string    `gorm:"type:varchar(100);not null" json:"name"`
	ParentCode string    `gorm:"type:varchar(50);index" json:"parent_code"`
	Level      int       `gorm:"not null;default:1" json:"level"`
	SortOrder  int       `gorm:"not null;default:0" json:"sort_order"`
	IsEnabled  bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FundSector) TableName() string {
	return "fund_sectors"
}

type FundCategory struct {
	Code        string    `gorm:"primaryKey;type:varchar(50)" json:"code"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	IsEnabled   bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FundCategory) TableName() string {
	return "fund_categories"
}

type InstrumentSectorMap struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	InstrumentCode string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_instrument_sector_map_code_exchange,priority:1" json:"instrument_code"`
	Exchange       string    `gorm:"type:varchar(8);not null;uniqueIndex:idx_instrument_sector_map_code_exchange,priority:2" json:"exchange"`
	SectorCode     string    `gorm:"type:varchar(50);not null;index" json:"sector_code"`
	Source         string    `gorm:"type:varchar(50);not null" json:"source"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (InstrumentSectorMap) TableName() string {
	return "instrument_sector_map"
}

type FundSectorSnapshot struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FundID            string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_fund_sector_snapshot_fund_date,priority:1;index" json:"fund_id"`
	AsOfDate          time.Time `gorm:"type:date;not null;uniqueIndex:idx_fund_sector_snapshot_fund_date,priority:2" json:"as_of_date"`
	PrimarySectorCode string    `gorm:"type:varchar(50);not null;index" json:"primary_sector_code"`
	Source            string    `gorm:"type:varchar(50);not null" json:"source"`
	Confidence        string    `gorm:"type:varchar(20);not null" json:"confidence"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FundSectorSnapshot) TableName() string {
	return "fund_sector_snapshots"
}

type FundSectorBreakdown struct {
	ID            uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	FundID        string          `gorm:"type:varchar(10);not null;uniqueIndex:idx_fund_sector_breakdown_fund_date_sector,priority:1;index" json:"fund_id"`
	AsOfDate      time.Time       `gorm:"type:date;not null;uniqueIndex:idx_fund_sector_breakdown_fund_date_sector,priority:2" json:"as_of_date"`
	SectorCode    string          `gorm:"type:varchar(50);not null;uniqueIndex:idx_fund_sector_breakdown_fund_date_sector,priority:3;index" json:"sector_code"`
	WeightPercent decimal.Decimal `gorm:"type:decimal(8,4);not null" json:"weight_percent"`
	Rank          int             `gorm:"not null;default:0" json:"rank"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FundSectorBreakdown) TableName() string {
	return "fund_sector_breakdown"
}

type FundClassificationOverride struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FundID            string    `gorm:"type:varchar(10);not null;uniqueIndex" json:"fund_id"`
	CategoryCode      string    `gorm:"type:varchar(50);index" json:"category_code"`
	PrimarySectorCode string    `gorm:"type:varchar(50);index" json:"primary_sector_code"`
	SectorTagsJSON    string    `gorm:"type:text" json:"sector_tags_json"`
	Note              string    `gorm:"type:text" json:"note"`
	UpdatedBy         string    `gorm:"type:varchar(100)" json:"updated_by"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FundClassificationOverride) TableName() string {
	return "fund_classification_overrides"
}
