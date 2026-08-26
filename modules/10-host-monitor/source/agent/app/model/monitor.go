// =============================================================================
// 模块: Monitor 主机监控 (agent/app/model/monitor.go)
// 文件: monitor.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// MonitorBase (struct)
type MonitorBase struct {
	BaseModel
	Cpu         float64     `json:"cpu"`
	TopCPU      string      `json:"topCPU"`
	TopCPUItems interface{} `gorm:"-" json:"topCPUItems"`

	Memory      float64     `json:"memory"`
	TopMem      string      `json:"topMem"`
	TopMemItems interface{} `gorm:"-" json:"topMemItems"`

	LoadUsage float64 `json:"loadUsage"`
	CpuLoad1  float64 `json:"cpuLoad1"`
	CpuLoad5  float64 `json:"cpuLoad5"`
	CpuLoad15 float64 `json:"cpuLoad15"`
}

type MonitorIO struct {
	BaseModel
	Name  string `json:"name"`
	Read  uint64 `json:"read"`
	Write uint64 `json:"write"`
	Count uint64 `json:"count"`
	Time  uint64 `json:"time"`
}

type MonitorNetwork struct {
	BaseModel
	Name string  `json:"name"`
	Up   float64 `json:"up"`
	Down float64 `json:"down"`
}

// MonitorGPU (struct)
type MonitorGPU struct {
	BaseModel
	ProductName   string  `json:"productName"`
	GPUUtil       float64 `json:"gpuUtil"`
	Temperature   float64 `json:"temperature"`
	PowerDraw     float64 `json:"powerDraw"`
	MaxPowerLimit float64 `json:"maxPowerLimit"`
	MemUsed       float64 `json:"memUsed"`
	MemTotal      float64 `json:"memTotal"`
	FanSpeed      int     `json:"fanSpeed"`
	Processes     string  `json:"processes"`
}
