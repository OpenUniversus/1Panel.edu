// =============================================================================
// 模块: AI Agent 智能体 (agent/app/repo/agent_account_model.go)
// 文件: agent_account_model.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import "github.com/1Panel-dev/1Panel/agent/app/model"

// AgentAccountModelRepo (struct)
type AgentAccountModelRepo struct{}

// IAgentAccountModelRepo (interface)
type IAgentAccountModelRepo interface {
	List(opts ...DBOption) ([]model.AgentAccountModel, error)
	GetFirst(opts ...DBOption) (*model.AgentAccountModel, error)
	Create(item *model.AgentAccountModel) error
	Save(item *model.AgentAccountModel) error
	DeleteByID(id uint) error
	Delete(opts ...DBOption) error
}

func NewIAgentAccountModelRepo() IAgentAccountModelRepo {
	return &AgentAccountModelRepo{}
}

func (a AgentAccountModelRepo) List(opts ...DBOption) ([]model.AgentAccountModel, error) {
	var list []model.AgentAccountModel
	if err := getDb(opts...).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (a AgentAccountModelRepo) GetFirst(opts ...DBOption) (*model.AgentAccountModel, error) {
	var item model.AgentAccountModel
	if err := getDb(opts...).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (a AgentAccountModelRepo) Create(item *model.AgentAccountModel) error {
	return getDb().Create(item).Error
}

func (a AgentAccountModelRepo) Save(item *model.AgentAccountModel) error {
	return getDb().Save(item).Error
}

func (a AgentAccountModelRepo) DeleteByID(id uint) error {
	return getDb().Delete(&model.AgentAccountModel{}, id).Error
}

func (a AgentAccountModelRepo) Delete(opts ...DBOption) error {
	return getDb(opts...).Delete(&model.AgentAccountModel{}).Error
}
