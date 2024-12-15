package dao

import (
	"ruleGoProject/config"
	"ruleGoProject/internal/model"
	"sync"

	"github.com/rulego/rulego/utils/json"
)

// IndexKeySpe key 连接符
var IndexKeySpe = ":"

type RuleDao struct {
	config   config.Config
	username string
	index    Index
	sync.RWMutex
}

// Index 定义索引结构，仅包含必要元数据
type Index struct {
	// key=chainId
	Rules map[string]RuleMeta `json:"rules"`
}

type RuleMeta struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Root       bool   `json:"root"`
	Disabled   bool   `json:"disabled"`
	UpdateTime string `json:"updateTime"`
}

func NewRuleDao(config config.Config, username string) (*RuleDao, error) {
	dao := &RuleDao{
		config:   config,
		username: username,
		index:    Index{Rules: make(map[string]RuleMeta)},
	}

	return dao, nil
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

// 按规则链id从数据库查询规则链
func (d *RuleDao) FindDataBaseByRuleChainId(chainId string) ([]byte, error) {
	ruleConfigInfo, gErr := FindRegulationByRuleChainId(chainId)
	if gErr != nil {
		return nil, gErr
	}
	return []byte(ruleConfigInfo.RuleConfig), nil
}

// 查询最新修改的一条规则链
func (d *RuleDao) FindLatestDataBase() ([]byte, error) {
	ruleConfigInfo, gErr := FindLatestRegulation()
	if gErr != nil {
		return nil, gErr
	}
	return []byte(ruleConfigInfo.RuleConfig), nil
}

func (d *RuleDao) getAllIndex() []RuleMeta {
	d.RLock()
	defer d.RUnlock()
	var items []RuleMeta
	for _, v := range d.index.Rules {
		items = append(items, v)
	}
	return items
}
