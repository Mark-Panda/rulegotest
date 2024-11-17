package dao

import (
	"ruleGoProject/config"
	"ruleGoProject/internal/model"

	"github.com/rulego/rulego/utils/json"
)

type RuleDao struct {
	config config.Config
}

func NewRuleDao(config config.Config) (*RuleDao, error) {
	return &RuleDao{
		config: config,
	}, nil
}

// 保存或更新到数据库
func (d *RuleDao) SaveToDataBase(chainId string, def []byte) error {
	v, _ := json.Format(def)
	ruleConfigInfo, gErr := FindRegulationByRuleChainId(chainId)
	if gErr != nil {
		return gErr
	}
	if ruleConfigInfo != nil && ruleConfigInfo.RuleChainId != "" {
		return UpdateRegulationByRuleChainId(chainId, string(v))
	}
	createInfo := model.Regulation{
		RuleChainId: chainId,
		RuleConfig:  string(v),
	}
	return CreateRegulation(createInfo)
}

// 从数据库删除规则链
func (d *RuleDao) DeleteToDataBase(chainId string) error {
	return DeleteRegulationByRuleChainId(chainId)
}
