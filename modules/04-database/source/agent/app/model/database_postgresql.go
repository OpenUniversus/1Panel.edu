// =============================================================================
// 模块: Database 数据库 (agent/app/model/database_postgresql.go)
// 文件: database_postgresql.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// DatabasePostgresql (struct)
type DatabasePostgresql struct {
	BaseModel
	Name           string `json:"name" gorm:"not null"`
	From           string `json:"from" gorm:"not null;default:local"`
	PostgresqlName string `json:"postgresqlName" gorm:"not null"`
	Format         string `json:"format" gorm:"not null"`
	Username       string `json:"username" gorm:"not null"`
	Password       string `json:"password" gorm:"not null"`
	SuperUser      bool   `json:"superUser"`
	IsDelete       bool   `json:"isDelete"`
	Description    string `json:"description"`
}
