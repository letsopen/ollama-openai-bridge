package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	targetHost string
	apiKey     string
	debugMode  bool
)

func initConfig() {
	// 1. 上游 OpenAI 兼容服务 BaseURL (默认 http://127.0.0.1:3000/v1)
	targetHost = getEnv("UPSTREAM_BASE_URL", "http://127.0.0.1:3000/v1")
	targetHost = strings.TrimSuffix(targetHost, "/")

	// 2. 上游 API Key
	apiKey = getEnv("UPSTREAM_API_KEY", "")
	if apiKey != "" && !strings.HasPrefix(apiKey, "Bearer ") {
		apiKey = "Bearer " + apiKey
	}

	// 3. Debug 日志开关 (true / false)
	debugMode = getEnv("DEBUG_LOG", "false") == "true"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func debugLog(format string, v ...interface{}) {
	if debugMode {
		log.Printf(format, v...)
	}
}

func main() {
	initConfig()

	log.Printf("=== Ollama-OpenAI Bridge 启动配置 ===")
	log.Printf("监听端口: :11434 (固定)")
	log.Printf("上游 BaseURL: %s", targetHost)
	log.Printf("Debug 模式: %v", debugMode)
	log.Printf("=====================================")

	// Ollama 基础探测
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte("Ollama is running"))
			return
		}
		if r.URL.Path == "/api/version" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write([]byte(`{"version":"0.1.48"}`))
			return
		}
		http.NotFound(w, r)
	})

	http.HandleFunc("/api/tags", handleTags)
	http.HandleFunc("/api/chat", handleChat)

	log.Fatal(http.ListenAndServe(":11434", nil))
}

// 动态拉取上游 OpenAI 兼容服务商的模型列表并转为 Ollama 格式
func handleTags(w http.ResponseWriter, r *http.Request) {
	debugLog("[Proxy] 正在从上游同步模型列表: %s/models", targetHost)

	outReq, err := http.NewRequest("GET", targetHost+"/models", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if apiKey != "" {
		outReq.Header.Set("Authorization", apiKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(outReq)
	if err != nil {
		log.Printf("[Error] 同步上游模型失败: %v", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"models":[],"error":""}`))
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var openAiResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &openAiResp); err != nil {
		log.Printf("[Error] 解析上游模型 JSON 失败: %v", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"models":[],"error":""}`))
		return
	}

	type OllamaModel struct {
		Name       string `json:"name"`
		Model      string `json:"model"`
		ModifiedAt string `json:"modified_at"`
		Size       int64  `json:"size"`
		Digest     string `json:"digest"`
	}

	var ollamaModels []OllamaModel
	for _, item := range openAiResp.Data {
		ollamaModels = append(ollamaModels, OllamaModel{
			Name:       item.ID,
			Model:      item.ID,
			ModifiedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Size:       0,
			Digest:     "fake-digest",
		})
	}

	ollamaResp := map[string]interface{}{
		"models": ollamaModels,
		"error":  "",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(ollamaResp)
	debugLog("[Proxy] 成功向客户端同步了 %d 个模型", len(ollamaModels))
}

// 协议转换与流式转发 (Ollama /api/chat -> OpenAI /chat/completions)
func handleChat(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	debugLog("\n[DEBUG] >>> 收到客户端请求: %s", string(bodyBytes))

	var ollamaReq map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &ollamaReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	openAiReq := make(map[string]interface{})
	if model, ok := ollamaReq["model"].(string); ok {
		openAiReq["model"] = strings.Split(model, ":")[0]
	}
	if messages, ok := ollamaReq["messages"].([]interface{}); ok {
		openAiReq["messages"] = messages
	}
	openAiReq["stream"] = true
	openAiReq["max_tokens"] = 4096

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(openAiReq)

	newBodyBytes := buf.Bytes()
	debugLog("[DEBUG] >>> 转换为 OpenAI 标准格式并转发")

	outReq, err := http.NewRequest("POST", targetHost+"/chat/completions", bytes.NewReader(newBodyBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	outReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		outReq.Header.Set("Authorization", apiKey)
	}
	outReq.Header.Set("Content-Length", strconv.Itoa(len(newBodyBytes)))

	client := &http.Client{}
	resp, err := client.Do(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(resp.StatusCode)

	flusher, canFlush := w.(http.Flusher)
	modelName, _ := openAiReq["model"].(string)

	scannerBuf := make([]byte, 4096)
	var lineBuffer bytes.Buffer

	for {
		n, err := resp.Body.Read(scannerBuf)
		if n > 0 {
			lineBuffer.Write(scannerBuf[:n])

			for {
				lineBytes := lineBuffer.Bytes()
				newlineIdx := bytes.IndexByte(lineBytes, '\n')
				if newlineIdx == -1 {
					break
				}

				line := string(lineBytes[:newlineIdx])
				lineBuffer.Next(newlineIdx + 1)

				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				if strings.HasPrefix(line, "data:") {
					payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

					if payload == "[DONE]" {
						finalChunk := map[string]interface{}{
							"model": modelName,
							"done":  true,
							"message": map[string]string{
								"role":    "assistant",
								"content": "",
							},
						}
						finalBytes, _ := json.Marshal(finalChunk)
						w.Write(finalBytes)
						w.Write([]byte("\n"))
						if canFlush {
							flusher.Flush()
						}
						debugLog("[DEBUG] <<< 转换并发送流终止包")
						break
					}

					var openaiChunk map[string]interface{}
					if jsonErr := json.Unmarshal([]byte(payload), &openaiChunk); jsonErr == nil {
						content := ""
						if choices, ok := openaiChunk["choices"].([]interface{}); ok && len(choices) > 0 {
							if choice, ok := choices[0].(map[string]interface{}); ok {
								if delta, ok := choice["delta"].(map[string]interface{}); ok {
									if c, ok := delta["content"].(string); ok {
										content = c
									}
								}
							}
						}

						if content != "" {
							ollamaChunk := map[string]interface{}{
								"model":      modelName,
								"created_at": time.Now().UTC().Format(time.RFC3339Nano),
								"message": map[string]string{
									"role":    "assistant",
									"content": content,
								},
								"done": false,
							}

							var chunkBuf bytes.Buffer
							chunkEnc := json.NewEncoder(&chunkBuf)
							chunkEnc.SetEscapeHTML(false)
							chunkEnc.Encode(ollamaChunk)

							w.Write(chunkBuf.Bytes())
							if canFlush {
								flusher.Flush()
							}
						}
					}
				}
			}
		}

		if err != nil {
			if err != io.EOF {
				log.Printf("读取后端流异常: %v", err)
			}
			break
		}
	}
}
