# agent-in-action

`agent-in-action` 是一个用于学习和实践 AI Agent 相关技术的 Go 语言笔记仓库。当前主要包含两个子项目：

- **`llm-demo`**：通过原生 `curl` 演示如何直接调用不同 LLM 厂商的 API，并对比它们的请求格式差异。
- **`llm-gateway`**：一个基于 Go 标准库实现的 LLM 统一接入网关，封装多家厂商 API 差异，支持故障转移、路由策略、流式输出和成本统计。

---

## 仓库结构

```text
agent-in-action/
├── llm-demo/          # 原生 HTTP 调用 LLM 的示例与笔记
├── llm-gateway/       # LLM 统一接入网关（CLI + HTTP Server）
├── go.mod             # Go 模块定义
├── go.sum             # 依赖校验
└── readme.md          # 本文件
```

---

## 子项目简介

### llm-demo

`llm-demo` 记录了如何不借助任何 SDK，直接用 `curl` 调用 LLM 接口，并对比 OpenAI、Anthropic Claude、Google Gemini 等厂商在请求格式上的差异。

适合作为理解 LLM 协议差异的入门示例。

详情见 [`llm-demo/readme.md`](llm-demo/readme.md)。

### llm-gateway

`llm-gateway` 是一个更完整的工程示例，目标是用统一的抽象层接入多家 LLM 厂商。

核心能力：

- 多厂商统一接入：DeepSeek、豆包、Ollama、Claude、Gemini
- 同步调用与 SSE 流式输出
- 故障转移与路由策略：`priority` / `cheapest` / `latency`
- 基于 token 用量的成本估算
- 基于反射的 JSON Schema 生成
- 零外部生产依赖（仅 `godotenv` 用于本地 `.env` 加载）

详情见 [`llm-gateway/README.md`](llm-gateway/README.md)。

---

## 环境要求

- Go 1.26+
- 至少一个 LLM 厂商的 API Key，或本地运行的 Ollama 服务

---

## 本地部署 Ollama

如果你不想使用第三方 LLM 厂商的 API Key，可以直接在本地部署 Ollama 作为 Provider。Ollama 提供与 OpenAI 兼容的 `/v1/chat/completions` 接口，`llm-gateway` 已内置默认地址 `http://localhost:11434/v1`。

### 1. 安装 Ollama

访问官网下载对应系统的安装包：

- 官网：https://ollama.com
- macOS 也可以使用 Homebrew：

```bash
brew install ollama
```

### 2. 启动 Ollama 服务

安装完成后，启动 Ollama 服务（默认监听 `127.0.0.1:11434`）：

```bash
ollama serve
```

> 提示：在 macOS 上，通过官方安装包启动后，Ollama 通常会作为后台服务运行；如果命令行提示端口被占用，说明服务已经启动。

### 3. 拉取模型

例如拉取 `qwen2.5` 模型（约 4.7GB，请确保磁盘空间充足）：

```bash
ollama pull qwen2.5
```

其他常用模型：`llama3`、`deepseek-r1:8b`、`gemma2` 等，可在 [Ollama Library](https://ollama.com/library) 查看。

### 4. 验证本地服务

```bash
curl http://localhost:11434/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

如果能正常返回模型回复，说明本地 Ollama 部署成功。

### 5. 在 llm-gateway 中使用

进入 `llm-gateway` 目录，复制并编辑 `.env`：

```bash
cd llm-gateway
cp .env.example .env
```

在 `.env` 中配置 Ollama 相关环境变量（`OLLAMA_API_KEY` 可留空）：

```bash
OLLAMA_BASE_URL=http://localhost:11434/v1
OLLAMA_API_KEY=
OLLAMA_MODEL=qwen2.5
```

然后即可通过 `make run` 或 `make run-web` 使用本地模型。

---

## 快速开始

### 进入 llm-gateway 子项目

```bash
cd llm-gateway
```

### 配置环境变量

```bash
cp .env.example .env
# 编辑 .env，填入你的 API Key 和模型名
```

### 运行 CLI

```bash
make run QUESTION="为什么 Go 适合开发 AI Agent？"
```

### 启动 HTTP 服务

```bash
make run-web
```

更多用法请参考 [`llm-gateway/README.md`](llm-gateway/README.md)。

---

## 参考资源

### Kimi Code

- 安装：https://www.kimi.com/code?from=kimi_homepage_sidebar
- 文档：https://www.kimi.com/code/docs/

```shell
curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash
```

### DeepSeek API

- 文档：https://api-docs.deepseek.com/zh-cn/
