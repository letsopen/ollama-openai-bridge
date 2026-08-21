# Ollama-OpenAI Protocol Bridge

[English](README.md) | [中文文档](README_CN.md)

An ultra-lightweight, zero-dependency Go gateway that translates **Ollama API requests** into standard **OpenAI-compatible API requests**, and streams responses back in real-time. 

It allows you to seamlessly connect any Ollama-supported client, tool, or IDE plugin to any OpenAI-compatible LLM provider (such as OpenAI, DeepSeek, Qwen, One-API, LiteLLM, etc.) without changing your client configuration.

## 💡 Why this bridge?
Many clients (like IDE extensions or local tools) are hardcoded to use the **Ollama protocol** (`/api/tags`, `/api/chat`), but your primary or preferred LLM infrastructure runs on **OpenAI-compatible endpoints** (`/v1/models`, `/v1/chat/completions`). 

This bridge acts as a transparent translator:
* **Protocol Translation**: Converts Ollama NDJSON requests into standard OpenAI SSE (Server-Sent Events) requests, and translates SSE responses back into Ollama's NDJSON stream format.
* **Dynamic Model Sync**: Automatically queries upstream `/v1/models` and maps them into Ollama-compatible metadata (`/api/tags`).
* **Stream Stabilization**: Solves stream boundary issues (such as `unexpected EOF` errors in strict clients) by handling frame reconstruction and clean termination.
* **Payload Sanitization**: Strips proprietary or non-standard payload fields (like Ollama's `keep_alive` or `options`) to prevent upstream API panics.

## ✨ Features
* **Zero Dependencies**: Pure Go standard library implementation.
* **Ultra Lightweight**: Minimal footprint, Docker image is ~10MB.
* **Plug and Play**: No database, no complex configuration, starts instantly.

## 🚀 Quick Start (Docker)

You can deploy the bridge instantly using Docker.

### 1. Pull the Image
```bash
docker pull yourusername/ollama-openai-bridge:latest

```

### 2. Run the Container

```bash
docker run -d \
  --name ollama-openai-bridge \
  -p 11434:11434 \
  -e UPSTREAM_BASE_URL="[http://your-openai-provider.com/v1](http://your-openai-provider.com/v1)" \
  -e UPSTREAM_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxxx" \
  -e DEBUG_LOG="false" \
  --restart always \
  yourusername/ollama-openai-bridge:latest

```

## ⚙️ Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `UPSTREAM_BASE_URL` | `http://127.0.0.1:3000/v1` | The base URL of your OpenAI-compatible API. **(Do not include trailing slash)** |
| `UPSTREAM_API_KEY` | `""` | The API Key for the upstream provider. The `Bearer ` prefix is auto-injected if missing. |
| `DEBUG_LOG` | `false` | Set to `true` to enable verbose request/response logging for troubleshooting. |

*(Note: The container listens on port `11434` internally to perfectly mimic Ollama. Map it to `11434` on your host.)*

## 🛠️ Usage Example (Client Configuration)

Configure your Ollama-supported client to point to this bridge:

* **Host/URL**: `http://127.0.0.1:11434`
* **Model**: Choose any model provided by your upstream OpenAI-compatible provider.
