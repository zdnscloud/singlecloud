package nodeagent

import (
	"sync"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

type NodeAgentManager struct {
	api.DefaultHandler

	lock       sync.Mutex
	nodeAgents map[string]*NodeAgent
}

func New() *NodeAgentManager {
	return &NodeAgentManager{
		nodeAgents: make(map[string]*NodeAgent),
	}
}

func (m *NodeAgentManager) List(ctx *resttypes.Context) interface{} {
	return m.GetNodeAgents()

}

func (m *NodeAgentManager) GetNodeAgents() []*NodeAgent {
	m.lock.Lock()
	defer m.lock.Unlock()

	var nodes []*NodeAgent
	for _, node := range m.nodeAgents {
		nodes = append(nodes, node)
	}
	return nodes
}

func (m *NodeAgentManager) GetNodeAgent(name string) (*NodeAgent, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	agent, ok := m.nodeAgents[name]
	return agent, ok
}

func (m *NodeAgentManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	node := ctx.Object.(*NodeAgent)
	node.SetID(node.Name)
	node.SetType(NodeAgentType)

	m.lock.Lock()
	defer m.lock.Unlock()
	m.nodeAgents[node.Name] = node
	return node, nil
}

func (m *NodeAgentManager) RegisterSchemas(version *resttypes.APIVersion, schemas *resttypes.Schemas) {
	schemas.MustImportAndCustomize(version, NodeAgent{}, m, SetNodeAgentSchema)
}
