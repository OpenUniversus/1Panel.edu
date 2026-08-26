// =============================================================================
// 模块: Runtime AI 运行时 (agent/app/model/runtime.go)
// 文件: runtime.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import (
	"path"

	"github.com/1Panel-dev/1Panel/agent/global"
)

// Runtime (struct)
type Runtime struct {
	BaseModel
	Name          string `gorm:"not null" json:"name"`
	AppDetailID   uint   `json:"appDetailID"`
	Image         string `json:"image"`
	WorkDir       string `json:"workDir"`
	DockerCompose string `json:"dockerCompose"`
	Env           string `json:"env"`
	Params        string `json:"params"`
	Version       string `gorm:"not null" json:"version"`
	Type          string `gorm:"not null" json:"type"`
	Status        string `gorm:"not null" json:"status"`
	Resource      string `gorm:"not null" json:"resource"`
	Port          string `json:"port"`
	Message       string `json:"message"`
	CodeDir       string `json:"codeDir"`
	ContainerName string `json:"containerName"`
	Remark        string `json:"remark"`
}

func (r *Runtime) GetComposePath() string {
	return path.Join(r.GetPath(), "docker-compose.yml")
}

func (r *Runtime) GetEnvPath() string {
	return path.Join(r.GetPath(), ".env")
}

func (r *Runtime) GetPath() string {
	return path.Join(global.Dir.RuntimeDir, r.Type, r.Name)
}

func (r *Runtime) GetFPMPath() string {
	return path.Join(global.Dir.RuntimeDir, r.Type, r.Name, "conf", "php-fpm.conf")
}

func (r *Runtime) GetPHPPath() string {
	return path.Join(global.Dir.RuntimeDir, r.Type, r.Name, "conf", "php.ini")
}

func (r *Runtime) GetLogPath() string {
	return path.Join(r.GetPath(), "build.log")
}

func (r *Runtime) GetSlowLogPath() string {
	return path.Join(r.GetPath(), "log", "fpm.slow.log")
}
