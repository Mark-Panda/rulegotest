package model

import (
	"time"

	"gorm.io/gorm"
)

type Regulation struct {
	gorm.Model
	Id          int64           `gorm:"column:id"`
	RuleChainId string          `gorm:"column:rule_chain_id"`
	RuleConfig  string          `gorm:"column:rule_config"`
	CreatedAt   *time.Time      `gorm:"column:created_at"`
	UpdatedAt   *time.Time      `gorm:"column:updated_at"`
	DeletedAt   *gorm.DeletedAt `gorm:"column:deleted_at"`
}
