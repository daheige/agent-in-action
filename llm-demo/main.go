package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	baseURL = "https://api.deepseek.com" // 模型厂商的 base URL
	model   = "deepseek-v4-pro"          // 使用的模型名称
)

// Message 请求message
type Message struct {
	Role    string `json:"role"`    // 角色
	Content string `json:"content"` // 内容
}

// ChatRequest 请求结构体
type ChatRequest struct {
	Model    string    `json:"model"`    // 使用的模型
	Messages []Message `json:"messages"` // 发送的message
	Stream   bool      `json:"stream"`   // 是否使用流式输出
}

// ChatResponse 返回结果
type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`     // 提示词消耗token
		CompletionTokens int `json:"completion_tokens"` // 完成消耗token
		TotalTokens      int `json:"total_tokens"`      // 总的tokens
	} `json:"usage"`
}

var question string

func main() {
	flag.StringVar(&question, "question", "用一句话解释什么是rust语言", "question")
	flag.Parse()

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Fatal("请先设置 LLM_API_KEY")
	}

	payload := ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "user", Content: question},
		},
		Stream: false,
	}

	// 转化为json格式
	body, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	// 建议设置合理的时间
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 创建 http request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// 检测是否请求成功
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		log.Fatalf("模型接口返回 %s: %s", resp.Status, raw)
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatal("failed to decode response,err:", err)
	}
	if len(result.Choices) == 0 {
		log.Fatal("模型响应中没有 choices")
	}

	// 输出内容
	fmt.Println(result.Choices[0].Message.Content)
	fmt.Printf(
		"token: input=%d output=%d total=%d\n",
		result.Usage.PromptTokens,
		result.Usage.CompletionTokens,
		result.Usage.TotalTokens,
	)
}
