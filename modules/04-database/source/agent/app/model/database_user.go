// =============================================================================
// 模块: Database 数据库 (agent/app/model/database_user.go)
// 文件: database_user.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// DatabaseUser (struct)
type DatabaseUser struct {
	BaseModel
	Type        string `json:"type" gorm:"not null;uniqueIndex:idx_database_user"`
	Database    string `json:"database" gorm:"not null;uniqueIndex:idx_database_user"`
	Username    string `json:"username" gorm:"not null;uniqueIndex:idx_database_user"`
	Host        string `json:"host" gorm:"uniqueIndex:idx_database_user"`
	Password    string `json:"password"`
	Description string `json:"description"`
	IsDelete    bool   `json:"isDelete"`
}

type DatabaseUserGrant struct {
	BaseModel
	Type     string `json:"type" gorm:"not null;uniqueIndex:idx_database_user_grant"`
	Database string `json:"database" gorm:"not null;uniqueIndex:idx_database_user_grant"`
	DBName   string `json:"dbName" gorm:"not null;uniqueIndex:idx_database_user_grant"`
	Username string `json:"username" gorm:"not null;uniqueIndex:idx_database_user_grant"`
	Host     string `json:"host" gorm:"not null;uniqueIndex:idx_database_user_grant"`
}
