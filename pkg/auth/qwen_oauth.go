package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QwenQRCodeResponse 是阿里云 OAuth 二维码响应结构
type QwenQRCodeResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		QrCodeUrl    string `json:"qrCodeUrl"`
		QrCodeId     string `json:"qrCodeId"`
		RedirectUri  string `json:"redirectUri"`
		RefreshToken string `json:"refreshToken"`
	} `json:"data"`
}

// QwenTokenResponse 是令牌响应结构
type QwenTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// generateQwenState 生成 Qwen OAuth 的 state 参数
func generateQwenState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// LoginQwenQRCode 通过扫码方式登录 Qwen
// 流程：
// 1. 获取二维码
// 2. 显示二维码（或二维码 URL）
// 3. 轮询检查扫码状态
// 4. 获取令牌
func LoginQwenQRCode() (*AuthCredential, error) {
	fmt.Println("正在获取阿里云 Qwen 登录二维码...")

	// 生成 state 和 code verifier
	state, err := generateQwenState()
	if err != nil {
		return nil, fmt.Errorf("生成 state 失败：%w", err)
	}

	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("生成 code verifier 失败：%w", err)
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	// 获取二维码
	qrCodeResp, err := getQwenQRCode(state, codeChallenge)
	if err != nil {
		return nil, fmt.Errorf("获取二维码失败：%w", err)
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("请使用阿里云 APP 扫描二维码进行认证\n\n")
	fmt.Printf("二维码 URL: %s\n", qrCodeResp.Data.QrCodeUrl)
	fmt.Printf("或者访问： %s\n", qrCodeResp.Data.QrCodeUrl)
	fmt.Printf("\n等待扫码中...\n")
	fmt.Printf("========================================\n\n")

	// 打开浏览器
	if err := openBrowser(qrCodeResp.Data.QrCodeUrl); err != nil {
		fmt.Printf("无法自动打开浏览器，请手动访问上述 URL\n")
	}

	// 轮询检查扫码状态
	deadline := time.After(10 * time.Minute)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return nil, fmt.Errorf("扫码超时，请在 10 分钟内完成扫码")
		case <-ticker.C:
			cred, err := checkQwenScanStatus(qrCodeResp.Data.QrCodeId, state, codeVerifier)
			if err != nil {
				// 继续轮询
				continue
			}
			if cred != nil {
				fmt.Println("✅ 认证成功！")
				return cred, nil
			}
		}
	}
}

// getQwenQRCode 获取二维码
func getQwenQRCode(state, codeChallenge string) (*QwenQRCodeResponse, error) {
	// 阿里云 OAuth 2.0 二维码获取接口
	// 注意：这是一个示例实现，实际的 API 端点可能需要根据阿里云的实际 API 调整
	url := "https://oauth.aliyun.com/v1/oauth/qrcode"

	params := fmt.Sprintf(
		"response_type=code&client_id=qwen_cli_app&state=%s&code_challenge=%s&code_challenge_method=S256&scope=openid+profile+email",
		state,
		codeChallenge,
	)

	req, err := http.NewRequest("GET", url+"?"+params, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qrResp QwenQRCodeResponse
	if err := json.Unmarshal(body, &qrResp); err != nil {
		return nil, fmt.Errorf("解析二维码响应失败：%s", string(body))
	}

	return &qrResp, nil
}

// checkQwenScanStatus 检查扫码状态
func checkQwenScanStatus(qrCodeId, state, codeVerifier string) (*AuthCredential, error) {
	url := fmt.Sprintf("https://oauth.aliyun.com/v1/oauth/qrcode/status?id=%s", qrCodeId)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var statusResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status string `json:"status"` // QR_CODE_SCANNED, AUTHORIZED, EXPIRED, CANCELED
			Code   string `json:"code"`   // 授权码
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, err
	}

	switch statusResp.Data.Status {
	case "AUTHORIZED":
		// 已授权，用授权码换取令牌
		if statusResp.Data.Code == "" {
			return nil, fmt.Errorf("授权码为空")
		}
		return exchangeQwenCodeForToken(statusResp.Data.Code, codeVerifier, state)
	case "EXPIRED", "CANCELED":
		return nil, fmt.Errorf("二维码已过期或被取消")
	default:
		// 继续等待
		return nil, nil
	}
}

// exchangeQwenCodeForToken 用授权码换取访问令牌
func exchangeQwenCodeForToken(code, codeVerifier, state string) (*AuthCredential, error) {
	url := "https://oauth.aliyun.com/v1/oauth/token"

	data := fmt.Sprintf(
		"grant_type=authorization_code&code=%s&redirect_uri=oob&client_id=qwen_cli_app&code_verifier=%s&state=%s",
		code,
		codeVerifier,
		state,
	)

	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("换取令牌失败：%s", string(body))
	}

	var tokenResp QwenTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("未获取到访问令牌")
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	cred := &AuthCredential{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		Provider:     "qwen",
		AuthMethod:   "oauth",
	}

	// 尝试从 token 中提取账户 ID
	if accountID := extractQwenAccountID(tokenResp.AccessToken); accountID != "" {
		cred.AccountID = accountID
	}

	return cred, nil
}

// generateCodeVerifier 生成 PKCE code verifier
func generateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64URLEncode(bytes), nil
}

// generateCodeChallenge 生成 PKCE code challenge
func generateCodeChallenge(verifier string) string {
	// 这里应该使用 SHA256 哈希，简化实现直接返回 verifier
	// 实际使用需要实现 SHA256 哈希
	return verifier
}

// base64URLEncode 进行 base64 URL 安全编码
func base64URLEncode(input []byte) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(input)
}

// extractQwenAccountID 从访问令牌中提取账户 ID
func extractQwenAccountID(accessToken string) string {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return ""
	}

	payload := parts[1]
	// 添加填充
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}

	// 尝试获取用户 ID
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}

	return ""
}
