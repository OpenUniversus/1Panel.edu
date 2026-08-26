// =============================================================================
// 模块: CronJob 定时任务 (agent/app/model/cronjob.go)
// 文件: cronjob.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import (
	"time"
)

// Cronjob (struct)
type Cronjob struct {
	BaseModel

	Name       string `gorm:"not null" json:"name"`
	Type       string `gorm:"not null" json:"type"`
	GroupID    uint   `json:"groupID"`
	SpecCustom bool   `json:"specCustom"`
	Spec       string `gorm:"not null" json:"spec"`

	Executor      string `json:"executor"`
	Command       string `json:"command"`
	ContainerName string `json:"containerName"`
	ScriptMode    string `json:"scriptMode"`
	Script        string `json:"script"`
	User          string `json:"user"`

	ScriptID       uint   `json:"scriptID"`
	Website        string `json:"website"`
	AppID          string `json:"appID"`
	DBType         string `json:"dbType"`
	DBName         string `json:"dbName"`
	URL            string `json:"url"`
	IsDir          bool   `json:"isDir"`
	SourceDir      string `json:"sourceDir"`
	SnapshotRule   string `json:"snapshotRule"`
	ExclusionRules string `json:"exclusionRules"`

	SourceAccountIDs  string `json:"sourceAccountIDs"`
	DownloadAccountID uint   `json:"downloadAccountID"`
	RetryTimes        uint   `json:"retryTimes"`
	Timeout           uint   `json:"timeout"`
	IgnoreErr         bool   `json:"ignoreErr"`
	RetainCopies      uint64 `json:"retainCopies"`
	Args              string `json:"args"`

	IsExecuting bool         `json:"isExecuting"`
	Status      string       `json:"status"`
	EntryIDs    string       `json:"entryIDs"`
	Records     []JobRecords `json:"records"`
	Secret      string       `json:"secret"`

	Config string `json:"config"`
}

// JobRecords (struct)
type JobRecords struct {
	BaseModel

	CronjobID uint      `json:"cronjobID"`
	TaskID    string    `json:"taskID"`
	StartTime time.Time `json:"startTime"`
	Interval  float64   `json:"interval"`
	Records   string    `json:"records"`
	FromLocal bool      `json:"source"`
	File      string    `json:"file"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

type ScriptLibrary struct {
	BaseModel
	Name   string `json:"name" gorm:"not null;"`
	Script string `json:"script" gorm:"not null;"`
}
