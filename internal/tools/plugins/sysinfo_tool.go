package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// SysInfoTool 系统信息工具
type SysInfoTool struct{}

// NewSysInfoTool 创建系统信息工具实例
func NewSysInfoTool() interface{} {
	return &SysInfoTool{}
}

// ID 返回工具唯一标识符
func (t *SysInfoTool) ID() string {
	return "sysinfo"
}

// Name 返回工具名称
func (t *SysInfoTool) Name() string {
	return "系统信息工具"
}

// Type 返回工具类型
func (t *SysInfoTool) Type() string {
	return "system"
}

// Description 返回工具描述
func (t *SysInfoTool) Description() string {
	return "获取系统信息，包括CPU、内存、磁盘、网络等监控数据"
}

// Parameters 返回工具参数schema
func (t *SysInfoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"cpu", "memory", "disk", "network", "load", "overview"},
				"description": "查询类型：cpu(CPU信息)、memory(内存)、disk(磁盘)、network(网络)、load(负载)、overview(概览)",
			},
			"detail": map[string]any{
				"type":        "boolean",
				"default":     false,
				"description": "是否显示详细信息",
			},
		},
		"required": []string{"action"},
	}
}

// Execute 执行工具
func (t *SysInfoTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	action, ok := args["action"].(string)
	if !ok {
		return "", fmt.Errorf("参数 action 必须是字符串类型")
	}

	detail := false
	if d, exists := args["detail"]; exists {
		if b, ok := d.(bool); ok {
			detail = b
		}
	}

	switch action {
	case "cpu":
		return t.getCPUInfo(ctx, detail)
	case "memory":
		return t.getMemoryInfo(ctx, detail)
	case "disk":
		return t.getDiskInfo(ctx, detail)
	case "network":
		return t.getNetworkInfo(ctx, detail)
	case "load":
		return t.getLoadInfo(ctx, detail)
	case "overview":
		return t.getOverview(ctx)
	default:
		return "", fmt.Errorf("不支持的操作类型: %s", action)
	}
}

// getCPUInfo 获取CPU信息
func (t *SysInfoTool) getCPUInfo(ctx context.Context, detail bool) (string, error) {
	// CPU 使用率
	percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return "", fmt.Errorf("获取CPU使用率失败: %v", err)
	}

	result := strings.Builder{}
	result.WriteString("🔥 CPU 信息\n")
	result.WriteString(fmt.Sprintf("总体使用率: %.1f%%\n", percentages[0]))

	if detail {
		// CPU 详细信息
		cpuInfo, err := cpu.InfoWithContext(ctx)
		if err == nil && len(cpuInfo) > 0 {
			info := cpuInfo[0]
			result.WriteString(fmt.Sprintf("CPU 型号: %s\n", info.ModelName))
			result.WriteString(fmt.Sprintf("核心数: %d\n", info.Cores))
			result.WriteString(fmt.Sprintf("频率: %.0f MHz\n", info.Mhz))
		}

		// 各核心使用率
		perCPU, err := cpu.PercentWithContext(ctx, time.Second, true)
		if err == nil {
			result.WriteString("\n各核心使用率:\n")
			for i, usage := range perCPU {
				result.WriteString(fmt.Sprintf("CPU%d: %.1f%%\n", i, usage))
			}
		}
	}

	return result.String(), nil
}

// getMemoryInfo 获取内存信息
func (t *SysInfoTool) getMemoryInfo(ctx context.Context, detail bool) (string, error) {
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("获取内存信息失败: %v", err)
	}

	result := strings.Builder{}
	result.WriteString("💾 内存信息\n")
	result.WriteString(fmt.Sprintf("总内存: %s\n", formatBytes(vmem.Total)))
	result.WriteString(fmt.Sprintf("已用内存: %s (%.1f%%)\n", formatBytes(vmem.Used), vmem.UsedPercent))
	result.WriteString(fmt.Sprintf("可用内存: %s\n", formatBytes(vmem.Available)))

	if detail {
		result.WriteString(fmt.Sprintf("空闲内存: %s\n", formatBytes(vmem.Free)))
		result.WriteString(fmt.Sprintf("缓存: %s\n", formatBytes(vmem.Cached)))
		result.WriteString(fmt.Sprintf("缓冲区: %s\n", formatBytes(vmem.Buffers)))

		// 交换分区信息
		swap, err := mem.SwapMemoryWithContext(ctx)
		if err == nil {
			result.WriteString("\n交换分区:\n")
			result.WriteString(fmt.Sprintf("总大小: %s\n", formatBytes(swap.Total)))
			result.WriteString(fmt.Sprintf("已用: %s (%.1f%%)\n", formatBytes(swap.Used), swap.UsedPercent))
		}
	}

	return result.String(), nil
}

// getDiskInfo 获取磁盘信息
func (t *SysInfoTool) getDiskInfo(ctx context.Context, detail bool) (string, error) {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return "", fmt.Errorf("获取磁盘分区失败: %v", err)
	}

	result := strings.Builder{}
	result.WriteString("💿 磁盘信息\n")

	for _, partition := range partitions {
		usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			continue
		}

		result.WriteString(fmt.Sprintf("\n📁 %s (%s)\n", partition.Mountpoint, partition.Device))
		result.WriteString(fmt.Sprintf("总大小: %s\n", formatBytes(usage.Total)))
		result.WriteString(fmt.Sprintf("已用: %s (%.1f%%)\n", formatBytes(usage.Used), usage.UsedPercent))
		result.WriteString(fmt.Sprintf("可用: %s\n", formatBytes(usage.Free)))

		if detail {
			result.WriteString(fmt.Sprintf("文件系统: %s\n", partition.Fstype))
			result.WriteString(fmt.Sprintf("Inode总数: %d\n", usage.InodesTotal))
			result.WriteString(fmt.Sprintf("Inode已用: %d (%.1f%%)\n", usage.InodesUsed, usage.InodesUsedPercent))
		}
	}

	return result.String(), nil
}

// getNetworkInfo 获取网络信息
func (t *SysInfoTool) getNetworkInfo(ctx context.Context, detail bool) (string, error) {
	interfaces, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("获取网络接口失败: %v", err)
	}

	result := strings.Builder{}
	result.WriteString("🌐 网络信息\n")

	for _, iface := range interfaces {
		// 跳过回环接口（除非是详细模式）
		if !detail && iface.Name == "lo" {
			continue
		}

		result.WriteString(fmt.Sprintf("\n📡 %s\n", iface.Name))
		if len(iface.Addrs) > 0 {
			for _, addr := range iface.Addrs {
				result.WriteString(fmt.Sprintf("地址: %s\n", addr.Addr))
			}
		}

		if detail {
			result.WriteString(fmt.Sprintf("MAC: %s\n", iface.HardwareAddr))
			result.WriteString(fmt.Sprintf("MTU: %d\n", iface.MTU))

			// 获取网络统计
			stats, err := net.IOCountersWithContext(ctx, true)
			if err == nil {
				for _, stat := range stats {
					if stat.Name == iface.Name {
						result.WriteString(fmt.Sprintf("发送: %s\n", formatBytes(stat.BytesSent)))
						result.WriteString(fmt.Sprintf("接收: %s\n", formatBytes(stat.BytesRecv)))
						break
					}
				}
			}
		}
	}

	return result.String(), nil
}

// getLoadInfo 获取系统负载信息
func (t *SysInfoTool) getLoadInfo(ctx context.Context, detail bool) (string, error) {
	loadAvg, err := load.AvgWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("获取系统负载失败: %v", err)
	}

	result := strings.Builder{}
	result.WriteString("⚡ 系统负载\n")
	result.WriteString(fmt.Sprintf("1分钟: %.2f\n", loadAvg.Load1))
	result.WriteString(fmt.Sprintf("5分钟: %.2f\n", loadAvg.Load5))
	result.WriteString(fmt.Sprintf("15分钟: %.2f\n", loadAvg.Load15))

	if detail {
		hostInfo, err := host.InfoWithContext(ctx)
		if err == nil {
			result.WriteString("\n主机信息:\n")
			result.WriteString(fmt.Sprintf("主机名: %s\n", hostInfo.Hostname))
			result.WriteString(fmt.Sprintf("操作系统: %s %s\n", hostInfo.OS, hostInfo.PlatformVersion))
			result.WriteString(fmt.Sprintf("架构: %s\n", hostInfo.KernelArch))
			result.WriteString(fmt.Sprintf("运行时间: %s\n", formatDuration(time.Duration(hostInfo.Uptime)*time.Second)))
		}
	}

	return result.String(), nil
}

// getOverview 获取系统概览
func (t *SysInfoTool) getOverview(ctx context.Context) (string, error) {
	result := strings.Builder{}
	result.WriteString("📊 系统概览\n")

	// CPU
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		result.WriteString(fmt.Sprintf("🔥 CPU: %.1f%%\n", cpuPercent[0]))
	}

	// 内存
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		result.WriteString(fmt.Sprintf("💾 内存: %s / %s (%.1f%%)\n",
			formatBytes(vmem.Used), formatBytes(vmem.Total), vmem.UsedPercent))
	}

	// 负载
	loadAvg, err := load.AvgWithContext(ctx)
	if err == nil {
		result.WriteString(fmt.Sprintf("⚡ 负载: %.2f, %.2f, %.2f\n",
			loadAvg.Load1, loadAvg.Load5, loadAvg.Load15))
	}

	// 主机信息
	hostInfo, err := host.InfoWithContext(ctx)
	if err == nil {
		result.WriteString(fmt.Sprintf("🖥️  主机: %s (%s)\n", hostInfo.Hostname, hostInfo.OS))
		result.WriteString(fmt.Sprintf("⏰ 运行时间: %s\n", formatDuration(time.Duration(hostInfo.Uptime)*time.Second)))
	}

	return result.String(), nil
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration 格式化时间段
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	} else {
		return fmt.Sprintf("%d分钟", minutes)
	}
}
