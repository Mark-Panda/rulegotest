package dao

import (
	"ruleGoProject/internal/model"
)

// 查询所有需要加载的规则链
func GetAllLoadRegulation() ([]model.Regulation, error) {
	re := make([]model.Regulation, 0)
	err := model.DBClient.Client.Model(&model.Regulation{}).Find(&re).Error
	return re, err
}

// 创建规则链
func CreateRegulation(r model.Regulation) error {
	err := model.DBClient.Client.Create(&r).Error
	return err
}

// 根据ID更新规则链
func UpdateRegulationByRuleChainId(ruleChainId string, ruleConfig string) error {
	return model.DBClient.Client.Model(&model.Regulation{}).Where("rule_chain_id = ?", ruleChainId).Update("rule_config", ruleConfig).Error
}

// 根据规则链ID查询规则链信息
func FindRegulationByRuleChainId(ruleChainId string) (*model.Regulation, error) {
	r := &model.Regulation{}
	err := model.DBClient.Client.Model(&model.Regulation{}).Where("rule_chain_id = ?", ruleChainId).Limit(1).Find(&r).Error
	return r, err
}

// 创建规则链
func SaveRegulation(r model.Regulation) error {
	err := model.DBClient.Client.Save(&r).Error
	return err
}

// 根据规则链ID查询规则链信息
func DeleteRegulationByRuleChainId(ruleChainId string) error {
	r := model.Regulation{}
	err := model.DBClient.Client.Model(&model.Regulation{}).Where("rule_chain_id = ?", ruleChainId).Delete(&r).Error
	return err
}
