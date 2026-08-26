// =============================================================================
// 模块: Terminal 终端 (agent/utils/psutil/process.go)
// 文件: process.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package psutil

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/tklauser/go-sysconf"
)

const defaultClockTicks = 100
const maxProcessCreateTimeSkew = time.Minute

// ============================================================
// ProcessCreateTimeResolver  兼容 LXC 容器的进程启动时间解析器
// ============================================================
// 字段:
//   - procRoot (string) — /proc 根目录（HOST_PROC 注入）
//   - bootTime (int64) — 主机启动时间（秒）
//   - clockTicks (uint64) — 系统时钟频率（SC_CLK_TCK）
// ============================================================
type ProcessCreateTimeResolver struct {
	procRoot   string
	bootTime   int64
	clockTicks uint64
}

// ============================================================
// NewProcessCreateTimeResolver  构造解析器（读 /proc/stat 拿 btime、SC_CLK_TCK）
// ============================================================
// 返回:
//   - *ProcessCreateTimeResolver — 启动时间解析器
// 流程:
//   1. 读 HOST_PROC 注入
//   2. sysconf 拿 SC_CLK_TCK
//   3. 读 /proc/stat 拿 btime
// ============================================================
func NewProcessCreateTimeResolver() *ProcessCreateTimeResolver {
	procRoot := os.Getenv("HOST_PROC")
	if procRoot == "" {
		procRoot = "/proc"
	}

	resolver := &ProcessCreateTimeResolver{
		procRoot:   procRoot,
		clockTicks: defaultClockTicks,
	}
	if clockTicks, err := sysconf.Sysconf(sysconf.SC_CLK_TCK); err == nil && clockTicks > 0 {
		resolver.clockTicks = uint64(clockTicks)
	}
	if bootTime, err := readBootTime(filepath.Join(procRoot, "stat")); err == nil {
		resolver.bootTime = bootTime
	}
	return resolver
}

// CreateTime returns the process start time in milliseconds since Unix epoch.
//
// On some LXC systems, gopsutil combines a container-relative /proc/uptime with
// the host-relative starttime from /proc/<pid>/stat, which can place the start
// time in the future. In that case, reading btime and starttime from /proc keeps
// both values on the same clock base, matching the calculation used by ps.
// ============================================================
// CreateTime  拿进程启动时间（毫秒），LXC 下会回退到 btime+startticks
// ============================================================
// 参数:
//   - proc (*process.Process) — 目标进程
// 返回:
//   - (int64, error) — 启动时间（Unix 毫秒）
// 流程:
//   1. 先用 gopsutil 自己的 CreateTime
//   2. 校验"未来时间"（说明跨 clock 出错）
//   3. 回退方案：btime + /proc/<pid>/stat starttime
// 调用: service.GetProcessInfoByPID -> this
// ============================================================
func (r *ProcessCreateTimeResolver) CreateTime(proc *process.Process) (int64, error) {
	now := time.Now()
	createTime, createTimeErr := proc.CreateTime()
	if createTimeErr == nil && isValidProcessCreateTime(createTime, now) {
		return createTime, nil
	}

	if r.bootTime > 0 && r.clockTicks > 0 {
		statPath := filepath.Join(r.procRoot, strconv.Itoa(int(proc.Pid)), "stat")
		if content, err := os.ReadFile(statPath); err == nil {
			if startTicks, err := parseProcessStartTicks(string(content)); err == nil {
				startMillis := startTicks * 1000 / r.clockTicks
				fallbackTime := r.bootTime*1000 + int64(startMillis)
				if isValidProcessCreateTime(fallbackTime, now) {
					return fallbackTime, nil
				}
			}
		}
	}
	if createTimeErr != nil {
		return 0, fmt.Errorf("resolve process create time: %w", createTimeErr)
	}
	return 0, fmt.Errorf("invalid process create time: %d", createTime)
}

// ============================================================
// isValidProcessCreateTime  校验"启动时间"在合理范围（不是未来）
// ============================================================
// 参数:
//   - createTime (int64) — 毫秒
//   - now (time.Time) — 当前时间
// 返回:
//   - bool — true 表示在 [0, now+1min] 内
// ============================================================
func isValidProcessCreateTime(createTime int64, now time.Time) bool {
	return createTime > 0 && createTime <= now.Add(maxProcessCreateTimeSkew).UnixMilli()
}

// ============================================================
// readBootTime  读 /proc/stat 的 btime 字段（系统启动秒数）
// ============================================================
// 参数:
//   - path (string) — /proc/stat 路径
// 返回:
//   - (int64, error) — 启动秒数
// 流程:
//   1. 打开文件
//   2. 找到 "btime <数值>" 这一行
//   3. 解析秒数
// ============================================================
func readBootTime(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || fields[0] != "btime" {
			continue
		}
		bootTime, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse btime: %w", err)
		}
		if bootTime <= 0 {
			return 0, errors.New("invalid btime")
		}
		return bootTime, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, errors.New("btime not found")
}

// ============================================================
// parseProcessStartTicks  解析 /proc/<pid>/stat 的 starttime（第 22 字段）
// ============================================================
// 参数:
//   - stat (string) — 整个 /proc/<pid>/stat 文件内容
// 返回:
//   - (uint64, error) — 启动后经过的 clock tick 数
// 流程:
//   1. 先按括号切出 comm（第二个字段，含空格）
//   2. 之后从 state 开始数到 starttime（索引 19）
//   3. 解析为无符号整数
// ============================================================
func parseProcessStartTicks(stat string) (uint64, error) {
	// The second field (comm) is wrapped in parentheses and may contain spaces
	// or parentheses, so split only after its closing parenthesis. The remaining
	// fields begin at field 3 (state), making starttime (field 22) index 19.
	commEnd := strings.LastIndex(stat, ")")
	if commEnd == -1 || commEnd+1 >= len(stat) {
		return 0, errors.New("invalid process stat")
	}
	fields := strings.Fields(stat[commEnd+1:])
	const startTimeIndex = 19
	if len(fields) <= startTimeIndex {
		return 0, errors.New("process stat has too few fields")
	}
	return strconv.ParseUint(fields[startTimeIndex], 10, 64)
}
