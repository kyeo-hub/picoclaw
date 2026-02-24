# Qwen OAuth 扫码登录使用说明

## 概述

PicoClaw 现已支持通过阿里云 OAuth 扫码方式登录通义千问（Qwen）大模型。

## 登录步骤

### 1. 扫码登录

```bash
picoclaw auth login --provider qwen --qr-code
```

执行后：
1. 系统会自动打开浏览器显示阿里云 OAuth 二维码
2. 使用阿里云 APP 扫描二维码
3. 等待认证完成
4. 认证成功后，凭证会自动保存

### 2. 查看登录状态

```bash
picoclaw auth status
```

### 3. 登出

```bash
picoclaw auth logout --provider qwen
```

## 配置使用

### 方法一：修改配置文件

编辑 `~/.picoclaw/config.json`：

```json
{
  "agents": {
    "defaults": {
      "provider": "qwen",
      "model": "qwen-plus"
    }
  },
  "providers": {
    "qwen": {
      "api_key": "",
      "api_base": "https://dashscope.aliyuncs.com/compatible-mode/v1",
      "auth_method": "oauth"
    }
  }
}
```

### 方法二：使用示例配置

复制示例配置：

```bash
cp config/config.qwen.example.json ~/.picoclaw/config.json
```

## 支持的模型

- qwen-turbo
- qwen-plus
- qwen-max
- qwen-max-longcontext
- qwen-vl-max
- qwen-vl-plus
- qwen-audio-turbo

## 环境变量

也可以通过环境变量配置：

```bash
export PICOCLAW_PROVIDERS_QWEN_AUTH_METHOD=oauth
export PICOCLAW_AGENTS_DEFAULTS_PROVIDER=qwen
export PICOCLAW_AGENTS_DEFAULTS_MODEL=qwen-plus
```

## 故障排除

### 二维码无法显示

如果浏览器无法打开，可以手动访问显示的二维码 URL。

### 扫码后认证失败

确保使用的是阿里云 APP 进行扫码，并且账号已实名认证。

### 令牌过期

OAuth 令牌有过期时间，过期后需要重新登录：

```bash
picoclaw auth logout --provider qwen
picoclaw auth login --provider qwen --qr-code
```

## API 端点

- OAuth 授权端点：`https://oauth.aliyun.com`
- Qwen API 端点：`https://dashscope.aliyuncs.com/compatible-mode/v1`

## 注意事项

1. 扫码登录需要有效的阿里云账号
2. 通义千问服务可能需要实名认证
3. 令牌有有效期，请注意及时更新
4. 建议在安全的环境下使用扫码登录
