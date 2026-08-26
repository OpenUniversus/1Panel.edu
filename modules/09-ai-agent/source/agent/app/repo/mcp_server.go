// =============================================================================
// 模块: AI Agent 智能体 (agent/app/repo/mcp_server.go)
// 文件: mcp_server.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import "github.com/1Panel-dev/1Panel/agent/app/model"

// McpServerRepo (struct)
type McpServerRepo struct {
}

// IMcpServerRepo (interface)
type IMcpServerRepo interface {
	Page(page, size int, opts ...DBOption) (int64, []model.McpServer, error)
	GetFirst(opts ...DBOption) (*model.McpServer, error)
	Create(mcpServer *model.McpServer) error
	Save(mcpServer *model.McpServer) error
	DeleteBy(opts ...DBOption) error
	List(opts ...DBOption) ([]model.McpServer, error)
}

func NewIMcpServerRepo() IMcpServerRepo {
	return &McpServerRepo{}
}

func (m McpServerRepo) Page(page, size int, opts ...DBOption) (int64, []model.McpServer, error) {
	var servers []model.McpServer
	db := getDb(opts...).Model(&model.McpServer{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&servers).Error
	return count, servers, err
}

func (m McpServerRepo) GetFirst(opts ...DBOption) (*model.McpServer, error) {
	var mcpServer model.McpServer
	if err := getDb(opts...).First(&mcpServer).Error; err != nil {
		return nil, err
	}
	return &mcpServer, nil
}

func (m McpServerRepo) List(opts ...DBOption) ([]model.McpServer, error) {
	var mcpServers []model.McpServer
	if err := getDb(opts...).Find(&mcpServers).Error; err != nil {
		return nil, err
	}
	return mcpServers, nil
}

func (m McpServerRepo) Create(mcpServer *model.McpServer) error {
	return getDb().Create(mcpServer).Error
}

func (m McpServerRepo) Save(mcpServer *model.McpServer) error {
	return getDb().Save(mcpServer).Error
}

func (m McpServerRepo) DeleteBy(opts ...DBOption) error {
	return getDb(opts...).Delete(&model.McpServer{}).Error
}
