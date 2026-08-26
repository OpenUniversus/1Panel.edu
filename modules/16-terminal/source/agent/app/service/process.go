// =============================================================================
// 模块: Terminal 终端 (agent/app/service/process.go)
// 文件: process.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/utils/common"
	agentPsutil "github.com/1Panel-dev/1Panel/agent/utils/psutil"
	"github.com/1Panel-dev/1Panel/agent/utils/websocket"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// ============================================================
// ProcessService  进程服务（杀进程 / 查监听 / 查详情）
// ============================================================
// 方法:
//   - StopProcess — 杀进程
//   - GetProcessInfoByPID — 查单个进程详情
//   - GetListeningProcess — 查监听端口的进程
// ============================================================
type ProcessService struct{}

// ============================================================
// IProcessService  进程服务的接口（方便注入和测试）
// ============================================================
// 方法:
//   - StopProcess(req) — 杀进程
//   - GetProcessInfoByPID(pid) — 查进程详情
//   - GetListeningProcess(c) — 查监听端口进程
// ============================================================
type IProcessService interface {
	StopProcess(req request.ProcessReq) error
	GetProcessInfoByPID(pid int32) (*websocket.PsProcessData, error)
	GetListeningProcess(c context.Context) ([]ListeningProcess, error)
}

// ============================================================
// NewIProcessService  构造一个 IProcessService 实例
// ============================================================
// 返回:
//   - IProcessService — 进程服务接口实现
// ============================================================
func NewIProcessService() IProcessService {
	return &ProcessService{}
}

// ============================================================
// StopProcess  根据 PID 杀进程（用 gopsutil 的 Kill）
// ============================================================
// 参数:
//   - req (request.ProcessReq) — 含 PID
// 返回:
//   - error — 失败原因
// 流程:
//   1. 拿到进程对象
//   2. 调 proc.Kill()
// 调用: api/v2.StopProcess -> this
// ============================================================
func (ps *ProcessService) StopProcess(req request.ProcessReq) error {
	proc, err := process.NewProcess(req.PID)
	if err != nil {
		return err
	}
	if err := proc.Kill(); err != nil {
		return err
	}
	return nil
}

// ============================================================
// ListeningProcess  正在监听端口的进程聚合信息
// ============================================================
// 字段:
//   - PID (int32) — 进程 ID
//   - Port (map[uint32]struct{}) — 这个进程占用的所有端口（去重）
//   - Protocol (uint32) — 协议（TCP/UDP）
//   - Name (string) — 进程名
// ============================================================
type ListeningProcess struct {
	PID      int32
	Port     map[uint32]struct{}
	Protocol uint32
	Name     string
}

// ============================================================
// GetListeningProcess  列出所有"正在监听端口"的进程
// ============================================================
// 参数:
//   - c (context.Context) — 上下文（用于取消和超时）
// 返回:
//   - ([]ListeningProcess, error) — 监听进程列表
// 流程:
//   1. 用 gopsutil 拉全部网络连接
//   2. 过滤出 LISTEN 状态的 TCP / UDP 端口
//   3. 按 (PID, 协议) 聚合去重
//   4. 读进程名并打包
// 调用: api/v2.GetListeningProcess -> this
// ============================================================
func (ps *ProcessService) GetListeningProcess(c context.Context) ([]ListeningProcess, error) {
	conn, err := net.ConnectionsMaxWithContext(c, "inet", 32768)
	if err != nil {
		return nil, err
	}
	// One cache entry per (PID, socket type) so TCP and UDP sockets are not merged under one Protocol.
	type procKey struct {
		pid      int32
		protocol uint32
	}
	procCache := make(map[procKey]ListeningProcess, 64)

	for _, conn := range conn {
		if conn.Pid == 0 {
			continue
		}

		if (conn.Status == "LISTEN" && conn.Type == syscall.SOCK_STREAM) || (conn.Type == syscall.SOCK_DGRAM && conn.Raddr.Port == 0) {
			key := procKey{pid: conn.Pid, protocol: conn.Type}
			if _, exists := procCache[key]; !exists {
				proc, err := process.NewProcess(conn.Pid)
				if err != nil {
					continue
				}
				procData := ListeningProcess{
					PID: conn.Pid,
				}
				procData.Name, _ = proc.Name()
				procData.Port = make(map[uint32]struct{})
				procData.Port[conn.Laddr.Port] = struct{}{}
				procData.Protocol = conn.Type
				procCache[key] = procData
			} else {
				p := procCache[key]
				p.Port[conn.Laddr.Port] = struct{}{}
				procCache[key] = p
			}
		}
	}

	procs := make([]ListeningProcess, 0, len(procCache))
	for _, proc := range procCache {
		procs = append(procs, proc)
	}

	return procs, nil
}

// ============================================================
// GetProcessInfoByPID  按 PID 查进程的完整快照（CPU/内存/连接/IO/环境变量等）
// ============================================================
// 参数:
//   - pid (int32) — 进程 ID
// 返回:
//   - (*websocket.PsProcessData, error) — 完整进程信息
// 流程:
//   1. 拿到进程对象，确认在跑
//   2. 依次读名字、父 PID、用户、状态、启动时间、线程数
//   3. 读网络连接、CPU、IO、命令行
//   4. 读 /proc 拿详细内存
//   5. 读环境变量、打开的文件
// 调用: api/v2.GetProcessInfoByPID -> this
// ============================================================
func (ps *ProcessService) GetProcessInfoByPID(pid int32) (*websocket.PsProcessData, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("get process info by pid %v: %v", pid, err)
	}

	exists, err := p.IsRunning()
	if err != nil || !exists {
		return nil, fmt.Errorf("process %v is not running", pid)
	}

	data := &websocket.PsProcessData{
		PID: pid,
	}

	if name, err := p.Name(); err == nil {
		data.Name = name
	}

	if ppid, err := p.Ppid(); err == nil {
		data.PPID = ppid
	}

	if username, err := p.Username(); err == nil {
		data.Username = username
	}

	if status, err := p.Status(); err == nil {
		if len(status) > 0 {
			data.Status = status[0]
		}
	}

	if createTime, err := agentPsutil.NewProcessCreateTimeResolver().CreateTime(p); err == nil {
		data.StartTime = time.Unix(createTime/1000, 0).Format("2006-01-02 15:04:05")
	}

	if numThreads, err := p.NumThreads(); err == nil {
		data.NumThreads = numThreads
	}

	if connections, err := p.Connections(); err == nil {
		data.NumConnections = len(connections)

		var connects []websocket.ProcessConnect
		for _, conn := range connections {
			pc := websocket.ProcessConnect{
				Status: conn.Status,
				Laddr:  conn.Laddr,
				Raddr:  conn.Raddr,
				PID:    pid,
				Name:   data.Name,
			}
			connects = append(connects, pc)
		}
		data.Connects = connects
	}

	if cpuPercent, err := p.CPUPercent(); err == nil {
		data.CpuValue = cpuPercent
		data.CpuPercent = fmt.Sprintf("%.2f%%", cpuPercent)
	}

	if ioCounters, err := p.IOCounters(); err == nil {
		data.DiskRead = common.FormatBytes(ioCounters.ReadBytes)
		data.DiskWrite = common.FormatBytes(ioCounters.WriteBytes)
	}

	if cmdline, err := p.Cmdline(); err == nil {
		data.CmdLine = cmdline
	}

	if memDetail, err := getMemoryDetail(p.Pid); err == nil {
		data.Rss = common.FormatBytes(memDetail.RSS)
		data.VMS = common.FormatBytes(memDetail.VMS)
		data.HWM = common.FormatBytes(memDetail.HWM)
		data.Data = common.FormatBytes(memDetail.Data)
		data.Stack = common.FormatBytes(memDetail.Stack)
		data.Locked = common.FormatBytes(memDetail.Locked)
		data.Swap = common.FormatBytes(memDetail.Swap)
		data.Dirty = common.FormatBytes(memDetail.Dirty)
		data.RssValue = memDetail.RSS
		data.PSS = common.FormatBytes(memDetail.PSS)
		data.USS = common.FormatBytes(memDetail.USS)
		data.Shared = common.FormatBytes(memDetail.Shared)
		data.Text = common.FormatBytes(memDetail.Text)
	}

	if envs, err := p.Environ(); err == nil {
		data.Envs = envs
	}

	if openFiles, err := p.OpenFiles(); err == nil {
		data.OpenFiles = openFiles
	}

	return data, nil
}

// ============================================================
// MemoryDetail  从 /proc 读出的进程内存细分
// ============================================================
// 字段:
//   - RSS (uint64) — 实际占用的物理内存
//   - VMS (uint64) — 虚拟内存大小
//   - HWM (uint64) — 进程历史峰值物理内存
//   - Data (uint64) — 数据段
//   - Stack (uint64) — 栈
//   - Locked (uint64) — 被锁住的内存
//   - Swap (uint64) — 换出到磁盘的部分
//   - PSS (uint64) — 按比例分摊的物理内存
//   - USS (uint64) — 进程独占的物理内存
//   - Shared (uint64) — 共享内存
//   - Text (uint64) — 代码段
//   - Dirty (uint64) — 脏页
// ============================================================
type MemoryDetail struct {
	RSS    uint64
	VMS    uint64
	HWM    uint64
	Data   uint64
	Stack  uint64
	Locked uint64
	Swap   uint64

	PSS    uint64
	USS    uint64
	Shared uint64
	Text   uint64
	Dirty  uint64
}

// ============================================================
// getMemoryDetail  从 /proc 读进程的内存细分（PSS/USS 等）
// ============================================================
// 参数:
//   - pid (int32) — 进程 ID
// 返回:
//   - (*MemoryDetail, error) — 内存细节
// 流程:
//   1. 先试 /proc/<pid>/smaps_rollup（汇总，开销小）
//   2. 拿不到再降级到 /proc/<pid>/smaps（逐段加总）
// 调用: GetProcessInfoByPID -> this
// ============================================================
func getMemoryDetail(pid int32) (*MemoryDetail, error) {
	mem := &MemoryDetail{}

	if err := readStatus(pid, mem); err != nil {
		return nil, err
	}

	if err := readSmapsRollup(pid, mem); err != nil {
		if err := readSmaps(pid, mem); err != nil {
			return nil, err
		}
	}
	return mem, nil
}

// ============================================================
// readStatus  解析 /proc/<pid>/status 拿 VmRSS / VmSize 等基础内存字段
// ============================================================
// 参数:
//   - pid (int32) — 进程 ID
//   - mem (*MemoryDetail) — 写入的目标结构
// 返回:
//   - error — 读/解析失败
// 流程:
//   1. 打开 /proc/<pid>/status
//   2. 逐行扫描，把 VmRSS / VmSize 等映射到 mem 字段
// 调用: getMemoryDetail -> this
// ============================================================
func readStatus(pid int32, mem *MemoryDetail) error {
	filePath := fmt.Sprintf("/proc/%d/status", pid)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024

		switch key {
		case "VmRSS":
			mem.RSS = value
		case "VmSize":
			mem.VMS = value
		case "VmData":
			mem.Data = value
		case "VmSwap":
			mem.Swap = value
		case "VmExe":
			mem.Text = value
		case "RssShmem":
			mem.Shared = value
		case "VmHWM":
			mem.HWM = value
		case "VmStk":
			mem.Stack = value
		case "VmLck":
			mem.Locked = value
		}
	}

	return scanner.Err()
}

// ============================================================
// readSmapsRollup  解析 /proc/<pid>/smaps_rollup 拿 PSS/USS 汇总
// ============================================================
// 参数:
//   - pid (int32) — 进程 ID
//   - mem (*MemoryDetail) — 写入目标
// 返回:
//   - error — 读/解析失败
// 流程:
//   1. 打开 smaps_rollup
//   2. 逐行扫描，提取 Pss / Private_Clean / Private_Dirty / Shared 字段
// 调用: getMemoryDetail -> this
// ============================================================
func readSmapsRollup(pid int32, mem *MemoryDetail) error {
	filePath := fmt.Sprintf("/proc/%d/smaps_rollup", pid)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024

		switch key {
		case "Pss":
			mem.PSS = value
		case "Private_Clean", "Private_Dirty":
			mem.USS += value
		case "Shared_Clean", "Shared_Dirty":
			if mem.Shared == 0 {
				mem.Shared = value
			}
		}
	}

	return scanner.Err()
}

// ============================================================
// readSmaps  解析 /proc/<pid>/smaps 拿 PSS/USS（rollup 不可用时回退）
// ============================================================
// 参数:
//   - pid (int32) — 进程 ID
//   - mem (*MemoryDetail) — 写入目标
// 返回:
//   - error — 读/解析失败
// 流程:
//   1. 打开 smaps（按段输出）
//   2. 累加每段的 PSS / Private / Shared
// 调用: getMemoryDetail -> this（回退路径）
// ============================================================
func readSmaps(pid int32, mem *MemoryDetail) error {
	filePath := fmt.Sprintf("/proc/%d/smaps", pid)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024

		switch key {
		case "Pss":
			mem.PSS += value
		case "Private_Clean", "Private_Dirty":
			mem.USS += value
		case "Shared_Clean", "Shared_Dirty":
			if mem.Shared == 0 {
				mem.Shared += value
			}
		}
	}

	return scanner.Err()
}
