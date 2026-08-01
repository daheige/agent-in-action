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

脚本安装（推荐）macOS / Linux：
```shell
curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash
```
npm 安装（需要 Node.js 22.19.0 或更高版本）
```shell
node --version
npm install -g @moonshot-ai/kimi-code
```
或用 pnpm：
```shell
pnpm add -g @moonshot-ai/kimi-code
```
升级：运行 `kimi upgrade`，CLI 会检查最新版本并展示更新选项。选择 Install update now 后根据当前安装来源执行升级；也可以直接用包管理器：
```shell
npm install -g @moonshot-ai/kimi-code@latest
```
卸载：脚本安装的用户删除 kimi 可执行文件即可；npm 安装的用户：
```shell
npm uninstall -g @moonshot-ai/kimi-code
```

Kimi Code CLI 是一个运行在终端中的 AI Agent，帮助你完成软件开发任务和日常的终端操作——阅读和修改代码、执行 Shell 命令、搜索文件、抓取网页，并在执行过程中根据反馈自主规划和调整下一步行动。

它适用于以下场景：

- 编写和修改代码：实现新功能、修复 bug、完成重构
- 理解项目：探索陌生的代码库，解答架构和实现层面的问题
- 自动化任务：批量处理文件、运行构建与测试、串联多个脚本
- 整套 CLI 以 TypeScript 编写，通过 npm 分发，运行在 Node.js 之上。

#### kimi 接入
- API 接入：将 Kimi Code 接入第三方开发工具时，需要手动配置 API Key 完成认证。
- 服务地址：Kimi Code API 同时兼容 OpenAI 和 Anthropic 两种协议。不同工具对地址配置的要求不同：
    - Base URL：部分工具（如 Claude Code）只需填写 Base URL，工具会自动拼接后续路径。
    - 完整 Endpoint：部分工具（如 Trae）需要填写完整的 API 请求地址。

按需选择对应的地址：
| 协议           | Base URL                         | 常用 Endpoint 示例                                    |
| ------------ | -------------------------------- | ------------------------------------------------- |
| OpenAI 兼容    | <https://api.kimi.com/coding/v1> | <https://api.kimi.com/coding/v1/chat/completions> |
| Anthropic 兼容 | <https://api.kimi.com/coding/>   | <https://api.kimi.com/coding/v1/messages>         |

- 模型 ID：不同模型 ID 对应不同的会员档位与上下文窗口，具体权益见 会员权益。

| Model ID                  | 说明                                                                                                                                              |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| k3                        | Kimi K3，Moderato 及以上会员可调用，Allegretto 及以上会员可解锁最高 100 万上下文；支持 low / high / max 三档思考强度，适合大型代码库分析、多文件重构、超长文档处理等长上下文任务。                              |
| k3-256k                   | Kimi K3 256K 上下文版，Moderato 及以上会员可调用；支持 low / high / max 三档思考强度。256k 上下文内效果与 K3 相同，k3（1M）消耗约为 k3-256k 两倍，适合日常问答、代码补全、常规功能开发、单文件或少量文件修场景，不支持视频输入。 |
| kimi-for-coding           | Kimi K2.7 Code，所有会员可用。                                                                                                                          |
| kimi-for-coding-highspeed | Kimi K2.7 Code 高速版，Allegretto 及以上会员可用。                                                                                                          |
- 获取 API Key：Kimi 会员可在 Kimi Code 控制台 创建和管理（最多 5 个，仅创建时显示一次，请妥善保存）

## kimi cli
首次启动时需要配置 API 来源。在交互界面中输入 /login 进入登录流程
```shell
/login
```
/login 会弹出平台选择器，支持两种方式：
- Kimi Code（OAuth） — 验证码流程，在任意设备打开链接、登录并输入验证码即可授权
- Kimi Platform API 密钥 — 输入来自 platform.kimi.com 或 platform.kimi.ai 的 API 密钥
- 需要退出登录时，输入 /logout 清除当前凭证。

### 常用命令与快捷键速查
第一次使用时，记住下面这些就够了：

会话相关命令
| 命令        | 说明               |
| --------- | ---------------- |
| /new 或 /clear     | 开启新会话，清空当前上下文    |
| /sessions 或 /resume | 浏览历史会话，选择恢复      |
| /model    | 切换当前使用的模型        |
| /compact  | 手动压缩上下文，释放 token |
| /fork     | 派生当前会话，保留历史独立继续  |
| /init     | 分析当前代码库并生成 AGENTS.md |
| /web      | 在 web UI 中打开当前会话：选择一个运行中的实例进行连接，或在 TUI 退出后新开一个前台服务器。参见 kimi web：https://www.kimi.com/code/docs/kimi-code-cli/reference/kimi-command.html#kimi-web |
| fork        | 基于当前会话 fork 一份新会话，保留完整对话历史 |
| /tasks 或 /task    |     浏览后台任务列表    |
| /export    | 将当前会话导出为 Markdown 文件 |
| /exit    或 /quit 或 /q    | 退出 Kimi Code CLI    |

最常用快捷键
| 快捷键       | 说明               |
| --------- | ---------------- |
| Esc       | 中断流式输出 / 关闭弹窗    |
| Ctrl-C    | 中断输出；空闲时连按两次退出   |
| Shift-Tab | 切换 Plan 模式       |
| Ctrl-S    | 输出中途插入消息，无需等待结束  |
| Ctrl-O    | 折叠 / 展开工具输出和压缩摘要 |

想看完整列表，输入 /help 或访问：https://www.kimi.com/code/docs/kimi-code-cli/reference/slash-commands.html

## DeepSeek API

- 文档：https://api-docs.deepseek.com/zh-cn/
