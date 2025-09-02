package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"ai-ops/internal/util"
	"ai-ops/internal/util/errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultMCPManager 默认MCP管理器实现
type DefaultMCPManager struct {
	settings *MCPSettings
	sessions map[string]*mcp.ClientSession
	mutex    sync.RWMutex
}

// NewMCPManager 创建新的MCP管理器
func NewMCPManager() MCPManager {
	return &DefaultMCPManager{
		sessions: make(map[string]*mcp.ClientSession),
	}
}

// LoadSettings 加载MCP配置
func (m *DefaultMCPManager) LoadSettings(configPath string) error {
	util.Debugw("加载MCP配置", map[string]any{
		"config_path": configPath,
	})

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return errors.WrapErrorWithDetails(errors.ErrCodeConfigLoadFailed,
			"读取MCP配置文件失败", err,
			fmt.Sprintf("配置文件路径: %s", configPath))
	}

	// 解析JSON配置
	var settings MCPSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		util.Errorw("JSON解析失败，原始数据内容", map[string]any{
			"config_path":       configPath,
			"json_data_len":     len(data),
			"json_data_preview": string(data[:min(len(data), 200)]) + "...",
			"error":             err.Error(),
		})
		return errors.WrapErrorWithDetails(errors.ErrCodeConfigParseFailed,
			"解析MCP配置文件失败", err,
			fmt.Sprintf("配置文件路径: %s", configPath))
	}

	util.Debugw("JSON解析成功", map[string]any{
		"config_path":   configPath,
		"json_data_len": len(data),
	})

	m.mutex.Lock()
	m.settings = &settings
	m.mutex.Unlock()

	util.Infow("MCP配置加载成功", map[string]any{
		"server_count": len(settings.MCPServers),
	})

	return nil
}

// InitializeClients 初始化所有MCP客户端
func (m *DefaultMCPManager) InitializeClients(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.settings == nil {
		return errors.NewError(errors.ErrCodeMCPNotConfigured, "MCP配置未加载")
	}

	util.Debugw("初始化MCP客户端", map[string]any{
		"server_count": len(m.settings.MCPServers),
	})

	// 清理现有会话
	for _, session := range m.sessions {
		session.Close()
	}
	m.sessions = make(map[string]*mcp.ClientSession)

	// 初始化每个服务器的会话
	for serverName, config := range m.settings.MCPServers {
		if config.Disabled {
			util.Debugw("跳过已禁用的MCP服务器", map[string]any{
				"server_name": serverName,
			})
			continue
		}

		if config.Type != "stdio" && config.Type != "" {
			err := errors.NewErrorWithDetails(errors.ErrCodeMCPConnectionFailed,
				"不支持的MCP服务器类型",
				fmt.Sprintf("服务器名称: %s, 类型: %s", serverName, config.Type))
			errors.HandleError(err)
			continue
		}

		cmd := exec.CommandContext(ctx, config.Command, config.Args...)
		if config.Env != nil {
			cmd.Env = os.Environ()
			for k, v := range config.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}

		client := mcp.NewClient(&mcp.Implementation{Name: "ai-ops", Version: "1.0.0"}, nil)
		transport := mcp.NewCommandTransport(cmd)

		timeout := time.Duration(config.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second // 默认30秒
		}
		connectCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		session, err := client.Connect(connectCtx, transport)
		if err != nil {
			wrappedErr := errors.WrapErrorWithDetails(errors.ErrCodeMCPConnectionFailed,
				"MCP客户端连接失败", err,
				fmt.Sprintf("服务器名称: %s", serverName))
			errors.HandleError(wrappedErr)
			util.Errorw("MCP服务器连接失败", map[string]any{
				"server_name": serverName,
				"command":     config.Command,
				"args":        config.Args,
				"type":        config.Type,
				"timeout":     timeout,
				"error":       err.Error(),
			})
			continue
		}

		m.sessions[serverName] = session
	}

	util.Infow("MCP客户端初始化完成", map[string]any{
		"connected_count": len(m.sessions),
	})

	return nil
}

// GetClients 获取所有客户端
func (m *DefaultMCPManager) GetClients() map[string]*mcp.ClientSession {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 返回副本以避免并发问题
	sessions := make(map[string]*mcp.ClientSession)
	for name, session := range m.sessions {
		sessions[name] = session
	}
	return sessions
}

// GetClient 根据名称获取客户端
func (m *DefaultMCPManager) GetClient(name string) (*mcp.ClientSession, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.sessions[name]
	return session, exists
}

// Shutdown 关闭所有客户端
func (m *DefaultMCPManager) Shutdown() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	util.Debugw("关闭所有MCP客户端", map[string]any{
		"client_count": len(m.sessions),
	})

	var lastErr error
	for serverName, session := range m.sessions {
		if err := session.Close(); err != nil {
			wrappedErr := errors.WrapErrorWithDetails(errors.ErrCodeMCPConnectionFailed,
				"关闭MCP客户端失败", err,
				fmt.Sprintf("服务器名称: %s", serverName))
			errors.HandleError(wrappedErr)
			lastErr = wrappedErr
		}
	}

	m.sessions = make(map[string]*mcp.ClientSession)
	return lastErr
}
