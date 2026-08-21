# Ollama-OpenAI Protocol Bridge

[English](README.md) | [中文文档](README_CN.md)

这是一个基于 Go 语言编写的极简、零依赖协议转换网关。它能够将 **Ollama API 请求** 转换为标准的 **OpenAI 兼容 API 请求**，并实现实时的流式响应转换。

通过它，您可以让任何支持 Ollama 协议的客户端、命令行工具或 IDE 插件，无缝连接到任意 OpenAI 兼容的大模型服务商（如 OpenAI, DeepSeek, Qwen, One-API, LiteLLM 等）。

## 💡 为什么需要这个网关？
许多客户端工具或插件硬编码绑定了 **Ollama 协议**（如 `/api/tags`, `/api/chat`），而您的主力大模型基础设施可能运行在 **OpenAI 标准接口**上（如 `/v1/models`, `/v1/chat/completions`）。

本网关充当了透明的“同声传译员”：
* **协议动态转换**：将 Ollama 的 NDJSON 请求转换为标准的 OpenAI SSE 流，并将响应实时重构为 Ollama 兼容的 NDJSON 流返回。
* **动态模型同步**：自动向上游 `/v1/models` 请求模型列表，并动态转换为 Ollama 格式（`/api/tags`）供客户端展示。
* **流传输稳定性优化**：完美处理流媒体边界和终止符，彻底避免严格客户端因未对齐而抛出的 `unexpected EOF` 断流错误。
* **请求深度清洗**：自动剥离 Ollama 独有的非标字段（如 `keep_alive`, `options`），防止上游标准接口崩溃。

## ✨ 核心特性
* **零外部依赖**：纯 Go 语言标准库实现。
* **极度轻量**：极低的资源占用，Docker 镜像仅约 10MB。
* **开箱即用**：无数据库、无复杂配置，秒级启动。

## 🚀 快速开始 (Docker)

推荐使用 Docker 进行一键部署。

### 1. 拉取镜像
```bash
docker pull yourusername/ollama-openai-bridge:latest

```

### 2. 运行容器

```bash
docker run -d \
  --name ollama-openai-bridge \
  -p 11434:11434 \
  -e UPSTREAM_BASE_URL="http://您的上游API地址/v1" \
  -e UPSTREAM_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxxx" \
  -e DEBUG_LOG="false" \
  --restart always \
  yourusername/ollama-openai-bridge:latest

```

## ⚙️ 环境变量配置

| 环境变量 | 默认值 | 描述 |
| --- | --- | --- |
| `UPSTREAM_BASE_URL` | `http://127.0.0.1:3000/v1` | 上游 OpenAI 兼容 API 的基础地址。**(末尾请勿带斜杠)** |
| `UPSTREAM_API_KEY` | `""` | 上游服务的 API Key。（程序会自动识别并补全 `Bearer ` 前缀） |
| `DEBUG_LOG` | `false` | 设为 `true` 时，开启超详细的请求与流数据包日志，用于排查问题。 |

*(注：容器内部固定监听 `11434` 端口以完美伪装 Ollama 服务，请通过 Docker `-p` 映射到宿主机的 `11434` 端口。)*

## 🛠️ 使用示例（客户端配置）

将任何支持 Ollama 协议的客户端指向本网关即可：

* **Host/URL**: `http://127.0.0.1:11434`
* **Model**: 直接选择您在上游服务商中配置的模型名称。
