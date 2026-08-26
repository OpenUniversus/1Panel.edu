// =============================================================================
// 模块: AI Agent 智能体 (agent/app/dto/request/host_tool.go)
// 文件: host_tool.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package request

// HostToolTypeReq (struct)
type HostToolTypeReq struct {
	Type string `json:"type" validate:"required,oneof=supervisord"`
}

type HostToolOperateReq struct {
	Type    string `json:"type" validate:"required,oneof=supervisord"`
	Operate string `json:"operate" validate:"required,oneof=restart start stop"`
}

type HostToolCreate struct {
	Type string `json:"type" validate:"required"`
	SupervisorConfig
}

type SupervisorConfig struct {
	ConfigPath  string `json:"configPath"`
	ServiceName string `json:"serviceName"`
}

type HostToolConfigUpdate struct {
	Type    string `json:"type" validate:"required,oneof=supervisord"`
	Content string `json:"content"`
}

type SupervisorProcessConfig struct {
	Name        string `json:"name"`
	Operate     string `json:"operate"`
	Command     string `json:"command"`
	User        string `json:"user"`
	Dir         string `json:"dir"`
	Numprocs    string `json:"numprocs"`
	AutoRestart string `json:"autoRestart"`
	AutoStart   string `json:"autoStart"`
	Environment string `json:"environment"`
}

// SupervisorProcessFileReq (struct)
type SupervisorProcessFileReq struct {
	Name    string `json:"name" validate:"required"`
	Operate string `json:"operate" validate:"required,oneof=get clear update" `
	Content string `json:"content"`
	File    string `json:"file" validate:"required,oneof=out.log err.log config"`
}

type HostSupervisorProcessFileGetReq struct {
	Name string `json:"name" validate:"required"`
	File string `json:"file" validate:"required,oneof=out.log err.log config"`
}

type HostSupervisorProcessFileOperateReq struct {
	Name    string `json:"name" validate:"required"`
	Operate string `json:"operate" validate:"required,oneof=clear update"`
	Content string `json:"content"`
	File    string `json:"file" validate:"required,oneof=out.log err.log config"`
}
