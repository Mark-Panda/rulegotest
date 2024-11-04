package dao

import (
	"os"
	"path"
	"path/filepath"
	"ruleGoProject/config"
	"ruleGoProject/internal/constants"
	"ruleGoProject/internal/model"

	"github.com/rulego/rulego/utils/fs"
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

// 弃用 改为存储持久化 SaveToDataBase
func (d *RuleDao) Save(username, chainId string, def []byte) error {
	var paths = []string{d.config.DataDir, constants.DirWorkflows}
	paths = append(paths, username, constants.DirWorkflowsRule)
	pathStr := path.Join(paths...)
	//创建文件夹
	_ = fs.CreateDirs(pathStr)
	//保存到文件
	v, _ := json.Format(def)
	//保存规则链到文件
	return fs.SaveFile(filepath.Join(pathStr, chainId+constants.RuleChainFileSuffix), v)
}

// 弃用 改为持久化删除 DeleteToDataBase
func (d *RuleDao) Delete(username, chainId string) error {
	var paths = []string{d.config.DataDir, constants.DirWorkflows}
	paths = append(paths, username, constants.DirWorkflowsRule)
	pathStr := path.Join(paths...)
	file := filepath.Join(pathStr, chainId+constants.RuleChainFileSuffix)
	return os.RemoveAll(file)
}

// 保存或更新到数据库
func (d *RuleDao) SaveToDataBase(chainId string, def []byte) error {
	v, _ := json.Format(def)
	ruleConfigInfo, gErr := FindRegulationByRuleChainId(chainId)
	if gErr != nil {
		return gErr
	}
	if ruleConfigInfo != nil {
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
