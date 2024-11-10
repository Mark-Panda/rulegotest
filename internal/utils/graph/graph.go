package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"ruleGoProject/internal/dao"
)

// RuleChain 表示一个规则链
type RuleChain struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Node 表示一个节点
type Node struct {
	ID            string      `json:"id"`
	Type          string      `json:"type"`
	Name          string      `json:"name"`
	Configuration interface{} `json:"configuration"`
	DebugMode     bool        `json:"debugMode"`
}

// Connection 表示节点之间的连接
type Connection struct {
	FromID string `json:"fromId"`
	ToID   string `json:"toId"`
	Type   string `json:"type"`
}

// Metadata 表示规则链的元数据
type Metadata struct {
	Nodes       []Node       `json:"nodes"`
	Connections []Connection `json:"connections"`
}

// RuleChainWithMetadata 包含规则链及其元数据
type RuleChainWithMetadata struct {
	RuleChain RuleChain `json:"ruleChain"`
	Metadata  Metadata  `json:"metadata"`
}

// Graph 构建一个有向图来表示规则链中的节点和连接
type Graph struct {
	nodes map[string]bool
	edges map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]bool),
		edges: make(map[string][]string),
	}
}

func (g *Graph) AddNode(nodeID string) {
	g.nodes[nodeID] = true
}

func (g *Graph) AddEdge(from, to string) {
	g.edges[from] = append(g.edges[from], to)
}

func (g *Graph) BuildFromMetadata(metadata Metadata, chainsId string, ruleChains map[string]*RuleChainWithMetadata, visited map[string]bool) error {
	var err error
	// 检查规则是否已经访问过
	if visited[chainsId] {
		return nil
	}
	visited[chainsId] = true
	rootId := chainsId
	identifier := "->"
	g.AddNode(rootId)
	for _, node := range metadata.Nodes {
		rootToId := fmt.Sprintf(`%s%s%s`, rootId, identifier, node.ID)
		if node.Type == "flow" {
			targetId := node.Configuration.(map[string]interface{})["targetId"].(string)
			g.AddNode(targetId)
		} else {
			g.AddNode(rootToId)
		}
	}
	for _, conn := range metadata.Connections {
		isFlow := false
		fromId := fmt.Sprintf(`%s%s%s`, rootId, identifier, conn.FromID)
		g.AddEdge(rootId, fromId)
		rootToId := fmt.Sprintf(`%s%s%s`, rootId, identifier, conn.ToID)
		for _, node := range metadata.Nodes {
			if conn.ToID == node.ID && node.Type == "flow" {
				isFlow = true
				targetId := node.Configuration.(map[string]interface{})["targetId"].(string)
				g.AddEdge(fromId, targetId)
				// 根据targetId查询子规则链 递归
				subMetadata := ruleChains[targetId].Metadata
				err := g.BuildFromMetadata(subMetadata, targetId, ruleChains, visited)
				if err != nil {
					return err
				}
			}
		}
		if !isFlow {
			g.AddEdge(fromId, rootToId)
		}

	}
	return err
}

// 使用DFS检测环并记录冲突节点
func (g *Graph) DetectCycles() (bool, []string) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		if !visited[node] {
			// Mark the current node as visited and add it to the recursion stack
			visited[node] = true
			recStack[node] = true
			path = append(path, node)

			// Recur for all the vertices adjacent to this vertex
			for _, neighbor := range g.edges[node] {
				if !visited[neighbor] && dfs(neighbor) {
					return true
				} else if recStack[neighbor] {
					// If an adjacent node is visited and is in recStack then there's a cycle
					path = append(path, neighbor)
					return true
				}
			}
		}
		// Remove the node from the recursion stack
		recStack[node] = false
		if len(path) > 0 && path[len(path)-1] == node {
			path = path[:len(path)-1]
		}
		return false
	}

	for node := range g.nodes {
		if !visited[node] {
			if dfs(node) {
				// Reverse the path to start from the first node in the cycle
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return true, path
			}
		}
	}
	return false, nil
}

// 使用Kahn's算法进行拓扑排序，检测是否存在环
func (g *Graph) HasCycle() bool {
	inDegree := make(map[string]int)
	queue := []string{}
	topoOrder := []string{}

	// 初始化入度
	for node := range g.nodes {
		inDegree[node] = 0
	}
	for _, toList := range g.edges {
		for _, to := range toList {
			inDegree[to]++
		}
	}

	// 将所有入度为0的节点加入队列
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	// Kahn's算法
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		topoOrder = append(topoOrder, node)

		for _, neighbor := range g.edges[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 如果拓扑排序的结果包含所有节点，则无环
	return len(topoOrder) != len(g.nodes)
}

// 查找所有有关联的规则链
func FindAllRelevanceClue(chainId string, ruleChainsByte []byte, ruleChainsList *[]*RuleChainWithMetadata, chainMap map[string]bool) error {
	var chainRuleChains RuleChainWithMetadata
	if err := json.Unmarshal(ruleChainsByte, &chainRuleChains); err != nil {
		return err
	}

	// 使用指针方式更新切片
	*ruleChainsList = append(*ruleChainsList, &chainRuleChains)

	// 查询出所有关联的规则链
	for _, node := range chainRuleChains.Metadata.Nodes {
		if node.Type == "flow" {
			targetId := node.Configuration.(map[string]interface{})["targetId"].(string)
			if _, ok := chainMap[targetId]; !ok {
				subRuleConfigInfo, err := dao.FindRegulationByRuleChainId(targetId)
				if err != nil {
					return fmt.Errorf("规则链查询异常 %s", chainId)
				}
				if subRuleConfigInfo != nil && subRuleConfigInfo.RuleChainId != "" {
					chainMap[targetId] = true
					err := FindAllRelevanceClue(targetId, []byte(subRuleConfigInfo.RuleConfig), ruleChainsList, chainMap)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// CheckInfiniteLoop 检查死循环
func CheckInfiniteLoop(chainId string, ruleChainsByte []byte) error {
	var err error

	var ruleChainsList []*RuleChainWithMetadata
	chainMap := make(map[string]bool, 0)
	chainMap[chainId] = true
	// 假设这里有一个包含多个规则链的列表
	err = FindAllRelevanceClue(chainId, ruleChainsByte, &ruleChainsList, chainMap)
	if err != nil {
		return err
	}

	// 将规则链转换为map以便快速查找
	ruleChainsMap := make(map[string]*RuleChainWithMetadata)
	for _, rc := range ruleChainsList {
		ruleChainsMap[rc.RuleChain.ID] = rc
	}

	graph := NewGraph()

	// 构建图
	for _, rc := range ruleChainsList {
		err = graph.BuildFromMetadata(rc.Metadata, rc.RuleChain.ID, ruleChainsMap, map[string]bool{})
		if err != nil {
			return err
		}
	}

	if graph.HasCycle() {
		return errors.New("规则链中存在循环引用，请仔细检查规则链信息")
	} else {
		return nil
	}
}
