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
	createInfo := model.Regulation{
		RuleChainId: chainId,
		RuleConfig:  string(v),
	}
	return SaveRegulation(createInfo)
}

// 从数据库删除规则链
func (d *RuleDao) DeleteToDataBase(chainId string) error {
	return DeleteRegulationByRuleChainId(chainId)
}
