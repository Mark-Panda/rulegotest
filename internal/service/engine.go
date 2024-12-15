package service

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"ruleGoProject/config"
	"ruleGoProject/config/logger"
	_ "ruleGoProject/internal/components/xlsx2json"
	"ruleGoProject/internal/constants"
	"ruleGoProject/internal/dao"
	"ruleGoProject/internal/utils/graph"
	"sort"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/rulego/rulego"
	luaEngine "github.com/rulego/rulego-components/pkg/lua_engine"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/node_pool"
	"github.com/rulego/rulego/utils/fs"
	"github.com/rulego/rulego/utils/json"
	"github.com/rulego/rulego/utils/str"
)

var UserRuleEngineServiceImpl *UserRuleEngineService

// UserRuleEngineService 用户规则引擎池
type UserRuleEngineService struct {
	Pool   map[string]*RuleEngineService
	config config.Config
	locker sync.RWMutex
}

func NewUserRuleEngineServiceImpl(c config.Config) (*UserRuleEngineService, error) {
	s := &UserRuleEngineService{
		Pool:   make(map[string]*RuleEngineService),
		config: c,
	}
	userPath := path.Join(c.DataDir, constants.DirWorkflows)
	//创建文件夹
	_ = fs.CreateDirs(userPath)

	entries, err := os.ReadDir(userPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := s.Init(entry.Name()); err != nil {
				logger.Logger.Println("Init "+entry.Name()+" error:", err.Error())
			}
		}
	}
	return s, err
}

// Get 根据用户获取规则引擎池
func (s *UserRuleEngineService) Get(username string) (*RuleEngineService, bool) {
	s.locker.RLock()
	v, ok := s.Pool[username]
	s.locker.RUnlock()
	if !ok {
		if v, err := s.Init(username); err == nil {
			return v, true
		}
	}
	return v, ok
}

func (s *UserRuleEngineService) Init(username string) (*RuleEngineService, error) {
	if v, err := NewRuleEngineServiceAndInitRuleGo(s.config, username); err == nil {
		s.locker.Lock()
		s.Pool[username] = v
		s.locker.Unlock()
		return v, nil
	} else {
		return nil, err
	}
}

type DebugObserver struct {
	chainId  string
	clientId string
	fn       func(chainId, flowType string, nodeId string, msg types.RuleMsg, relationType string, err error)
}
type RuleEngineService struct {
	Pool       *OwnRuleGo
	username   string
	config     config.Config
	ruleConfig types.Config
	logger     *log.Logger
	//基于内存的节点调试数据管理器
	//如果需要查询历史数据，请把调试日志数据存放数据库等可以持久化载体
	ruleChainDebugData *RuleChainDebugData
	onDebugObserver    map[string]*DebugObserver
	ruleDao            *dao.RuleDao
	locker             sync.RWMutex
	mainRuleEngine     types.RuleEngine
}

func NewRuleEngineServiceAndInitRuleGo(c config.Config, username string) (*RuleEngineService, error) {
	ruleConfig := rulego.NewConfig(types.WithDefaultPool(), types.WithLogger(logger.Logger), types.WithNetPool(node_pool.DefaultNodePool))
	service, err := NewRuleEngineService(c, ruleConfig, username)
	if err != nil {
		return nil, err
	}
	service.InitRuleGo(logger.Logger, c.DataDir)
	return service, nil
}

func NewRuleEngineService(c config.Config, ruleConfig types.Config, username string) (*RuleEngineService, error) {
	var pool = NewRuleGo()
	ruleDao, err := dao.NewRuleDao(c, username)
	if err != nil {
		return nil, err
	}
	maxNodeLogSize := c.MaxNodeLogSize
	if maxNodeLogSize == 0 {
		maxNodeLogSize = 40
	}
	service := &RuleEngineService{
		Pool:            pool,
		username:        username,
		logger:          logger.Logger,
		config:          c,
		onDebugObserver: make(map[string]*DebugObserver),
		//基于内存的节点调试数据管理器
		ruleChainDebugData: NewRuleChainDebugData(maxNodeLogSize),
		ruleDao:            ruleDao,
		ruleConfig:         ruleConfig,
	}
	// service.initRuleGo(logger.Logger, c.DataDir)
	return service, nil
}

func (s *RuleEngineService) GetRuleConfig() types.Config {
	return s.ruleConfig
}

func (s *RuleEngineService) ExecuteAndWait(chainId string, msg types.RuleMsg, opts ...types.RuleContextOption) error {
	if e, ok := s.Pool.Get(chainId); ok {
		e.OnMsgAndWait(msg, opts...)
		return nil
	} else {
		return fmt.Errorf("user:%s chainId:%s not found", chainId, s.username)
	}
}
func (s *RuleEngineService) Execute(chainId string, msg types.RuleMsg, opts ...types.RuleContextOption) error {
	if e, ok := s.Pool.Get(chainId); ok {
		e.OnMsg(msg, opts...)
		return nil
	} else {
		return fmt.Errorf("user:%s chainId:%s not found", chainId, s.username)
	}
}

func (s *RuleEngineService) Get(chainId string) (types.RuleChain, bool) {
	if e, ok := s.Pool.Get(chainId); ok {
		return e.Definition(), true
	}
	return types.RuleChain{}, false
}

// GetDsl 获取DSL
func (s *RuleEngineService) GetDsl(chainId, nodeId string) ([]byte, error) {
	var def []byte
	if chainId != "" {
		ruleEngine, ok := s.Pool.Get(chainId)
		if ok {
			if nodeId == "" {
				def = ruleEngine.DSL()
			} else {
				def = ruleEngine.NodeDSL(types.EmptyRuleNodeId, types.RuleNodeId{Id: nodeId, Type: types.NODE})
				if def == nil {
					def = ruleEngine.NodeDSL(types.EmptyRuleNodeId, types.RuleNodeId{Id: nodeId, Type: types.CHAIN})
				}
			}
			return def, nil
		}
	}
	return nil, constants.ErrNotFound
}

// SaveDsl 保存或者更新DSL
func (s *RuleEngineService) SaveDsl(chainId string, def []byte) error {
	var err error
	// TODO: 校验def中ID不可以有特殊字符->
	// 检查规则链如果存在子规则链的话是否存在死循环的情况
	err = graph.CheckInfiniteLoop(chainId, def)
	if err != nil {
		return err
	}
	var ruleChain types.RuleChain
	err = json.Unmarshal(def, &ruleChain)
	if err != nil {
		return err
	}
	//修改更新时间
	s.fillAdditionalInfo(&ruleChain)

	//持久化规则链
	if err = s.ruleDao.SaveToDataBase(chainId, def); err != nil {
		return err
	}

	if ruleChain.RuleChain.Disabled {
		//下架规则
		return s.Undeploy(chainId)
	} else {
		//部署规则链
		return s.Deploy(chainId)
	}
}

// Undeploy 下架规则链引擎实例，并把规则链状态置为disabled
func (s *RuleEngineService) Undeploy(chainId string) error {
	ruleChain, exist := s.Get(chainId)
	if !exist {
		return errors.New("not found for" + chainId)
	}
	s.Pool.Del(chainId)

	ruleChain.RuleChain.Disabled = true

	b, err := json.Marshal(ruleChain)
	if err != nil {
		return err
	}
	//持久化规则链
	if err := s.ruleDao.SaveToDataBase(chainId, b); err != nil {
		return err
	}
	return nil
}

// Load 加载规则链，创建规则链引擎实例，如果规则链状态=disabled则不创建
func (s *RuleEngineService) Load(chainId string) error {
	def, err := s.ruleDao.FindDataBaseByRuleChainId(chainId)
	if ruleEngine, ok := s.Pool.Get(chainId); ok {
		err = ruleEngine.ReloadSelf(def)
	} else {
		_, err = s.Pool.New(chainId, def, rulego.WithConfig(s.ruleConfig))
	}
	if err != nil {
		s.ruleConfig.Logger.Printf("chainId:%s load err: %s", chainId, err.Error())
		var ruleChain types.RuleChain
		jsonErr := json.Unmarshal(def, &ruleChain)
		if jsonErr != nil {
			return jsonErr
		}
		saveErr := s.saveRuleChain(ruleChain, err)
		if saveErr != nil {
			s.ruleConfig.Logger.Printf("saveRuleChain err: %s", saveErr.Error())
		}
		return err
	}
	return nil
}

// Deploy 部署规则链，创建规则链引擎实例，并发规则链状态disabled设置成启用状态
func (s *RuleEngineService) Deploy(chainId string) error {
	var def []byte
	var err error
	ruleChain, exist := s.Get(chainId)
	if !exist {
		return errors.New("not found for" + chainId)
	}

	ruleChain.RuleChain.Disabled = false

	if def, err = json.Marshal(ruleChain); err != nil {
		return err
	} else {
		ruleEngine, ok := s.Pool.Get(chainId)
		if ok {
			err = ruleEngine.ReloadSelf(def)
		} else {
			_, err = s.Pool.New(chainId, def, rulego.WithConfig(s.ruleConfig))
		}
		saveErr := s.saveRuleChain(ruleChain, err)
		if saveErr != nil {
			s.ruleConfig.Logger.Printf("saveRuleChain err: %s", saveErr)
		}
		return err
	}
}

// SetMainChainId 设置主规则链
func (s *RuleEngineService) SetMainChainId(chainId string) error {
	return nil
	// if chainId == "" {
	// 	return errors.New("chainId 不能为空")
	// }
	// if err := s.userSettingDao.Save(constants.SettingKeyMainChainId, chainId); err != nil {
	// 	return err
	// } else {
	// 	if e, ok := s.Pool.Get(chainId); !ok {
	// 		return fmt.Errorf("请先部署规则链")
	// 	} else {
	// 		s.mainRuleEngine = e
	// 		return nil
	// 	}
	// }
}

// saveRuleChain 持久化规则链
func (s *RuleEngineService) saveRuleChain(ruleChain types.RuleChain, whenErr error) error {
	if whenErr != nil {
		ruleChain.RuleChain.Disabled = true
		ruleChain.RuleChain.PutAdditionalInfo(constants.AddiKeyMessage, whenErr.Error())
	}
	if def, err := json.Marshal(ruleChain); err != nil {
		return err
	} else {
		return s.ruleDao.SaveToDataBase(ruleChain.RuleChain.ID, def)
	}
}

func (s *RuleEngineService) GetEngine(chainId string) (types.RuleEngine, bool) {
	return s.Pool.Get(chainId)
}

// List 获取所有规则链
func (s *RuleEngineService) List(keywords string, root *bool, disabled *bool, size, page int) ([]types.RuleChain, int, error) {
	var ruleChains = make([]types.RuleChain, 0)

	s.Pool.Range(func(key, value any) bool {
		engine := value.(types.RuleEngine)
		ruleChains = append(ruleChains, engine.Definition())
		return true
	})

	var updateTimeKey = "updateTime"
	sort.Slice(ruleChains, func(i, j int) bool {
		var iTime, jTime string
		if v, ok := ruleChains[i].RuleChain.GetAdditionalInfo(updateTimeKey); ok {
			iTime = str.ToString(v)
		}
		if v, ok := ruleChains[j].RuleChain.GetAdditionalInfo(updateTimeKey); ok {
			jTime = str.ToString(v)
		}
		return iTime > jTime
	})
	return ruleChains, len(ruleChains), nil
}

func (s *RuleEngineService) GetLatest() ([]byte, error) {
	return s.ruleDao.FindLatestDataBase()
}

// Delete 删除规则链
func (s *RuleEngineService) Delete(chainId string) error {
	s.Pool.Del(chainId)
	if err := s.ruleDao.DeleteToDataBase(chainId); err != nil {
		// if err := s.ruleDao.Delete(s.username, chainId); err != nil {
		return err
	} else {
		return EventServiceImpl.DeleteByChainId(s.username, chainId)
	}
}

// SaveBaseInfo 保存规则链基本信息
func (s *RuleEngineService) SaveBaseInfo(chainId string, baseInfo types.RuleChainBaseInfo) error {
	if chainId != "" {
		ruleEngine, ok := s.Pool.Get(chainId)
		if ok {
			def := ruleEngine.RootRuleChainCtx().Definition()
			def.RuleChain.AdditionalInfo = baseInfo.AdditionalInfo
			def.RuleChain.Name = baseInfo.Name
			def.RuleChain.Root = baseInfo.Root
			def.RuleChain.DebugMode = baseInfo.DebugMode
			//填充更新时间
			s.fillAdditionalInfo(def)
		} else {
			def := types.RuleChain{
				RuleChain: baseInfo,
			}
			//修改更新时间
			s.fillAdditionalInfo(&def)
			jsonStr, _ := json.Marshal(def)
			if e, err := s.Pool.New(chainId, jsonStr, rulego.WithConfig(s.ruleConfig)); nil != err {
				return err
			} else {
				ruleEngine = e
			}
		}
		def, _ := json.Format(ruleEngine.DSL())
		return s.ruleDao.SaveToDataBase(chainId, def)
		// return s.ruleDao.Save(s.username, chainId, def)
	} else {
		return errors.New("not found for" + chainId)
	}
}

// SaveConfiguration 保存规则链配置
func (s *RuleEngineService) SaveConfiguration(chainId string, key string, configuration interface{}) error {
	if chainId != "" {
		ruleEngine, ok := s.Pool.Get(chainId)
		if ok {
			self := ruleEngine.RootRuleChainCtx().Definition()
			if self.RuleChain.Configuration == nil {
				self.RuleChain.Configuration = make(types.Configuration)
			}
			self.RuleChain.Configuration[key] = configuration

			//修改更新时间
			s.fillAdditionalInfo(self)

			if err := ruleEngine.ReloadSelf(ruleEngine.DSL()); err != nil {
				return err
			}
			def, _ := json.Format(ruleEngine.DSL())
			// return s.ruleDao.Save(s.username, chainId, def)
			return s.ruleDao.SaveToDataBase(chainId, def)
		} else {
			return errors.New("not found for" + chainId)
		}
	} else {
		return errors.New("not found for" + chainId)
	}
}

// OnDebug 调试日志
func (s *RuleEngineService) OnDebug(chainId, flowType string, nodeId string, msg types.RuleMsg, relationType string, err error) {
	s.locker.RLock()
	defer s.locker.RUnlock()
	for _, observer := range s.onDebugObserver {
		if observer.chainId == chainId {
			go observer.fn(chainId, flowType, nodeId, msg, relationType, err)
		}
	}
}

func (s *RuleEngineService) AddOnDebugObserver(chainId string, clientId string, fn func(chainId, flowType string, nodeId string, msg types.RuleMsg, relationType string, err error)) {
	s.locker.Lock()
	defer s.locker.Unlock()
	s.onDebugObserver[clientId] = &DebugObserver{
		chainId:  chainId,
		clientId: clientId,
		fn:       fn,
	}
}

func (s *RuleEngineService) RemoveOnDebugObserver(clientId string) {
	s.locker.Lock()
	defer s.locker.Unlock()
	delete(s.onDebugObserver, clientId)
}

func (s *RuleEngineService) DebugData() *RuleChainDebugData {
	return s.ruleChainDebugData
}

// 初始化规则链池
func (s *RuleEngineService) InitRuleGo(logger *log.Logger, workspacePath string) {

	ruleConfig := rulego.NewConfig(types.WithDefaultPool(), types.WithLogger(logger), types.WithNetPool(node_pool.DefaultNodePool))
	//加载自定义配置
	for k, v := range s.config.Global {
		ruleConfig.Properties.PutValue(k, v)
	}
	//加载lua第三方库
	ruleConfig.Properties.PutValue(luaEngine.LoadLuaLibs, s.config.LoadLuaLibs)
	ruleConfig.Properties.PutValue(action.KeyExecNodeWhitelist, s.config.CmdWhiteList)
	ruleConfig.Properties.PutValue(action.KeyWorkDir, s.config.DataDir)
	ruleConfig.OnDebug = func(chainId, flowType string, nodeId string, msg types.RuleMsg, relationType string, err error) {
		var errStr = ""
		if err != nil {
			errStr = err.Error()
		}
		if s.config.Debug {
			logger.Printf("chainId=%s,flowType=%s,nodeId=%s,data=%s,err=%s", chainId, flowType, nodeId, msg.Data, err)
		}
		//把日志记录到内存管理器，用于界面显示
		s.ruleChainDebugData.Add(chainId, nodeId, DebugData{
			Ts: time.Now().UnixMilli(),
			//节点ID
			NodeId: nodeId,
			//流向OUT/IN
			FlowType: flowType,
			//消息
			Msg: msg,
			//关系
			RelationType: relationType,
			//Err 错误
			Err: errStr,
		})
		s.OnDebug(chainId, flowType, nodeId, msg, relationType, err)
	}
	s.ruleConfig = ruleConfig

	//加载js
	jsPath := path.Join(workspacePath, "js")
	err := s.loadJs(jsPath)
	if err != nil {
		logger.Fatal("parser js file error:", err)
	}

	//加载组件插件
	pluginsPath := path.Join(workspacePath, "plugins")
	err = s.loadPlugins(pluginsPath)
	if err != nil {
		logger.Fatal("parser plugin file error:", err)
	}
	allRuleList, aErr := dao.GetAllLoadRegulation()
	if aErr != nil {
		logger.Fatal("parser plugin file error:", aErr)
	}
	ruleStrList := make([]string, 0)
	for _, itme := range allRuleList {
		ruleStrList = append(ruleStrList, itme.RuleConfig)
	}
	// 加载所有持久化规则链
	err = s.loadRulesByPersisted(ruleStrList)
	if err != nil {
		logger.Fatal("parser rule file error:", err)
	}
}

// 加载js
func (s *RuleEngineService) loadJs(folderPath string) error {
	//创建文件夹
	_ = fs.CreateDirs(folderPath)
	//遍历所有文件
	paths, err := fs.GetFilePaths(folderPath + "/*.js")
	if err != nil {
		return err
	}
	for _, file := range paths {
		if b := fs.LoadFile(file); b != nil {
			if p, err := goja.Compile(file, string(b), true); err != nil {
				s.logger.Printf("Compile js file=%s err=%s", file, err.Error())
			} else {
				s.ruleConfig.RegisterUdf(path.Base(file), types.Script{
					Type:    types.Js,
					Content: p,
				})
			}

		}
	}
	return nil
}

// 加载组件插件
func (s *RuleEngineService) loadPlugins(folderPath string) error {
	//创建文件夹
	_ = fs.CreateDirs(folderPath)
	//遍历所有文件
	paths, err := fs.GetFilePaths(folderPath + "/*.so")
	if err != nil {
		return err
	}
	for _, file := range paths {
		if err := rulego.Registry.RegisterPlugin(path.Base(file), file); err != nil {
			s.logger.Printf("load plugin=%s error=%s", file, err.Error())
		}
	}
	return nil
}

// fillAdditionalInfo 填充扩展字段
func (s *RuleEngineService) fillAdditionalInfo(def *types.RuleChain) {
	//修改更新时间
	if def.RuleChain.AdditionalInfo == nil {
		def.RuleChain.AdditionalInfo = make(map[string]interface{})
	}
	def.RuleChain.AdditionalInfo[constants.KeyUsername] = s.username
	nowStr := time.Now().Format("2006/01/02 15:04:05")
	if _, ok := def.RuleChain.AdditionalInfo["createTime"]; !ok {
		def.RuleChain.AdditionalInfo["createTime"] = nowStr
	}
	def.RuleChain.AdditionalInfo["updateTime"] = nowStr
}

/**
 * 加载规则链
 *
 * @folderPath 空路径 为了可以使用Load中的g.ruleEnginePool = engine.NewPool()赋值
 * @ruleList 持久化查出来的所有需要加载的规则
 */
func (s *RuleEngineService) loadRulesByPersisted(ruleList []string) error {
	var err error
	// 遍历所有的需要加载的规则链
	for _, item := range ruleList {
		var ruleTree RuleTree
		json.Unmarshal([]byte(item), &ruleTree)
		if err = s.Load(ruleTree.RuleChain.Id); err != nil {
			s.logger.Printf("load rule chain error: %s", err.Error())
			return err
		}
	}
	// //加载主规则链
	// if mainChainId := s.userSettingDao.Get(constants.SettingKeyMainChainId); mainChainId != "" {
	// 	if err := s.SetMainChainId(mainChainId); err != nil {
	// 		s.logger.Printf("load main rule chain error: %s", err.Error())
	// 	}
	// } else {
	// 	s.logger.Printf("main chain id is empty")
	// }
	return err
}

type RuleTree struct {
	RuleChain RuleChainTree `json:"ruleChain"`
	MetaData  interface{}   `json:"metadata"`
}

type RuleChainTree struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
