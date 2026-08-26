// =============================================================================
// 模块: Monitor 主机监控 (agent/app/dto/response/host_tool.go)
// 文件: host_tool.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package response

// HostToolRes (struct)
type HostToolRes struct {
	Type   string      `json:"type"`
	Config interface{} `json:"config"`
}

type Supervisor struct {
	ConfigPath  string `json:"configPath"`
	IncludeDir  string `json:"includeDir"`
	LogPath     string `json:"logPath"`
	IsExist     bool   `json:"isExist"`
	Init        bool   `json:"init"`
	Msg         string `json:"msg"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	CtlExist    bool   `json:"ctlExist"`
	ServiceName string `json:"serviceName"`
}

type HostToolConfig struct {
	Content string `json:"content"`
}

type SupervisorProcessConfig struct {
	Name        string          `json:"name"`
	Command     string          `json:"command"`
	User        string          `json:"user"`
	Dir         string          `json:"dir"`
	Numprocs    string          `json:"numprocs"`
	Msg         string          `json:"msg"`
	Status      []ProcessStatus `json:"status"`
	AutoRestart string          `json:"autoRestart"`
	AutoStart   string          `json:"autoStart"`
	Environment string          `json:"environment"`
}

// ProcessStatus (struct)
type ProcessStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	PID    string `json:"PID"`
	Uptime string `json:"uptime"`
	Msg    string `json:"msg"`
}
