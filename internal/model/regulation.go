package model

import (
	"gorm.io/gorm"
)

type Regulation struct {
	gorm.Model
	RuleChainId string `gorm:"column:rule_chain_id"`
	RuleConfig  string `gorm:"column:rule_config"`
}
