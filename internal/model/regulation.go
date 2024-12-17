package model

import (
	"gorm.io/gorm"
)

type Regulation struct {
	gorm.Model
	Id          int64  `gorm:"column:id"`
	RuleChainId string `gorm:"column:rule_chain_id"`
	RuleConfig  string `gorm:"column:rule_config"`
	CreatedAt   string `gorm:"column:created_at"`
	UpdatedAt   string `gorm:"column:updated_at"`
	DeletedAt   string `gorm:"column:deleted_at"`
}
