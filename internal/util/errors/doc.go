// Package errors 提供统一的错误处理系统
//
// 这个包实现了一个简化的错误处理框架，包括：
// - 统一的错误代码定义
// - 基本的错误创建方法
// - 简化的错误处理器
// - HTTP 错误处理中间件
//
// 基本用法：
//
//  1. 创建错误：
//     err := errors.NewError(errors.ErrCodeConfigNotFound, "配置文件未找到")
//     errWithDetails := errors.NewErrorWithDetails(errors.ErrCodeConfigNotFound, "配置文件未找到", "路径：/path/to/config.toml")
//     wrappedErr := errors.WrapError(errors.ErrCodeConfigInvalid, "配置文件无效", originalErr)
//
//  2. 使用预定义错误创建函数：
//     configErr := errors.NewConfigError("配置文件加载失败")
//     networkErr := errors.NewNetworkError("网络连接超时")
//     aiErr := errors.NewAIError("AI服务响应无效")
//     toolErr := errors.NewToolError("工具执行失败")
//     mcpErr := errors.NewMCPError("MCP服务器连接失败")
//
//  3. 处理错误：
//     errors.HandleError(err)
//     userMessage := errors.GetUserFriendlyMessage(err)
//
//  4. 检查错误类型：
//     if errors.IsErrorCode(err, errors.ErrCodeConfigNotFound) {
//     // 处理配置未找到错误
//     }
//
//  5. 使用中间件：
//     // HTTP中间件
//     http.Handle("/", errors.HTTPMiddleware(myHandler))
package errors
