// PicoClaw - Ultra-lightweight personal AI agent
// Qwen (通义千问) Provider implementation
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/auth"
)

// QwenProvider 实现 Qwen API 的 LLMProvider
type QwenProvider struct {
	apiKey      string
	httpClient  *HTTPProvider
	authMethod  string
}

// NewQwenProvider 创建新的 Qwen Provider
func NewQwenProvider(apiKey, apiBase string) *QwenProvider {
	if apiBase == "" {
		apiBase = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	return &QwenProvider{
		apiKey:     apiKey,
		httpClient: NewHTTPProvider(apiKey, apiBase, ""),
	}
}

// Chat 执行聊天请求
func (p *QwenProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	// 如果配置了 OAuth 认证，使用 OAuth token
	if p.authMethod == "oauth" {
		cred, err := auth.GetCredential("qwen")
		if err != nil {
			return nil, fmt.Errorf("加载认证凭证失败：%w", err)
		}
		if cred == nil {
			return nil, fmt.Errorf("未找到 Qwen 认证凭证，请运行：picoclaw auth login --provider qwen")
		}
		// 检查 token 是否过期
		if cred.IsExpired() {
			// TODO: 实现 token 刷新
			return nil, fmt.Errorf("Qwen 访问令牌已过期，请重新登录")
		}
		// 使用 OAuth token 创建临时 provider
		tempProvider := NewHTTPProvider(cred.AccessToken, "https://dashscope.aliyuncs.com/compatible-mode/v1", "")
		return tempProvider.Chat(ctx, messages, tools, model, options)
	}

	return p.httpClient.Chat(ctx, messages, tools, model, options)
}

// GetDefaultModel 返回默认模型
func (p *QwenProvider) GetDefaultModel() string {
	return "qwen-plus"
}

// createQwenAuthProvider 创建使用 OAuth 认证的 Qwen Provider
func createQwenAuthProvider() (LLMProvider, error) {
	cred, err := auth.GetCredential("qwen")
	if err != nil {
		return nil, fmt.Errorf("加载认证凭证失败：%w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("未找到 Qwen 认证凭证，请运行：picoclaw auth login --provider qwen")
	}

	return &QwenProvider{
		apiKey:     cred.AccessToken,
		authMethod: "oauth",
	}, nil
}

// LoginQwenOAuth 启动 Qwen OAuth 登录流程
func LoginQwenOAuth() error {
	fmt.Println("开始 Qwen OAuth 登录流程...")

	cred, err := auth.LoginQwenQRCode()
	if err != nil {
		return fmt.Errorf("登录失败：%w", err)
	}

	// 保存凭证
	if err := auth.SaveCredential("qwen", cred); err != nil {
		return fmt.Errorf("保存凭证失败：%w", err)
	}

	fmt.Println("✅ Qwen 登录成功！")
	fmt.Printf("账户 ID: %s\n", cred.AccountID)
	if !cred.ExpiresAt.IsZero() {
		fmt.Printf("令牌过期时间：%s\n", cred.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// RefreshQwenToken 刷新 Qwen 访问令牌
func RefreshQwenToken(refreshToken string) (*auth.AuthCredential, error) {
	// TODO: 实现令牌刷新逻辑
	return nil, fmt.Errorf("令牌刷新功能尚未实现")
}

// GetQwenModels 获取可用的 Qwen 模型列表
func GetQwenModels(apiKey string) ([]string, error) {
	// 这是一个辅助函数，用于获取 Qwen 可用的模型列表
	models := []string{
		"qwen-turbo",
		"qwen-plus",
		"qwen-max",
		"qwen-max-longcontext",
		"qwen-vl-max",
		"qwen-vl-plus",
		"qwen-audio-turbo",
	}
	return models, nil
}

// ParseQwenModel 解析模型名称，处理可能的 provider 前缀
func ParseQwenModel(model string) string {
	// 移除 qwen/ 前缀
	if strings.HasPrefix(model, "qwen/") {
		return strings.TrimPrefix(model, "qwen/")
	}
	if strings.HasPrefix(model, "dashscope/") {
		return strings.TrimPrefix(model, "dashscope/")
	}
	return model
}
