// =============================================================================
// 模块: Database 数据库 (agent/app/model/database_mongodb.go)
// 文件: database_mongodb.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// DatabaseMongodb (struct)
type DatabaseMongodb struct {
	BaseModel
	Name        string `json:"name" gorm:"not null"`
	From        string `json:"from" gorm:"not null;default:local"`
	MongodbName string `json:"mongodbName" gorm:"not null"`
	Username    string `json:"username" gorm:"not null"`
	Password    string `json:"password" gorm:"not null"`
	IsDelete    bool   `json:"isDelete"`
	Description string `json:"description"`
}
