// =============================================================================
// 模块: CronJob 定时任务 (agent/app/model/task.go)
// 文件: task.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import "time"

// Task (struct)
type Task struct {
	ID             string    `gorm:"primarykey;" json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Operate        string    `json:"operate"`
	LogFile        string    `json:"logFile"`
	Status         string    `json:"status"`
	ErrorMsg       string    `json:"errorMsg"`
	OperationLogID uint      `json:"operationLogID"`
	ResourceID     uint      `json:"resourceID"`
	CurrentStep    string    `json:"currentStep"`
	EndAt          time.Time `json:"endAt"`
	CreatedAt      time.Time `json:"createdAt"`
}
