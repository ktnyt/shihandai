package sentence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Ollama はローカルの Ollama サーバで動く LFM2.5 を呼び出す。
type Ollama struct {
	URL    string // 例: http://localhost:11434
	Model  string // 例: lfm2.5
	Client *http.Client
}

// NewOllama は既定のHTTPクライアントを持つ Ollama を作る。
// LFM2.5 は思考トークンを出すモデルなので、1回の生成に時間がかかることを見込む。
func NewOllama(url, model string) *Ollama {
	return &Ollama{
		URL:    strings.TrimRight(url, "/"),
		Model:  model,
		Client: &http.Client{Timeout: 120 * time.Second},
	}
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
	Error   string      `json:"error"`
}

// Generate は使えるかなと単語の例を渡して短文を1つ生成する。
func (o *Ollama) Generate(ctx context.Context, allowed, hints []string) (string, error) {
	prompt := fmt.Sprintf(
		"日本語のタイピング練習の例文を1つ作ってください。\n"+
			"次のひらがなだけを使えます: %s\n"+
			"これ以外の文字（漢字、カタカナ、英数字、記号）は使えません。\n",
		strings.Join(allowed, " "))
	if len(hints) > 0 {
		prompt += "この文字だけで打てる単語の例: " + strings.Join(hints, " ") + "\n"
	}
	prompt += "条件: 8〜25文字の自然な短い文。文だけを出力してください。"

	body, err := json.Marshal(chatRequest{
		Model: o.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Options: map[string]any{
			"temperature": 0.9,
			"num_predict": 2048, // 思考トークンの分も含む
		},
	})
	if err != nil {
		return "", fmt.Errorf("リクエストの生成に失敗: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.URL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("リクエストの生成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama への接続に失敗: %w", err)
	}
	defer resp.Body.Close()

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("ollama の応答の解析に失敗: %w", err)
	}
	if parsed.Error != "" {
		return "", fmt.Errorf("ollama がエラーを返した: %s", parsed.Error)
	}
	return parsed.Message.Content, nil
}

// Available はモデルが利用できるかを確かめる。
func (o *Ollama) Available(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{"model": o.Model})
	if err != nil {
		return fmt.Errorf("リクエストの生成に失敗: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.URL+"/api/show", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("リクエストの生成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.Client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama への接続に失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("モデル %q が見つからない (ollama pull %s を実行してください)", o.Model, o.Model)
	}
	return nil
}
