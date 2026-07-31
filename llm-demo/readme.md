# curl请求
先配置环境变量
```shell
# 这里我使用的是deepseek llm
export LLM_API_KEY=sk-your-apikey
export LLM_BASE_URL=https://api.deepseek.com
export LLM_MODEL=deepseek-v4-pro
```
执行curl请求
```shell
curl "$LLM_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $LLM_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"$LLM_MODEL\",
    \"messages\": [
      {
        \"role\": \"user\",
        \"content\": \"用一句话解释什么是 AI Agent\"
      }
    ],
    \"stream\": false
  }"
```

# 不同厂商的格式差异
```ini
// OpenAI：messages 数组，system 是一种 role
{
  "model": "gpt-5",
  "messages": [
    {"role": "system", "content": "你是助手"},
    {"role": "user", "content": "你好"}
  ]
}

// Anthropic：system 是顶层字段，messages 只有 user/assistant
{
  "model": "claude-sonnet-4-20250514",
  "system": "你是助手",
  "messages": [
    {"role": "user", "content": "你好"}
  ],
  "max_tokens": 1024
}

// Gemini：contents 数组，role 使用 "model"，systemInstruction 单独配置
{
  "contents": [
    {"role": "user", "parts": [{"text": "你好"}]}
  ],
  "systemInstruction": {
    "parts": [{"text": "你是助手"}]
  }
}
```
几个必须关注的差异点：

- system 的位置不同。OpenAI 把 system 作为 messages 中的一项；Anthropic 使用顶层 system 字段；Gemini 使用 systemInstruction。
- role 命名不同。OpenAI 和 Anthropic 使用 assistant；Gemini 使用 model 表示模型回复。
- content 形态不同。OpenAI 常见是 string，也支持多模态数组；Anthropic 使用 string 或 content blocks；Gemini 使用 parts 数组。
- max_tokens 要求不同。Anthropic Messages API 要求请求中带 max_tokens；OpenAI 兼容协议通常可选。
