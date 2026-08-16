package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	apiKey := strings.TrimSpace(os.Getenv("DASHSCOPE_API_KEY"))
	if apiKey == "" {
		log.Fatal("请设置 DASHSCOPE_API_KEY 后再启动")
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  apiKey,
		Model:   "qwen3.8-max",
	})
	if err != nil {
		log.Fatalf("创建 ChatModel 失败: %v", err)
	}

	// 消息历史列表，先放入系统消息
	history := []*schema.Message{
		schema.SystemMessage("你是一个友好的Go语言助手，回答简洁，每次不超过100字。"),
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("开始对话（输入 quit 退出）：")

	for {
		fmt.Print("\n你: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "quit" {
			fmt.Println("再见！")
			break
		}
		if input == "" {
			continue
		}

		// 将用户消息追加到历史
		history = append(history, schema.UserMessage(input))

		// 用完整的历史消息列表调用模型
		stream, err := chatModel.Stream(ctx, history)
		if err != nil {
			log.Printf("调用失败: %v", err)
			continue
		}

		fmt.Print("助手: ")
		var reply strings.Builder

		for {
			chunk, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				log.Printf("读取失败: %v", err)
				break
			}
			fmt.Print(chunk.Content)
			reply.WriteString(chunk.Content)
		}
		stream.Close()
		fmt.Println()

		// 将模型回复追加到历史，下一轮对话就能带上这次的上下文
		history = append(history, &schema.Message{
			Role:    schema.Assistant,
			Content: reply.String(),
		})
	}
}
