package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 120 * time.Second,
}

type chatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type contentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *imgURL `json:"image_url,omitempty"`
}

type imgURL struct {
	URL string `json:"url"`
}

type chatRequest struct {
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func OCRPage(baseURL string, pngData []byte) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(pngData)
	dataURL := "data:image/png;base64," + b64

	content, err := json.Marshal([]contentPart{
		{Type: "image_url", ImageURL: &imgURL{URL: dataURL}},
		{Type: "text", Text: "OCR markdown"},
	})
	if err != nil {
		return "", fmt.Errorf("marshal content: %w", err)
	}

	body, err := json.Marshal(chatRequest{
		Messages: []chatMessage{
			{Role: "user", Content: content},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal body: %w", err)
	}

	url := strings.TrimSuffix(baseURL, "/") + "/v1/chat/completions"
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("http post %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api error %d from %s: %s", resp.StatusCode, url, string(respBody))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
