package service

import (
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/engine"
)

type OwnRuleGo struct {
	ruleEnginePool *engine.Pool
}

// 实例化
func NewRuleGo() *OwnRuleGo {
	return &OwnRuleGo{
		ruleEnginePool: engine.NewPool(),
	}
}

// Engine returns the rule engine pool.
func (g *OwnRuleGo) Engine() *engine.Pool {
	return g.ruleEnginePool
}

// Load loads all rule chain configurations from the specified folder and its subFolders into the rule engine instance pool.
// The rule chain ID is taken from the ruleChain.id specified in the rule chain file.
func (g *OwnRuleGo) Load(folderPath string, opts ...types.RuleEngineOption) error {
	if g.ruleEnginePool == nil {
		g.ruleEnginePool = engine.NewPool()
	}
	return g.ruleEnginePool.Load(folderPath, opts...)
}

// New creates a new RuleEngine and stores it in the RuleGo rule chain pool.
// If the specified id is empty (""), the ruleChain.id from the rule chain file is used.
func (g *OwnRuleGo) New(id string, rootRuleChainSrc []byte, opts ...types.RuleEngineOption) (types.RuleEngine, error) {
	return g.ruleEnginePool.New(id, rootRuleChainSrc, opts...)
}

// Get retrieves a rule engine instance by its ID.
func (g *OwnRuleGo) Get(id string) (types.RuleEngine, bool) {
	return g.ruleEnginePool.Get(id)
}

// Del removes a rule engine instance by its ID.
func (g *OwnRuleGo) Del(id string) {
	g.ruleEnginePool.Del(id)
}

// Stop releases all rule engine instances.
func (g *OwnRuleGo) Stop() {
	g.ruleEnginePool.Stop()
}

// Range iterates over all rule engine instances.
func (g *OwnRuleGo) Range(f func(key, value any) bool) {
	g.ruleEnginePool.Range(f)
}

// Reload reloads all rule engine instances.
func (g *OwnRuleGo) Reload(opts ...types.RuleEngineOption) {
	g.ruleEnginePool.Reload(opts...)
}

// OnMsg calls all rule engine instances to process a message.
// All rule chains in the rule engine instance pool will attempt to process the message.
func (g *OwnRuleGo) OnMsg(msg types.RuleMsg) {
	g.ruleEnginePool.Range(func(key, value any) bool {
		if item, ok := value.(types.RuleEngine); ok {
			item.OnMsg(msg)
		}
		return true
	})
}
