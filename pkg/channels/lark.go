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
	"compress/zlib"

	"github.com/gorilla/websocket"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

const (
	feishuAPIBase      = "https://open.feishu.cn/open-apis"
	feishuWSBaseURL   = "https://open.feishu.cn"
	larkAPIBase        = "https://open.larksuite.com/open-apis"
	larkWSBaseURL      = "https://open.larksuite.com"
	wsHeartbeatTimeout = 300 * time.Second
	tokenRefreshSkew   = 120 * time.Second
	defaultTokenTTL    = 7200 * time.Second
)

type LarkPlatform int

const (
	PlatformFeishu LarkPlatform = iota
	PlatformLark
)

func (p LarkPlatform) apiBase() string {
	switch p {
	case PlatformFeishu:
		return feishuAPIBase
	case PlatformLark:
		return larkAPIBase
	default:
		return feishuAPIBase
	}
}

func (p LarkPlatform) wsBase() string {
	switch p {
	case PlatformFeishu:
		return feishuWSBaseURL
	case PlatformLark:
		return larkWSBaseURL
	default:
		return feishuWSBaseURL
	}
}

func (p LarkPlatform) localeHeader() string {
	switch p {
	case PlatformFeishu:
		return "zh"
	case PlatformLark:
		return "en"
	default:
		return "zh"
	}
}

type LarkChannel struct {
	appID             string
	appSecret         string
	allowedUsers       []string
	platform          LarkPlatform
	httpClient        *http.Client
	tenantToken       string
	tokenExpiry       time.Time
	tokenMutex        sync.RWMutex
	wsSeenIDs        map[string]time.Time
	wsSeenIDsMutex   sync.RWMutex
	botOpenID        string
	botOpenIDMutex    sync.RWMutex
}

type WsEndpointResponse struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg,omitempty"`
	Data *WsEndpoint `json:"data,omitempty"`
}

type WsEndpoint struct {
	URL          string       `json:"URL"`
	ClientConfig  *ClientConfig `json:"ClientConfig,omitempty"`
}

type ClientConfig struct {
	PingInterval int64 `json:"PingInterval"`
}

type TenantTokenResponse struct {
	Code             int    `json:"code"`
	Msg              string `json:"msg,omitempty"`
	TenantAccessToken string `json:"tenant_access_token,omitempty"`
	Expire           int64  `json:"expire,omitempty"`
}

type BotInfoResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg,omitempty"`
	Bot  *BotInfo `json:"bot,omitempty"`
}

type BotInfo struct {
	OpenID string `json:"open_id"`
}

type LarkEvent struct {
	Header LarkEventHeader `json:"header"`
	Event  json.RawMessage `json:"event"`
}

type LarkEventHeader struct {
	EventType string `json:"event_type"`
	EventID   string `json:"event_id"`
}

type MessageReceiveEvent struct {
	Sender  LarkSender  `json:"sender"`
	Message LarkMessage `json:"message"`
}

type LarkSender struct {
	SenderID   LarkSenderID `json:"sender_id"`
	SenderType string       `json:"sender_type"`
}

type LarkSenderID struct {
	OpenID string `json:"open_id"`
	UserID string `json:"user_id"`
	UnionID string `json:"union_id"`
}

type LarkMessage struct {
	MessageID   string `json:"message_id"`
	ChatID      string `json:"chat_id"`
	ChatType    string `json:"chat_type"`
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
	Mentions    []json.RawMessage `json:"mentions,omitempty"`
}

type SendMessageRequest struct {
	ReceiveID     string `json:"receive_id"`
	ReceiveIDType string `json:"receive_id_type"`
	MsgType       string `json:"msg_type"`
	Content       string `json:"content"`
}

type MessageContent struct {
	Text string `json:"text"`
}

func NewLarkChannel(appID, appSecret string, allowedUsers []string) *LarkChannel {
	log.Printf("Lark: initializing with appID=%s, allowedUsers=%v", appID, allowedUsers)
	return &LarkChannel{
		appID:       appID,
		appSecret:    appSecret,
		allowedUsers: normalizeAllowedUsers(allowedUsers),
		platform:     PlatformFeishu,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		wsSeenIDs:   make(map[string]time.Time),
	}
}

func (c *LarkChannel) Name() string {
	return "lark"
}

func (c *LarkChannel) AppID() string {
	return c.appID
}

func (c *LarkChannel) AppSecret() string {
	return c.appSecret
}

func (c *LarkChannel) Send(ctx context.Context, message *types.SendMessage) error {
	log.Printf("Lark Send: sending message to recipient=%s", message.Recipient)

	token, err := c.getTenantAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get tenant access token: %w", err)
	}

	content := MessageContent{
		Text: message.Content,
	}
	contentJSON, _ := json.Marshal(content)

	req := SendMessageRequest{
		ReceiveID:     message.Recipient,
		ReceiveIDType: "chat_id",
		MsgType:       "text",
		Content:       string(contentJSON),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/im/v1/messages?receive_id_type=chat_id", c.platform.apiBase())
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Lark message send failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *LarkChannel) Listen(ctx context.Context, msgChan chan<- types.ChannelMessage) error {
	log.Printf("Lark: Listen started, appID=%s", c.appID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Lark: context cancelled")
			return ctx.Err()
		default:
			if err := c.listenWebSocket(ctx, msgChan); err != nil {
				log.Printf("Lark: WebSocket error: %v, reconnecting in 5 seconds...", err)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
					continue
				}
			}
		}
	}
}

func (c *LarkChannel) listenWebSocket(ctx context.Context, msgChan chan<- types.ChannelMessage) error {
	wsEndpoint, clientConfig, err := c.getWSEndpoint()
	if err != nil {
		return fmt.Errorf("failed to get WS endpoint: %w", err)
	}

	log.Printf("Lark: connecting to WebSocket: %s", wsEndpoint)

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	log.Printf("Lark: WebSocket connected")

	pingInterval := clientConfig.PingInterval
	if pingInterval < 10 {
		pingInterval = 120
	}

	ticker := time.NewTicker(time.Duration(pingInterval) * time.Second)
	defer ticker.Stop()

	lastRecv := time.Now()
	timeoutCheck := time.NewTicker(10 * time.Second)
	defer timeoutCheck.Stop()

	seq := uint64(0)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Lark: context cancelled")
			return ctx.Err()

		case <-ticker.C:
			seq++
			ping := map[string]interface{}{
				"seq_id":  seq,
				"log_id":  0,
				"service": 0,
				"method":  0,
				"headers": []map[string]string{
					{"key": "type", "value": "ping"},
				},
				"payload": nil,
			}
			if err := conn.WriteJSON(ping); err != nil {
				return fmt.Errorf("failed to send ping: %w", err)
			}

		case <-timeoutCheck.C:
			if time.Since(lastRecv) > wsHeartbeatTimeout {
				return fmt.Errorf("heartbeat timeout")
			}

		default:
			conn.SetReadDeadline(time.Now().Add(120 * time.Second))
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return fmt.Errorf("WebSocket closed unexpectedly: %w", err)
				}
				return fmt.Errorf("read error: %w", err)
			}

			switch messageType {
			case websocket.BinaryMessage:
				lastRecv = time.Now()
				if err := c.handleBinaryMessage(message, msgChan); err != nil {
					log.Printf("Lark: failed to handle binary message: %v", err)
				}
			case websocket.PingMessage:
				lastRecv = time.Now()
				if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
					return fmt.Errorf("failed to send pong: %w", err)
				}
			case websocket.CloseMessage:
				log.Printf("Lark: WebSocket closed by server")
				return nil
			}
		}
	}
}

func (c *LarkChannel) handleBinaryMessage(data []byte, msgChan chan<- types.ChannelMessage) error {
	log.Printf("Lark: received %d bytes", len(data))
	log.Printf("Lark: first 20 bytes: %x", data[:min(20, len(data))])

	// Decompress zlib compressed data if needed
	if len(data) > 0 && data[0] == 0x78 {
		log.Printf("Lark: data appears to be zlib compressed")
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create zlib reader: %w", err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to decompress data: %w", err)
		}
		log.Printf("Lark: decompressed to %d bytes", len(decompressed))
		log.Printf("Lark: first 20 bytes of decompressed: %x", decompressed[:min(20, len(decompressed))])
		data = decompressed
	}

	// Parse Protobuf frame
	frame, err := parseProtobufToJSON(data)
	if err != nil {
		log.Printf("Lark: failed to parse Protobuf frame: %v", err)
		log.Printf("Lark: raw data (first 100 chars): %s", string(data[:min(100, len(data))]))
		return nil
	}

	log.Printf("Lark: successfully parsed Protobuf frame")

	// Extract fields from parsed frame
	method, _ := frame["field_4"].(float64)
	service, _ := frame["field_3"].(float64)
	log.Printf("Lark: method=%.0f, service=%.0f", method, service)

	// Extract headers from field_5
	headers, ok := frame["field_5"].(map[string]interface{})
	if ok {
		log.Printf("Lark: headers=%v", headers)
	}

	// Extract payload from field_8
	payload, ok := frame["field_8"].(map[string]interface{})
	var payloadBytes []byte
	if ok {
		payloadBytes, _ = json.Marshal(payload)
		log.Printf("Lark: payload length=%d, first 200 chars: %s", len(payloadBytes), string(payloadBytes[:min(200, len(payloadBytes))]))
	}

	// Parse the JSON event from payload (do this for all frames, not just method=1)
	var larkEvent struct {
		Header struct {
			EventType string `json:"event_type"`
			EventID   string `json:"event_id"`
		}
		Event json.RawMessage `json:"event"`
	}

	if err := json.Unmarshal(payloadBytes, &larkEvent); err != nil {
		log.Printf("Lark: failed to parse event JSON: %v, payload: %s", err, string(payloadBytes))
		return nil
	}

	// Only process message receive events
	if larkEvent.Header.EventType != "im.message.receive_v1" {
		log.Printf("Lark: skipping non-message event: %s", larkEvent.Header.EventType)
		return nil
	}

	log.Printf("Lark: received message event")

	// Parse the message payload
	var msgPayload struct {
		Sender struct {
			SenderID struct {
				UnionID string `json:"union_id"`
				OpenID  string `json:"open_id"`
			} `json:"sender_id"`
			SenderType string `json:"sender_type"`
		} `json:"sender"`
		Message struct {
			ChatID      string `json:"chat_id"`
			ChatType    string `json:"chat_type"`
			MessageType string `json:"message_type"`
			Content     string `json:"content"`
			MessageID   string `json:"message_id"`
		} `json:"message"`
	}

	if err := json.Unmarshal(larkEvent.Event, &msgPayload); err != nil {
		log.Printf("Lark: failed to parse message payload: %v, event: %s", err, string(larkEvent.Event))
		return nil
	}

	log.Printf("Lark: parsed message payload, sender_type=%s", msgPayload.Sender.SenderType)

	messageID := msgPayload.Message.MessageID
	eventID := larkEvent.Header.EventID

	if messageID != "" && c.isMessageSeen(messageID) {
		log.Printf("Lark: skipping duplicate message: %s", messageID)
		return nil
	}

	if eventID != "" && c.isMessageSeen(eventID) {
		log.Printf("Lark: skipping duplicate event: %s", eventID)
		return nil
	}

	if messageID != "" {
		c.markMessageSeen(messageID)
	} else if eventID != "" {
		c.markMessageSeen(eventID)
	}

	log.Printf("Lark: message ID=%s, event ID=%s", messageID, eventID)

	// Skip bot messages
	if msgPayload.Sender.SenderType == "app" || msgPayload.Sender.SenderType == "bot" {
		log.Printf("Lark: skipping bot message")
		return nil
	}

	// Determine sender ID
	senderID := msgPayload.Sender.SenderID.UnionID
	if senderID == "" {
		senderID = msgPayload.Sender.SenderID.OpenID
	}
	if senderID == "" {
		senderID = "unknown"
	}
	log.Printf("Lark: sender ID=%s", senderID)

	// Parse message content
	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(msgPayload.Message.Content), &content); err != nil {
		log.Printf("Lark: failed to unmarshal message content: %v, content: %s", err, msgPayload.Message.Content)
		return nil
	}
	log.Printf("Lark: extracted content: %s", content.Text)

	// Send to message channel
	log.Printf("Lark: sending message to channel, content=%s", content.Text)
	msgChan <- types.ChannelMessage{
		Channel:     "lark",
		Sender:      senderID,
		Content:     content.Text,
		ReplyTarget: msgPayload.Message.ChatID,
		MessageID:   messageID,
	}
	log.Printf("Lark: message sent to channel")

	return nil
}

func (c *LarkChannel) extractMessageContent(content string) string {
	var contentMap map[string]interface{}
	if err := json.Unmarshal([]byte(content), &contentMap); err != nil {
		return content
	}

	if text, ok := contentMap["text"].(string); ok {
		return text
	}

	return ""
}

func (c *LarkChannel) getSenderID(sender LarkSender) string {
	if sender.SenderID.OpenID != "" {
		return sender.SenderID.OpenID
	}
	if sender.SenderID.UserID != "" {
		return sender.SenderID.UserID
	}
	if sender.SenderID.UnionID != "" {
		return sender.SenderID.UnionID
	}
	return ""
}

func (c *LarkChannel) isMessageSeen(messageID string) bool {
	c.wsSeenIDsMutex.RLock()
	defer c.wsSeenIDsMutex.RUnlock()
	_, exists := c.wsSeenIDs[messageID]
	return exists
}

func (c *LarkChannel) markMessageSeen(messageID string) {
	c.wsSeenIDsMutex.Lock()
	defer c.wsSeenIDsMutex.Unlock()
	c.wsSeenIDs[messageID] = time.Now()

	for id, ts := range c.wsSeenIDs {
		if time.Since(ts) > 30*time.Minute {
			delete(c.wsSeenIDs, id)
		}
	}
}

func (c *LarkChannel) getWSEndpoint() (string, *ClientConfig, error) {
	body := map[string]string{
		"AppID":     c.appID,
		"AppSecret":  c.appSecret,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/callback/ws/endpoint", c.platform.wsBase()), strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("locale", c.platform.localeHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("WS endpoint request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var wsResp WsEndpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&wsResp); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if wsResp.Code != 0 {
		return "", nil, fmt.Errorf("WS endpoint error: code=%d, msg=%s", wsResp.Code, wsResp.Msg)
	}

	if wsResp.Data == nil {
		return "", nil, fmt.Errorf("WS endpoint: empty data")
	}

	return wsResp.Data.URL, wsResp.Data.ClientConfig, nil
}

func (c *LarkChannel) getTenantAccessToken() (string, error) {
	c.tokenMutex.RLock()
	if !c.tokenExpiry.IsZero() && time.Now().Before(c.tokenExpiry) {
		token := c.tenantToken
		c.tokenMutex.RUnlock()
		return token, nil
	}
	c.tokenMutex.RUnlock()

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	body := map[string]string{
		"app_id":     c.appID,
		"app_secret":  c.appSecret,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/v3/tenant_access_token/internal", c.platform.apiBase()), strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TenantTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.Code != 0 {
		return "", fmt.Errorf("token error: code=%d, msg=%s", tokenResp.Code, tokenResp.Msg)
	}

	ttl := time.Duration(tokenResp.Expire) * time.Second
	if ttl == 0 {
		ttl = defaultTokenTTL
	}

	c.tenantToken = tokenResp.TenantAccessToken
	c.tokenExpiry = time.Now().Add(ttl).Add(-tokenRefreshSkew)

	return c.tenantToken, nil
}

func (c *LarkChannel) HealthCheck(ctx context.Context) error {
	token, err := c.getTenantAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get tenant access token: %w", err)
	}

	if token == "" {
		return fmt.Errorf("empty tenant access token")
	}

	return nil
}

func (c *LarkChannel) StartTyping(ctx context.Context, recipient string) error {
	return nil
}

func (c *LarkChannel) StopTyping(ctx context.Context, recipient string) error {
	return nil
}

func (c *LarkChannel) SupportsDraftUpdates() bool {
	return false
}

func (c *LarkChannel) SendDraft(ctx context.Context, message *types.SendMessage) (string, error) {
	return "", fmt.Errorf("draft updates not supported")
}

func (c *LarkChannel) UpdateDraft(ctx context.Context, recipient, messageID, text string) (string, error) {
	return "", fmt.Errorf("draft updates not supported")
}

func (c *LarkChannel) FinalizeDraft(ctx context.Context, recipient, messageID, text string) error {
	return fmt.Errorf("draft updates not supported")
}

func (c *LarkChannel) CancelDraft(ctx context.Context, recipient, messageID string) error {
	return fmt.Errorf("draft updates not supported")
}

func (c *LarkChannel) AddReaction(ctx context.Context, channelID, messageID, emoji string) error {
	log.Printf("Lark: adding reaction %s to message %s", emoji, messageID)

	token, err := c.getTenantAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get tenant access token: %w", err)
	}

	req := map[string]interface{}{
		"reaction_type": map[string]string{
			"emoji_type": emoji,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/im/v1/messages/%s/reactions", c.platform.apiBase(), messageID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Lark add reaction failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *LarkChannel) RemoveReaction(ctx context.Context, channelID, messageID, emoji string) error {
	log.Printf("Lark: removing reaction %s from message %s", emoji, messageID)

	token, err := c.getTenantAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get tenant access token: %w", err)
	}

	// Get reaction_id first
	reactionID, err := c.getReactionID(ctx, messageID, emoji, token)
	if err != nil {
		log.Printf("Lark: failed to get reaction ID: %v", err)
		return err
	}
	if reactionID == "" {
		log.Printf("Lark: reaction not found")
		return nil
	}

	url := fmt.Sprintf("%s/im/v1/messages/%s/reactions/%s", c.platform.apiBase(), messageID, reactionID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Lark remove reaction failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *LarkChannel) getReactionID(ctx context.Context, messageID, emoji, token string) (string, error) {
	url := fmt.Sprintf("%s/im/v1/messages/%s/reactions", c.platform.apiBase(), messageID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Lark get reactions failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ReactionID string `json:"reaction_id"`
				ReactionType struct {
					EmojiType string `json:"emoji_type"`
				} `json:"reaction_type"`
			} `json:"items"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("Lark get reactions error: code=%d", result.Code)
	}

	for _, item := range result.Data.Items {
		if item.ReactionType.EmojiType == emoji {
			return item.ReactionID, nil
		}
	}

	return "", nil
}

func (c *LarkChannel) isUserAllowed(userID string) bool {
	if len(c.allowedUsers) == 0 {
		return true
	}

	for _, user := range c.allowedUsers {
		if user == "*" || user == userID {
			return true
		}
	}

	return false
}