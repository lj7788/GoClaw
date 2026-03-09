package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LarkStreamingSession represents a streaming card session
type LarkStreamingSession struct {
	client        *http.Client
	appID         string
	appSecret     string
	domain        string
	cardID        string
	messageID     string
	sequence      int
	currentText   string
	closed        bool
	lastUpdate    time.Time
	pendingText   string
	updateMutex   sync.Mutex
	token         string
	tokenExpiry   time.Time
	tokenMutex    sync.RWMutex
}

// NewLarkStreamingSession creates a new streaming session
func NewLarkStreamingSession(appID, appSecret, domain string) *LarkStreamingSession {
	return &LarkStreamingSession{
		client:    &http.Client{Timeout: 30 * time.Second},
		appID:     appID,
		appSecret: appSecret,
		domain:    domain,
	}
}

func (s *LarkStreamingSession) getAPIBase() string {
	if s.domain == "lark" {
		return "https://open.larksuite.com/open-apis"
	}
	if s.domain != "" && s.domain != "feishu" && strings.HasPrefix(s.domain, "http") {
		return strings.TrimSuffix(s.domain, "/") + "/open-apis"
	}
	return "https://open.feishu.cn/open-apis"
}

func (s *LarkStreamingSession) getTenantAccessToken(ctx context.Context) (string, error) {
	s.tokenMutex.RLock()
	if !s.tokenExpiry.IsZero() && time.Now().Before(s.tokenExpiry) {
		token := s.token
		s.tokenMutex.RUnlock()
		return token, nil
	}
	s.tokenMutex.RUnlock()

	s.tokenMutex.Lock()
	defer s.tokenMutex.Unlock()

	// Check again after acquiring lock
	if !s.tokenExpiry.IsZero() && time.Now().Before(s.tokenExpiry) {
		return s.token, nil
	}

	body := map[string]string{
		"app_id":     s.appID,
		"app_secret": s.appSecret,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/auth/v3/tenant_access_token/internal", s.getAPIBase())
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		Code              int    `json:"code"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int64  `json:"expire"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.Code != 0 {
		return "", fmt.Errorf("token error: code=%d", tokenResp.Code)
	}

	s.token = tokenResp.TenantAccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(tokenResp.Expire) * time.Second)

	return s.token, nil
}

// Start starts the streaming session
func (s *LarkStreamingSession) Start(ctx context.Context, receiveID, receiveIDType, question string) error {
	log.Printf("Lark: starting streaming session")

	token, err := s.getTenantAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Create streaming card with question
	content := "⏳ 思考中..."
	if question != "" {
		content = fmt.Sprintf("❓ %s\n\n⏳ 思考中...", truncateQuestion(question))
	}

	cardJSON := map[string]interface{}{
		"schema": "2.0",
		"config": map[string]interface{}{
			"streaming_mode": true,
			"summary": map[string]string{
				"content": "[Generating...]",
			},
			"streaming_config": map[string]interface{}{
				"print_frequency_ms": map[string]int{
					"default": 50,
				},
				"print_step": map[string]int{
					"default": 1,
				},
			},
		},
		"body": map[string]interface{}{
			"elements": []map[string]interface{}{
				{
					"tag":       "markdown",
					"content":   content,
					"element_id": "content",
				},
			},
		},
	}

	cardBody, err := json.Marshal(map[string]string{
		"type": "card_json",
		"data": func() string {
			b, _ := json.Marshal(cardJSON)
			return string(b)
		}(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	url := fmt.Sprintf("%s/cardkit/v1/cards", s.getAPIBase())
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(cardBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create card failed (status %d): %s", resp.StatusCode, string(body))
	}

	var createResp struct {
		Code int `json:"code"`
		Data struct {
			CardID string `json:"card_id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if createResp.Code != 0 {
		return fmt.Errorf("create card error: code=%d", createResp.Code)
	}

	s.cardID = createResp.Data.CardID
	log.Printf("Lark: card created, card_id=%s", s.cardID)

	// Send card to message
	cardContent := map[string]interface{}{
		"type": "card",
		"data": map[string]string{
			"card_id": s.cardID,
		},
	}

	contentJSON, _ := json.Marshal(cardContent)

	var sendResp struct {
		Code      int    `json:"code"`
		MessageID string `json:"message_id"`
		Msg       string `json:"msg"`
	}

	// Reply to message using receiveID
	url = fmt.Sprintf("%s/im/v1/messages/%s/reply", s.getAPIBase(), receiveID)
	req := map[string]interface{}{
		"msg_type":  "interactive",
		"content":   string(contentJSON),
	}

	reqBody, _ := json.Marshal(req)
	httpReq, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err = s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reply failed (status %d): %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if sendResp.Code != 0 {
		return fmt.Errorf("send card failed: code=%d, msg=%s", sendResp.Code, sendResp.Msg)
	}

	s.messageID = sendResp.MessageID
	s.sequence = 1
	s.currentText = ""
	s.closed = false

	log.Printf("Lark: streaming started, card_id=%s, message_id=%s", s.cardID, s.messageID)

	return nil
}

// Update updates the streaming card content
func (s *LarkStreamingSession) Update(ctx context.Context, text string) error {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if s.closed {
		return nil
	}

	// Merge text
	merged := mergeStreamingText(s.pendingText, text)
	if merged == "" || merged == s.currentText {
		return nil
	}

	// Throttle updates (max 10/sec)
	now := time.Now()
	if now.Sub(s.lastUpdate) < 100*time.Millisecond {
		s.pendingText = merged
		return nil
	}
	s.pendingText = ""
	s.lastUpdate = now

	// Update card content
	token, err := s.getTenantAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	s.sequence++
	url := fmt.Sprintf("%s/cardkit/v1/cards/%s/elements/content/content", s.getAPIBase(), s.cardID)
	req := map[string]interface{}{
		"content":  merged,
		"sequence": s.sequence,
		"uuid":     fmt.Sprintf("s_%s_%d", s.cardID, s.sequence),
	}

	reqBody, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Lark: update card failed (status %d): %s", resp.StatusCode, string(body))
		return fmt.Errorf("update card failed: %s", string(body))
	}

	s.currentText = merged
	log.Printf("Lark: card updated, sequence=%d, text_len=%d", s.sequence, len(merged))

	return nil
}

// Close closes the streaming session
func (s *LarkStreamingSession) Close(ctx context.Context, finalText string) error {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	// Merge pending text
	merged := mergeStreamingText(s.currentText, s.pendingText)
	if finalText != "" {
		merged = mergeStreamingText(merged, finalText)
	}

	// Final update if needed
	if merged != "" && merged != s.currentText {
		token, err := s.getTenantAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get token: %w", err)
		}

		s.sequence++
		url := fmt.Sprintf("%s/cardkit/v1/cards/%s/elements/content/content", s.getAPIBase(), s.cardID)
		req := map[string]interface{}{
			"content":  merged,
			"sequence": s.sequence,
			"uuid":     fmt.Sprintf("s_%s_%d", s.cardID, s.sequence),
		}

		reqBody, _ := json.Marshal(req)
		httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(reqBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(httpReq)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Lark: final update failed (status %d): %s", resp.StatusCode, string(body))
		} else {
			s.currentText = merged
		}
	}

	// Close streaming mode
	s.sequence++
	url := fmt.Sprintf("%s/cardkit/v1/cards/%s/settings", s.getAPIBase(), s.cardID)
	req := map[string]interface{}{
		"settings": map[string]interface{}{
			"config": map[string]interface{}{
				"streaming_mode": false,
				"summary": map[string]string{
					"content": truncateSummary(merged),
				},
			},
		},
		"sequence": s.sequence,
		"uuid":     fmt.Sprintf("c_%s_%d", s.cardID, s.sequence),
	}

	reqBody, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	token, err := s.getTenantAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Lark: close streaming failed (status %d): %s", resp.StatusCode, string(body))
		return fmt.Errorf("close streaming failed: %s", string(body))
	}

	log.Printf("Lark: streaming closed, card_id=%s", s.cardID)

	return nil
}

func (s *LarkStreamingSession) IsActive() bool {
	return !s.closed
}

func mergeStreamingText(previous, next string) string {
	if previous == "" {
		return next
	}
	if next == "" || next == previous {
		return previous
	}
	if strings.HasPrefix(next, previous) {
		return next
	}
	if strings.HasPrefix(previous, next) {
		return previous
	}
	if strings.Contains(next, previous) {
		return next
	}
	if strings.Contains(previous, next) {
		return previous
	}

	// Merge partial overlaps
	maxOverlap := min(len(previous), len(next))
	for overlap := maxOverlap; overlap > 0; overlap-- {
		if previous[len(previous)-overlap:] == next[:overlap] {
			return previous + next[overlap:]
		}
	}

	return previous + next
}

func truncateSummary(text string, maxLen ...int) string {
	length := 50
	if len(maxLen) > 0 {
		length = maxLen[0]
	}
	if text == "" {
		return ""
	}
	
	// Remove <think>...</think> tags (reasoning/thinking blocks from models like MiniMax)
	clean := removeThinkTags(text)
	
	clean = strings.ReplaceAll(clean, "\n", " ")
	clean = strings.TrimSpace(clean)
	if len(clean) <= length {
		return clean
	}
	return clean[:length-3] + "..."
}

func removeThinkTags(text string) string {
	result := ""
	rest := text
	for {
		start := strings.Index(rest, "<think>")
		if start == -1 {
			result += rest
			break
		}
		result += rest[:start]
		rest = rest[start+len("<think>"):]
		end := strings.Index(rest, "</think>")
		if end == -1 {
			break
		}
		rest = rest[end+len("</think>"):]
	}
	return strings.TrimSpace(result)
}

func truncateQuestion(question string, maxLen ...int) string {
	length := 30
	if len(maxLen) > 0 {
		length = maxLen[0]
	}
	if question == "" {
		return ""
	}
	clean := strings.ReplaceAll(question, "\n", " ")
	clean = strings.TrimSpace(clean)
	if len(clean) <= length {
		return clean
	}
	return clean[:length-3] + "..."
}
