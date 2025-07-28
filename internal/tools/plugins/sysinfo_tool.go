package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"ai-ops/internal/util"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// SysInfoTool 系统信息检查工具实现
type SysInfoTool struct{}

func (s *SysInfoTool) Name() string { return "sysinfo" }
func (s *SysInfoTool) Description() string {
	return "系统信息检查工具，支持查询CPU、内存、磁盘使用情况和高资源占用进程"
}

func (s *SysInfoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "要执行的操作",
				"enum":        []string{"cpu", "memory", "disk", "all"},
			},
			"path": map[string]any{
				"type":        "string",
				"description": "磁盘路径（仅在action为disk时使用）",
				"default":     "/",
			},
		},
		"required": []string{"action"},
	}
}

func (s *SysInfoTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return "", util.NewError(util.ErrCodeInvalidParam, "缺少或无效的 action 参数")
	}

	util.Infow("执行系统信息检查", map[string]any{"action": action})

	switch action {
	case "cpu":
		return s.getCPUInfo(ctx)
	case "memory":
		return s.getMemoryInfo(ctx)
	case "disk":
		path := "/"
		if p, exists := args["path"].(string); exists && p != "" {
			path = p
		}
		return s.getDiskInfo(ctx, path)
	case "all":
		return s.getAllInfo(ctx)
	default:
		return "", util.NewError(util.ErrCodeInvalidParam, fmt.Sprintf("不支持的操作: %s", action))
	}
}

// getCPUInfo 获取CPU信息
func (s *SysInfoTool) getCPUInfo(ctx context.Context) (string, error) {
	// 获取CPU使用率
	percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return "", util.NewError(util.ErrCodeInternalErr, fmt.Sprintf("获取CPU使用率失败: %v", err))
	}

	// 获取CPU信息
	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return "", util.NewError(util.ErrCodeInternalErr, fmt.Sprintf("获取CPU信息失败: %v", err))
	}

	// 获取CPU使用率前三的进程
	topCPUProcesses, err := s.getTopCPUProcesses(ctx)
	if err != nil {
		return "", err
	}

	result := map[string]any{
		"cpu_usage_percent": percentages[0],
		"cpu_count":         len(cpuInfo),
		"cpu_info":          cpuInfo[0].ModelName,
		"cpu_cores":         cpuInfo[0].Cores,
		"top_cpu_processes": topCPUProcesses,
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return fmt.Sprintf("CPU信息:\n%s", string(jsonData)), nil
}

// getMemoryInfo 获取内存信息
func (s *SysInfoTool) getMemoryInfo(ctx context.Context) (string, error) {
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return "", util.NewError(util.ErrCodeInternalErr, fmt.Sprintf("获取内存信息失败: %v", err))
	}

	// 获取内存使用率前三的进程
	topMemoryProcesses, err := s.getTopMemoryProcesses(ctx)
	if err != nil {
		return "", err
	}

	result := map[string]any{
		"total_gb":             fmt.Sprintf("%.2f", float64(vmem.Total)/1024/1024/1024),
		"available_gb":         fmt.Sprintf("%.2f", float64(vmem.Available)/1024/1024/1024),
		"used_gb":              fmt.Sprintf("%.2f", float64(vmem.Used)/1024/1024/1024),
		"used_percent":         fmt.Sprintf("%.2f", vmem.UsedPercent),
		"free_gb":              fmt.Sprintf("%.2f", float64(vmem.Free)/1024/1024/1024),
		"top_memory_processes": topMemoryProcesses,
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return fmt.Sprintf("内存信息:\n%s", string(jsonData)), nil
}

// getDiskInfo 获取磁盘信息
func (s *SysInfoTool) getDiskInfo(ctx context.Context, path string) (string, error) {
	usage, err := disk.UsageWithContext(ctx, path)
	if err != nil {
		return "", util.NewError(util.ErrCodeInternalErr, fmt.Sprintf("获取磁盘信息失败: %v", err))
	}

	result := map[string]any{
		"path":         path,
		"total_gb":     fmt.Sprintf("%.2f", float64(usage.Total)/1024/1024/1024),
		"used_gb":      fmt.Sprintf("%.2f", float64(usage.Used)/1024/1024/1024),
		"free_gb":      fmt.Sprintf("%.2f", float64(usage.Free)/1024/1024/1024),
		"used_percent": fmt.Sprintf("%.2f", usage.UsedPercent),
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return fmt.Sprintf("磁盘信息 (%s):\n%s", path, string(jsonData)), nil
}

// ProcessInfo 进程信息结构
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float32 `json:"memory_mb"`
	MemoryPercent float32 `json:"memory_percent"`
}

// getTopCPUProcesses 获取CPU使用率前3的进程
func (s *SysInfoTool) getTopCPUProcesses(ctx context.Context) ([]ProcessInfo, error) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, util.NewError(util.ErrCodeInternalErr, fmt.Sprintf("获取进程列表失败: %v", err))
	}

	var processInfos []ProcessInfo

	for _, p := range processes {
		// 获取进程名称
		name, err := p.NameWithContext(ctx)
		if err != nil {
			continue
		}

		// 获取CPU使用率
		cpuPercent, err := p.CPUPercentWithContext(ctx)
		if err != nil {
			cpuPercent = 0
		}

		// 获取内存信息
		memInfo, err := p.MemoryInfoWithContext(ctx)
		if err != nil {
			continue
		}

		// 获取内存使用百分比
		memPercent, err := p.MemoryPercentWithContext(ctx)
		if err != nil {
			memPercent = 0
		}

		processInfos = append(processInfos, ProcessInfo{
			PID:           p.Pid,
			Name:          name,
			CPUPercent:    cpuPercent,
			MemoryMB:      float32(memInfo.RSS) / 1024 / 1024,
			MemoryPercent: memPercent,
		})
	}

	// 按CPU使用率排序，获取前3个
	sort.Slice(processInfos, func(i, j int) bool {
		return processInfos[i].CPUPercent > processInfos[j].CPUPercent
	})

	if len(processInfos) > 3 {
		processInfos = processInfos[:3]
	}

	return processInfos, nil
}

// getTopMemoryProcesses 获取内存使用率前3的进程
func (s *SysInfoTool) getTopMemoryProcesses(ctx context.Context) ([]ProcessInfo, error) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, util.NewError(util.ErrCodeInternalErr, fmt.Sprintf("获取进程列表失败: %v", err))
	}

	var processInfos []ProcessInfo

	for _, p := range processes {
		// 获取进程名称
		name, err := p.NameWithContext(ctx)
		if err != nil {
			continue
		}

		// 获取CPU使用率
		cpuPercent, err := p.CPUPercentWithContext(ctx)
		if err != nil {
			cpuPercent = 0
		}

		// 获取内存信息
		memInfo, err := p.MemoryInfoWithContext(ctx)
		if err != nil {
			continue
		}

		// 获取内存使用百分比
		memPercent, err := p.MemoryPercentWithContext(ctx)
		if err != nil {
			memPercent = 0
		}

		processInfos = append(processInfos, ProcessInfo{
			PID:           p.Pid,
			Name:          name,
			CPUPercent:    cpuPercent,
			MemoryMB:      float32(memInfo.RSS) / 1024 / 1024,
			MemoryPercent: memPercent,
		})
	}

	// 按内存使用率排序，获取前3个
	sort.Slice(processInfos, func(i, j int) bool {
		return processInfos[i].MemoryPercent > processInfos[j].MemoryPercent
	})

	if len(processInfos) > 3 {
		processInfos = processInfos[:3]
	}

	return processInfos, nil
}

// getAllInfo 获取所有系统信息
func (s *SysInfoTool) getAllInfo(ctx context.Context) (string, error) {
	cpuInfo, err := s.getCPUInfo(ctx)
	if err != nil {
		return "", err
	}

	memInfo, err := s.getMemoryInfo(ctx)
	if err != nil {
		return "", err
	}

	diskInfo, err := s.getDiskInfo(ctx, "/")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s", cpuInfo, memInfo, diskInfo), nil
}

// NewSysInfoTool 创建系统信息工具实例
func NewSysInfoTool() interface{} {
	return &SysInfoTool{}
}
