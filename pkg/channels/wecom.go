package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

const (
	weComAPIBase           = "https://qyapi.weixin.qq.com/cgi-bin"
	weComWebSocketBaseURL  = "wss://openws.work.weixin.qq.com"
	weComMessageTimeout    = 5 * time.Second
	weComHeartbeatInterval = 30 * time.Second
)

type WecomChannel struct {
	botID         string
	botSecret     string
	client        *http.Client
	websocketURL  string
	allowedUsers  []string
	groupPolicy   string
	groupAllowFrom []string
	
	wsClient   *WecomWSClient
	wsMutex    sync.RWMutex
	
	accessToken   string
	tokenExpiry   time.Time
	tokenMutex    sync.RWMutex
}

type WecomMessage struct {
	MsgID        string      `json:"msgid"`
	ChatID       string      `json:"chatid"`
	ChatType     string      `json:"chattype"`
	From         WecomUser   `json:"from"`
	ResponseType string      `json:"response_url"`
	MsgType      string      `json:"msgtype"`
	ReqID        string      `json:"req_id"`
	Text         *WecomText  `json:"text"`
	Image        *WecomImage `json:"image"`
	Mixed        *WecomMixed `json:"mixed"`
	Voice        *WecomVoice `json:"voice"`
	File         *WecomFile  `json:"file"`
	Quote        *WecomQuote `json:"quote"`
}

type WecomUser struct {
	UserID string `json:"userid"`
}

type WecomText struct {
	Content string `json:"content"`
}

type WecomImage struct {
	URL     string `json:"url"`
	AesKey  string `json:"aeskey"`
	Base64  string `json:"base64"`
	MD5     string `json:"md5"`
}

type WecomMixed struct {
	MsgItem []WecomMixedItem `json:"msg_item"`
}

type WecomMixedItem struct {
	MsgType string      `json:"msgtype"`
	Text    *WecomText  `json:"text"`
	Image   *WecomImage `json:"image"`
	File    *WecomFile  `json:"file"`
}

type WecomVoice struct {
	Content string `json:"content"`
}

type WecomFile struct {
	URL     string `json:"url"`
	AesKey  string `json:"aeskey"`
}

type WecomQuote struct {
	MsgType string      `json:"msgtype"`
	Text    *WecomText  `json:"text"`
	Image   *WecomImage `json:"image"`
	File    *WecomFile  `json:"file"`
}

type WecomSendMessage struct {
	ToUser       string             `json:"touser"`
	MsgType      string             `json:"msgtype"`
	Text         *WecomText         `json:"text,omitempty"`
	Markdown     *WecomMarkdown     `json:"markdown,omitempty"`
	TemplateCard *WecomTemplateCard `json:"template_card,omitempty"`
}

type WecomMarkdown struct {
	Content string `json:"content"`
}

type WecomTemplateCard struct {
	CardType   string           `json:"card_type"`
	MainTitle  *WecomTitle      `json:"main_title"`
	ImageCard  *WecomImageCard  `json:"image_card"`
	CardAction *WecomCardAction `json:"card_action"`
}

type WecomTitle struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

type WecomImageCard struct {
	Image string `json:"image"`
	URL   string `json:"url"`
}

type WecomCardAction struct {
	Type    int    `json:"type"`
	URL     string `json:"url"`
	AgentID string `json:"agentid"`
	Parame  string `json:"parame"`
}

type WecomAccessTokenResponse struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func NewWecomChannel(botID, botSecret string) *WecomChannel {
	log.Printf("WeCom: Creating channel with botID=%s", botID)
	
	wsClient := NewWecomWSClient(botID, botSecret, weComWebSocketBaseURL)
	
	return &WecomChannel{
		botID:        botID,
		botSecret:    botSecret,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		websocketURL: weComWebSocketBaseURL,
		allowedUsers: []string{"*"},
		groupPolicy:  "open",
		groupAllowFrom: []string{},
		wsClient:     wsClient,
	}
}

func (c *WecomChannel) Name() string {
	return "wecom"
}

func (c *WecomChannel) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	if !c.tokenExpiry.IsZero() && time.Now().Before(c.tokenExpiry) && c.accessToken != "" {
		token := c.accessToken
		c.tokenMutex.RUnlock()
		return token, nil
	}
	c.tokenMutex.RUnlock()

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	if !c.tokenExpiry.IsZero() && time.Now().Before(c.tokenExpiry) && c.accessToken != "" {
		return c.accessToken, nil
	}

	url := fmt.Sprintf("%s/gettoken?corpid=%s&corpsecret=%s", weComAPIBase, c.botID, c.botSecret)
	resp, err := c.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	var result WecomAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode access token response: %w", err)
	}

	if result.Errcode != 0 {
		return "", fmt.Errorf("WeCom API error: %d - %s", result.Errcode, result.Errmsg)
	}

	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)

	log.Printf("WeCom: Access token updated, expires in %d seconds", result.ExpiresIn)

	return c.accessToken, nil
}

func (c *WecomChannel) Send(ctx context.Context, message *types.SendMessage) error {
	log.Printf("WeCom: Sending message to %s", message.Recipient)
	log.Printf("WeCom: Message content: %s", message.Content)

	if message.Recipient == "" {
		return fmt.Errorf("recipient is empty")
	}

	streamID := generateStreamID()
	content := message.Content
	
	if len(content) == 0 {
		return fmt.Errorf("message content is empty")
	}
	
	chunkSize := 10
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		
		chunk := content[:end]
		finish := end >= len(content)
		
		if err := c.sendStreamReply(message.Recipient, streamID, chunk, finish); err != nil {
			return fmt.Errorf("failed to send stream chunk: %w", err)
		}
		
		if !finish {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	return nil
}

func (c *WecomChannel) sendStreamReply(reqID, streamID, content string, finish bool) error {
	c.wsMutex.RLock()
	wsClient := c.wsClient
	c.wsMutex.RUnlock()
	
	if wsClient == nil || !wsClient.IsConnected() {
		return fmt.Errorf("WebSocket not connected")
	}
	
	body := map[string]interface{}{
		"msgtype": "stream",
		"stream": map[string]interface{}{
			"id":      streamID,
			"finish":  finish,
			"content": content,
		},
	}
	
	frame := &WecomWSFrame{
		Cmd: "aibot_respond_msg",
		Headers: map[string]interface{}{
			"req_id": reqID,
		},
		Body: body,
	}
	
	//log.Printf("WeCom: Sending stream reply, reqID=%s, streamID=%s, finish=%v", reqID, streamID, finish)
	
	return wsClient.SendFrame(frame)
}

func (c *WecomChannel) Listen(ctx context.Context, msgChan chan<- types.ChannelMessage) error {
	log.Printf("WeCom: Starting message listener (WebSocket mode)")

	err := c.connectWebSocket(ctx, msgChan)
	if err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	<-ctx.Done()
	log.Printf("WeCom: Listener stopped")
	return nil
}

func (c *WecomChannel) connectWebSocket(ctx context.Context, msgChan chan<- types.ChannelMessage) error {
	c.wsMutex.Lock()
	c.wsClient.msgChan = msgChan
	c.wsMutex.Unlock()
	
	c.wsClient.SetConnectedHandler(func() {
		log.Printf("WeCom: WebSocket connected")
		c.wsMutex.Lock()
		c.wsClient.connected = true
		c.wsMutex.Unlock()
		
		go func() {
			time.Sleep(100 * time.Millisecond)
			if err := c.wsClient.SendAuth(); err != nil {
				log.Printf("WeCom: failed to send auth: %v", err)
			}
		}()
	})
	
	c.wsClient.SetAuthenticatedHandler(func() {
		log.Printf("WeCom: Authentication successful")
		c.wsMutex.Lock()
		c.wsClient.authenticated = true
		c.wsMutex.Unlock()
	})
	
	c.wsClient.SetDisconnectedHandler(func(reason string) {
		log.Printf("WeCom: WebSocket disconnected: %s", reason)
		c.wsMutex.Lock()
		c.wsClient.connected = false
		c.wsClient.authenticated = false
		c.wsMutex.Unlock()
	})
	
	c.wsClient.SetErrorHandler(func(err error) {
		log.Printf("WeCom: WebSocket error: %v", err)
	})
	
	c.wsClient.SetMessageHandler(func(frame *WecomWSFrame) {
		c.handleWebSocketFrame(frame)
	})
	
	err := c.wsClient.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}
	
	log.Printf("WeCom: WebSocket connection started")
	return nil
}

func (c *WecomChannel) HealthCheck(ctx context.Context) error {
	log.Printf("WeCom: Health check")
	
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	
	if token == "" {
		return fmt.Errorf("no access token available")
	}
	
	log.Printf("WeCom: Health check passed")
	return nil
}

func (c *WecomChannel) handleWebSocketFrame(frame *WecomWSFrame) {
	if frame.Cmd == "aibot_msg_callback" {
		c.handleMessageCallback(frame)
	} else if frame.Cmd == "aibot_event_callback" {
		c.handleEventCallback(frame)
	} else if frame.Cmd == "" {
		c.handleResponseFrame(frame)
	}
}

func (c *WecomChannel) handleMessageCallback(frame *WecomWSFrame) {
	body, ok := frame.Body.(map[string]interface{})
	if !ok {
		log.Printf("WeCom: invalid message callback body")
		return
	}
	
	reqID := ""
	if frame.Headers != nil {
		if reqIDVal, ok := frame.Headers["req_id"]; ok {
			reqID = fmt.Sprintf("%v", reqIDVal)
		}
	}
	
	msg := &WecomMessage{
		MsgID:        fmt.Sprintf("%v", body["msgid"]),
		ChatID:       fmt.Sprintf("%v", body["chatid"]),
		ChatType:     fmt.Sprintf("%v", body["chattype"]),
		ResponseType: fmt.Sprintf("%v", body["response_url"]),
		MsgType:      fmt.Sprintf("%v", body["msgtype"]),
		ReqID:        reqID,
	}
	
	if from, ok := body["from"].(map[string]interface{}); ok {
		msg.From.UserID = fmt.Sprintf("%v", from["userid"])
	}
	
	if text, ok := body["text"].(map[string]interface{}); ok {
		msg.Text = &WecomText{
			Content: fmt.Sprintf("%v", text["content"]),
		}
	}
	
	if image, ok := body["image"].(map[string]interface{}); ok {
		msg.Image = &WecomImage{
			URL:     fmt.Sprintf("%v", image["url"]),
			AesKey:  fmt.Sprintf("%v", image["aeskey"]),
			Base64:  fmt.Sprintf("%v", image["base64"]),
			MD5:     fmt.Sprintf("%v", image["md5"]),
		}
	}
	
	if mixed, ok := body["mixed"].(map[string]interface{}); ok {
		if msgItems, ok := mixed["msg_item"].([]interface{}); ok {
			msg.Mixed = &WecomMixed{
				MsgItem: make([]WecomMixedItem, 0, len(msgItems)),
			}
			for _, item := range msgItems {
				if itemMap, ok := item.(map[string]interface{}); ok {
					mixedItem := WecomMixedItem{
						MsgType: fmt.Sprintf("%v", itemMap["msgtype"]),
					}
					if text, ok := itemMap["text"].(map[string]interface{}); ok {
						mixedItem.Text = &WecomText{
							Content: fmt.Sprintf("%v", text["content"]),
						}
					}
					if img, ok := itemMap["image"].(map[string]interface{}); ok {
						mixedItem.Image = &WecomImage{
							URL:     fmt.Sprintf("%v", img["url"]),
							AesKey:  fmt.Sprintf("%v", img["aeskey"]),
							Base64:  fmt.Sprintf("%v", img["base64"]),
							MD5:     fmt.Sprintf("%v", img["md5"]),
						}
					}
					msg.Mixed.MsgItem = append(msg.Mixed.MsgItem, mixedItem)
				}
			}
		}
	}
	
	if voice, ok := body["voice"].(map[string]interface{}); ok {
		msg.Voice = &WecomVoice{
			Content: fmt.Sprintf("%v", voice["content"]),
		}
	}
	
	if file, ok := body["file"].(map[string]interface{}); ok {
		msg.File = &WecomFile{
			URL:     fmt.Sprintf("%v", file["url"]),
			AesKey:  fmt.Sprintf("%v", file["aeskey"]),
		}
	}
	
	if quote, ok := body["quote"].(map[string]interface{}); ok {
		msg.Quote = &WecomQuote{
			MsgType: fmt.Sprintf("%v", quote["msgtype"]),
		}
		if text, ok := quote["text"].(map[string]interface{}); ok {
			msg.Quote.Text = &WecomText{
				Content: fmt.Sprintf("%v", text["content"]),
			}
		}
		if img, ok := quote["image"].(map[string]interface{}); ok {
			msg.Quote.Image = &WecomImage{
				URL:     fmt.Sprintf("%v", img["url"]),
				AesKey:  fmt.Sprintf("%v", img["aeskey"]),
			}
		}
		if fl, ok := quote["file"].(map[string]interface{}); ok {
			msg.Quote.File = &WecomFile{
				URL:     fmt.Sprintf("%v", fl["url"]),
				AesKey:  fmt.Sprintf("%v", fl["aeskey"]),
			}
		}
	}
	
	if err := c.processMessage(msg); err != nil {
		log.Printf("WeCom: failed to process message: %v", err)
	}
}

func (c *WecomChannel) handleEventCallback(frame *WecomWSFrame) {
	log.Printf("WeCom: Processing event callback")
	body, ok := frame.Body.(map[string]interface{})
	if !ok {
		log.Printf("WeCom: invalid event callback body")
		return
	}
	
	msg := &WecomMessage{
		MsgID:   fmt.Sprintf("%v", body["msgid"]),
		MsgType: "event",
	}
	
	if from, ok := body["from"].(map[string]interface{}); ok {
		msg.From.UserID = fmt.Sprintf("%v", from["userid"])
	}
	
	if chatID, ok := body["chatid"].(string); ok {
		msg.ChatID = chatID
	}
	
	if chatType, ok := body["chattype"].(string); ok {
		msg.ChatType = chatType
	}
	
	if event, ok := body["event"].(map[string]interface{}); ok {
		if eventType, ok := event["eventtype"].(string); ok {
			log.Printf("WeCom: Event type: %s", eventType)
		}
	}
	
	if err := c.processEventMessage(msg); err != nil {
		log.Printf("WeCom: failed to process event: %v", err)
	}
}

func (c *WecomChannel) handleResponseFrame(frame *WecomWSFrame) {
	reqID := ""
	if frame.Headers != nil {
		if reqIDVal, ok := frame.Headers["req_id"]; ok {
			reqID = fmt.Sprintf("%v", reqIDVal)
		}
	}
	
	if reqID == "" {
		return
	}
	
	if strings.HasPrefix(reqID, "aibot_subscribe") {
		if frame.Errcode == 0 {
			log.Printf("WeCom: Authentication successful")
			c.wsMutex.Lock()
			c.wsClient.authenticated = true
			c.wsMutex.Unlock()
		} else {
			log.Printf("WeCom: Authentication failed: %d - %s", frame.Errcode, frame.Errmsg)
		}
	} else if strings.HasPrefix(reqID, "ping") {
		if frame.Errcode != 0 {
			log.Printf("WeCom: Heartbeat failed: %d - %s", frame.Errcode, frame.Errmsg)
		}
	}
}

func (c *WecomChannel) StartTyping(ctx context.Context, recipient string) error {
	return nil
}

func (c *WecomChannel) StopTyping(ctx context.Context, recipient string) error {
	return nil
}

func (c *WecomChannel) SupportsDraftUpdates() bool {
	return false
}

func (c *WecomChannel) SendDraft(ctx context.Context, message *types.SendMessage) (string, error) {
	return "", fmt.Errorf("WeCom does not support draft updates")
}

func (c *WecomChannel) UpdateDraft(ctx context.Context, recipient, messageID, text string) (string, error) {
	return "", fmt.Errorf("WeCom does not support draft updates")
}

func (c *WecomChannel) FinalizeDraft(ctx context.Context, recipient, messageID, text string) error {
	return fmt.Errorf("WeCom does not support draft updates")
}

func (c *WecomChannel) CancelDraft(ctx context.Context, recipient, messageID string) error {
	return fmt.Errorf("WeCom does not support draft updates")
}

func (c *WecomChannel) AddReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return nil
}

func (c *WecomChannel) RemoveReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return nil
}

func (c *WecomChannel) isUserAllowed(userID string) bool {
	for _, user := range c.allowedUsers {
		if user == "*" || user == userID {
			return true
		}
	}
	return false
}

func (c *WecomChannel) isGroupAllowed(groupID string) bool {
	if c.groupPolicy == "disabled" {
		return false
	}
	if c.groupPolicy == "open" {
		return true
	}
	
	for _, group := range c.groupAllowFrom {
		if group == "*" || group == groupID {
			return true
		}
	}
	return false
}

func (c *WecomChannel) isGroupSenderAllowed(groupID, senderID string) bool {
	return true
}

func (c *WecomChannel) parseMessageContent(msg *WecomMessage) (text string, imageURLs []string, err error) {
	if msg.Text != nil {
		text = msg.Text.Content
	}
	
	if msg.Image != nil && msg.Image.URL != "" {
		imageURLs = append(imageURLs, msg.Image.URL)
	}
	
	if msg.Mixed != nil {
		for _, item := range msg.Mixed.MsgItem {
			if item.Text != nil {
				if text != "" {
					text += "\n"
				}
				text += item.Text.Content
			}
			if item.Image != nil && item.Image.URL != "" {
				imageURLs = append(imageURLs, item.Image.URL)
			}
		}
	}
	
	if msg.Voice != nil {
		if text != "" {
			text += "\n"
		}
		text += msg.Voice.Content
	}
	
	if msg.File != nil {
		if text != "" {
			text += "\n"
		}
		text += "[文件消息]"
	}
	
	if msg.Quote != nil {
		if msg.Quote.Text != nil {
			if text != "" {
				text += "\n"
			}
			text += "引用: " + msg.Quote.Text.Content
		}
	}
	
	text = strings.TrimSpace(text)
	
	if strings.HasPrefix(text, "@") {
		if spaceIndex := strings.Index(text, " "); spaceIndex != -1 {
			text = strings.TrimSpace(text[spaceIndex+1:])
		} else {
			text = ""
		}
	}
	
	return text, imageURLs, nil
}

func (c *WecomChannel) processMessage(msg *WecomMessage) error {
	log.Printf("WeCom: Processing message from %s, reqID=%s", msg.From.UserID, msg.ReqID)
	
	if !c.isUserAllowed(msg.From.UserID) {
		log.Printf("WeCom: User %s not allowed", msg.From.UserID)
		return nil
	}
	
	if msg.ChatType == "group" {
		if !c.isGroupAllowed(msg.ChatID) {
			log.Printf("WeCom: Group %s not allowed", msg.ChatID)
			return nil
		}
		
		if !c.isGroupSenderAllowed(msg.ChatID, msg.From.UserID) {
			log.Printf("WeCom: Sender %s not allowed in group %s", msg.From.UserID, msg.ChatID)
			return nil
		}
	}
	
	text, _, err := c.parseMessageContent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}
	
	if text == "" {
		log.Printf("WeCom: Empty message content")
		return nil
	}
	
	log.Printf("WeCom: Message text: %s", text)
	
	c.wsMutex.RLock()
	msgChan := c.wsClient.msgChan
	c.wsMutex.RUnlock()
	
	if msgChan == nil {
		log.Printf("WeCom: msgChan is nil")
		return fmt.Errorf("message channel not initialized")
	}
	
	channelMsg := types.ChannelMessage{
		ID:          msg.MsgID,
		Sender:      msg.From.UserID,
		ReplyTarget: msg.ReqID,
		Content:     text,
		Channel:     "wecom",
		Timestamp:   uint64(time.Now().Unix()),
	}
	
	select {
	case msgChan <- channelMsg:
		log.Printf("WeCom: Message processed: %s", msg.MsgID)
	default:
		return fmt.Errorf("message channel full")
	}
	
	return nil
}

func (c *WecomChannel) sendMessageToUser(ctx context.Context, userID, content string) error {
	return c.Send(ctx, &types.SendMessage{
		Content:   content,
		Recipient: userID,
	})
}

func (c *WecomChannel) sendMessageToGroup(ctx context.Context, groupID, content string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	
	payload := &WecomSendMessage{
		ToUser:  groupID,
		MsgType: "text",
		Text: &WecomText{
			Content: content,
		},
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}
	
	reqURL := fmt.Sprintf("%s/message/send?access_token=%s", weComAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil
}

func (c *WecomChannel) sendMarkdownMessage(ctx context.Context, userID, markdown string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	
	payload := &WecomSendMessage{
		ToUser:  userID,
		MsgType: "markdown",
		Markdown: &WecomMarkdown{
			Content: markdown,
		},
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}
	
	reqURL := fmt.Sprintf("%s/message/send?access_token=%s", weComAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil
}

func (c *WecomChannel) sendTemplateCardMessage(ctx context.Context, userID string, card *WecomTemplateCard) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	
	payload := &WecomSendMessage{
		ToUser:       userID,
		MsgType:      "template_card",
		TemplateCard: card,
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}
	
	reqURL := fmt.Sprintf("%s/message/send?access_token=%s", weComAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil
}

func (c *WecomChannel) downloadFile(ctx context.Context, url, aesKey string) ([]byte, string, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, "", err
	}
	
	reqURL := fmt.Sprintf("%s/media/get?access_token=%s&media_id=%s", weComAPIBase, token, url)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return nil, "", fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil, "", nil
}

func (c *WecomChannel) processImageMessage(ctx context.Context, msg *WecomMessage) ([]string, error) {
	var imagePaths []string
	
	if msg.Image != nil && msg.Image.URL != "" {
		token, err := c.getAccessToken(ctx)
		if err != nil {
			return nil, err
		}
		
		reqURL := fmt.Sprintf("%s/media/get?access_token=%s&media_id=%s", weComAPIBase, token, msg.Image.URL)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to download image: %w", err)
		}
		defer resp.Body.Close()
		
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		
		if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
			return nil, fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
		}
		
		imagePaths = append(imagePaths, msg.Image.URL)
	}
	
	if msg.Mixed != nil {
		for _, item := range msg.Mixed.MsgItem {
			if item.Image != nil && item.Image.URL != "" {
				imagePaths = append(imagePaths, item.Image.URL)
			}
		}
	}
	
	if msg.Quote != nil && msg.Quote.Image != nil && msg.Quote.Image.URL != "" {
		imagePaths = append(imagePaths, msg.Quote.Image.URL)
	}
	
	return imagePaths, nil
}

func (c *WecomChannel) processFileMessage(ctx context.Context, msg *WecomMessage) ([]string, error) {
	var fileURLs []string
	
	if msg.File != nil && msg.File.URL != "" {
		fileURLs = append(fileURLs, msg.File.URL)
	}
	
	if msg.Mixed != nil {
		for _, item := range msg.Mixed.MsgItem {
			if item.File != nil && item.File.URL != "" {
				fileURLs = append(fileURLs, item.File.URL)
			}
		}
	}
	
	if msg.Quote != nil && msg.Quote.File != nil && msg.Quote.File.URL != "" {
		fileURLs = append(fileURLs, msg.Quote.File.URL)
	}
	
	return fileURLs, nil
}

func (c *WecomChannel) sendThinkingMessage(ctx context.Context, chatID string) error {
	return c.sendMessageToUser(ctx, chatID, "⏳ 思考中...")
}

func (c *WecomChannel) sendWelcomeMessage(ctx context.Context, chatID string) error {
	return c.sendMessageToUser(ctx, chatID, "您好！我是智能助手，有什么可以帮您的吗？")
}

func (c *WecomChannel) sendErrorMessage(ctx context.Context, chatID, errorMsg string) error {
	return c.sendMessageToUser(ctx, chatID, fmt.Sprintf("❌ 错误: %s", errorMsg))
}

func (c *WecomChannel) sendSuccessMessage(ctx context.Context, chatID string) error {
	return c.sendMessageToUser(ctx, chatID, "✅ 已完成")
}

func (c *WecomChannel) sendProgressMessage(ctx context.Context, chatID, progress string) error {
	return c.sendMessageToUser(ctx, chatID, fmt.Sprintf("⏳ %s", progress))
}

func (c *WecomChannel) sendFinalMessage(ctx context.Context, chatID, content string) error {
	return c.sendMessageToUser(ctx, chatID, content)
}

func (c *WecomChannel) sendStreamMessage(ctx context.Context, chatID, content string, finish bool) error {
	if finish {
		return c.sendFinalMessage(ctx, chatID, content)
	}
	return c.sendThinkingMessage(ctx, chatID)
}

func (c *WecomChannel) sendMarkdownStreamMessage(ctx context.Context, chatID, content string, finish bool) error {
	if finish {
		return c.sendMarkdownMessage(ctx, chatID, content)
	}
	return c.sendThinkingMessage(ctx, chatID)
}

func (c *WecomChannel) sendTemplateCardStreamMessage(ctx context.Context, chatID string, card *WecomTemplateCard, finish bool) error {
	if finish {
		return c.sendTemplateCardMessage(ctx, chatID, card)
	}
	return c.sendThinkingMessage(ctx, chatID)
}

func (c *WecomChannel) sendStreamWithCardMessage(ctx context.Context, chatID, content string, card *WecomTemplateCard, finish bool) error {
	if finish {
		return c.sendTemplateCardMessage(ctx, chatID, card)
	}
	return c.sendThinkingMessage(ctx, chatID)
}

func (c *WecomChannel) updateTemplateCardMessage(ctx context.Context, chatID string, card *WecomTemplateCard) error {
	return c.sendTemplateCardMessage(ctx, chatID, card)
}

func (c *WecomChannel) cancelDraftMessage(ctx context.Context, chatID string) error {
	return nil
}

func (c *WecomChannel) finalizeDraftMessage(ctx context.Context, chatID, content string) error {
	return c.sendFinalMessage(ctx, chatID, content)
}

func (c *WecomChannel) updateDraftMessage(ctx context.Context, chatID, content string) error {
	return c.sendProgressMessage(ctx, chatID, content)
}

func (c *WecomChannel) sendDraftMessage(ctx context.Context, chatID, content string) (string, error) {
	err := c.sendThinkingMessage(ctx, chatID)
	if err != nil {
		return "", err
	}
	return "draft_" + time.Now().Format("20060102150405"), nil
}

func (c *WecomChannel) isGroupPolicyEnabled() bool {
	return c.groupPolicy != "disabled"
}

func (c *WecomChannel) isGroupAllowedByPolicy(groupID string) bool {
	if !c.isGroupPolicyEnabled() {
		return false
	}
	return c.isGroupAllowed(groupID)
}

func (c *WecomChannel) isSenderAllowedByGroupPolicy(groupID, senderID string) bool {
	if !c.isGroupPolicyEnabled() {
		return false
	}
	return c.isGroupSenderAllowed(groupID, senderID)
}

func (c *WecomChannel) isDMAllowed(dmPolicy string, senderID string) bool {
	if dmPolicy == "disabled" {
		return false
	}
	if dmPolicy == "open" {
		return true
	}
	return c.isUserAllowed(senderID)
}

func (c *WecomChannel) checkGroupPolicy(chatID, senderID string) bool {
	if !c.isGroupAllowed(chatID) {
		log.Printf("WeCom: Group %s not allowed", chatID)
		return false
	}
	if !c.isGroupSenderAllowed(chatID, senderID) {
		log.Printf("WeCom: Sender %s not allowed in group %s", senderID, chatID)
		return false
	}
	return true
}

func (c *WecomChannel) checkDMPolicy(dmPolicy, senderID string) bool {
	return c.isDMAllowed(dmPolicy, senderID)
}

func (c *WecomChannel) processEventMessage(msg *WecomMessage) error {
	if msg.MsgType == "event" {
		log.Printf("WeCom: Processing event message: %s", msg.MsgID)
	}
	return nil
}

func (c *WecomChannel) processMixedMessage(msg *WecomMessage) error {
	if msg.Mixed != nil {
		log.Printf("WeCom: Processing mixed message with %d items", len(msg.Mixed.MsgItem))
	}
	return nil
}

func (c *WecomChannel) processQuoteMessage(msg *WecomMessage) error {
	if msg.Quote != nil {
		log.Printf("WeCom: Processing quote message: %s", msg.Quote.MsgType)
	}
	return nil
}

func (c *WecomChannel) processVoiceMessage(msg *WecomMessage) error {
	if msg.Voice != nil {
		log.Printf("WeCom: Processing voice message")
	}
	return nil
}

func (c *WecomChannel) processFileMessageContent(msg *WecomMessage) error {
	if msg.File != nil {
		log.Printf("WeCom: Processing file message")
	}
	return nil
}

func (c *WecomChannel) processImageMessageContent(msg *WecomMessage) error {
	if msg.Image != nil {
		log.Printf("WeCom: Processing image message")
	}
	return nil
}

func (c *WecomChannel) processTextMessageContent(msg *WecomMessage) error {
	if msg.Text != nil {
		log.Printf("WeCom: Processing text message: %s", msg.Text.Content)
	}
	return nil
}

func (c *WecomChannel) processMessageContent(msg *WecomMessage) error {
	if err := c.processTextMessageContent(msg); err != nil {
		return err
	}
	if err := c.processImageMessageContent(msg); err != nil {
		return err
	}
	if err := c.processVoiceMessage(msg); err != nil {
		return err
	}
	if err := c.processFileMessageContent(msg); err != nil {
		return err
	}
	if err := c.processMixedMessage(msg); err != nil {
		return err
	}
	if err := c.processQuoteMessage(msg); err != nil {
		return err
	}
	if err := c.processEventMessage(msg); err != nil {
		return err
	}
	return nil
}

func (c *WecomChannel) processIncomingMessage(ctx context.Context, msg *WecomMessage) error {
	if err := c.processMessageContent(msg); err != nil {
		return err
	}
	
	if msg.ChatType == "group" {
		if !c.checkGroupPolicy(msg.ChatID, msg.From.UserID) {
			return nil
		}
	} else {
		if !c.checkDMPolicy("open", msg.From.UserID) {
			return nil
		}
	}
	
	text, _, err := c.parseMessageContent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}
	
	if text == "" {
		log.Printf("WeCom: Empty message content")
		return nil
	}
	
	channelMsg := types.ChannelMessage{
		ID:        msg.MsgID,
		Sender:    msg.From.UserID,
		ReplyTarget: msg.ChatID,
		Content:   text,
		Channel:   "wecom",
		Timestamp: uint64(time.Now().Unix()),
	}
	
	select {
	case c.wsClient.msgChan <- channelMsg:
		log.Printf("WeCom: Message processed: %s", msg.MsgID)
	default:
		return fmt.Errorf("message channel full")
	}
	
	return nil
}

func (c *WecomChannel) SendMessageToChat(ctx context.Context, chatID, content string) error {
	c.wsMutex.RLock()
	wsClient := c.wsClient
	var authenticated bool
	if wsClient != nil {
		authenticated = wsClient.authenticated
	}
	c.wsMutex.RUnlock()
	
	if wsClient == nil || !authenticated {
		return fmt.Errorf("WebSocket not connected or not authenticated")
	}
	
	reqID := generateReqID("aibot_send_msg")
	
	body := map[string]interface{}{
		"chatid":   chatID,
		"msgtype":  "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	
	frame := &WecomWSFrame{
		Cmd: "aibot_send_msg",
		Headers: map[string]interface{}{
			"req_id": reqID,
		},
		Body: body,
	}
	
	log.Printf("WeCom: Sending message to chat %s", chatID)
	return wsClient.SendFrame(frame)
}

func (c *WecomChannel) IsConnected() bool {
	c.wsMutex.RLock()
	defer c.wsMutex.RUnlock()
	return c.wsClient != nil && c.wsClient.connected && c.wsClient.authenticated
}

func (c *WecomChannel) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(weComHeartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("WeCom: Heartbeat stopped")
			return
		case <-ticker.C:
			log.Printf("WeCom: Sending heartbeat")
		}
	}
}

func (c *WecomChannel) connect(ctx context.Context) error {
	log.Printf("WeCom: Connecting to WebSocket: %s", c.websocketURL)
	
	c.wsMutex.Lock()
	c.wsClient.connected = true
	c.wsMutex.Unlock()
	
	log.Printf("WeCom: WebSocket connected successfully")
	return nil
}

func (c *WecomChannel) disconnect() {
	c.wsMutex.Lock()
	defer c.wsMutex.Unlock()
	
	if c.wsClient.connected {
		c.wsClient.connected = false
		log.Printf("WeCom: WebSocket disconnected")
	}
}

func (c *WecomChannel) isConnected() bool {
	c.wsMutex.RLock()
	defer c.wsMutex.RUnlock()
	return c.wsClient.connected
}

func (c *WecomChannel) setConnected(connected bool) {
	c.wsMutex.Lock()
	defer c.wsMutex.Unlock()
	c.wsClient.connected = connected
}

func (c *WecomChannel) getBotID() string {
	return c.botID
}

func (c *WecomChannel) getBotSecret() string {
	return c.botSecret
}

func (c *WecomChannel) getWebSocketURL() string {
	return c.websocketURL
}

func (c *WecomChannel) setWebSocketURL(url string) {
	c.websocketURL = url
}

func (c *WecomChannel) getAllowedUsers() []string {
	return c.allowedUsers
}

func (c *WecomChannel) setAllowedUsers(users []string) {
	c.allowedUsers = users
}

func (c *WecomChannel) getGroupPolicy() string {
	return c.groupPolicy
}

func (c *WecomChannel) setGroupPolicy(policy string) {
	c.groupPolicy = policy
}

func (c *WecomChannel) getGroupAllowFrom() []string {
	return c.groupAllowFrom
}

func (c *WecomChannel) setGroupAllowFrom(groups []string) {
	c.groupAllowFrom = groups
}

func (c *WecomChannel) getAccessTokenFromCache() string {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.accessToken
}

func (c *WecomChannel) setAccessToken(token string, expiry time.Time) {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()
	c.accessToken = token
	c.tokenExpiry = expiry
}

func (c *WecomChannel) clearAccessToken() {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()
	c.accessToken = ""
	c.tokenExpiry = time.Time{}
}

func (c *WecomChannel) isTokenValid() bool {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return !c.tokenExpiry.IsZero() && time.Now().Before(c.tokenExpiry) && c.accessToken != ""
}

func (c *WecomChannel) refreshAccessToken(ctx context.Context) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	c.setAccessToken(token, c.tokenExpiry)
	return nil
}

func (c *WecomChannel) sendMessage(ctx context.Context, message *types.SendMessage) error {
	return c.Send(ctx, message)
}

func (c *WecomChannel) sendMessageToRecipient(ctx context.Context, recipient, content string) error {
	return c.Send(ctx, &types.SendMessage{
		Content:   content,
		Recipient: recipient,
	})
}

func (c *WecomChannel) sendMessageToChat(ctx context.Context, chatID, content string) error {
	if chatID == "" {
		return fmt.Errorf("chat ID is empty")
	}
	
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	
	payload := &WecomSendMessage{
		ToUser:  chatID,
		MsgType: "text",
		Text: &WecomText{
			Content: content,
		},
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}
	
	reqURL := fmt.Sprintf("%s/message/send?access_token=%s", weComAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil
}

func (c *WecomChannel) sendMarkdownToChat(ctx context.Context, chatID, markdown string) error {
	if chatID == "" {
		return fmt.Errorf("chat ID is empty")
	}
	
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	
	payload := &WecomSendMessage{
		ToUser:  chatID,
		MsgType: "markdown",
		Markdown: &WecomMarkdown{
			Content: markdown,
		},
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}
	
	reqURL := fmt.Sprintf("%s/message/send?access_token=%s", weComAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil
}

func (c *WecomChannel) sendTemplateCardToChat(ctx context.Context, chatID string, card *WecomTemplateCard) error {
	if chatID == "" {
		return fmt.Errorf("chat ID is empty")
	}
	
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	
	payload := &WecomSendMessage{
		ToUser:       chatID,
		MsgType:      "template_card",
		TemplateCard: card,
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}
	
	reqURL := fmt.Sprintf("%s/message/send?access_token=%s", weComAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil
}

func (c *WecomChannel) downloadImage(ctx context.Context, url string) ([]byte, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	
	reqURL := fmt.Sprintf("%s/media/get?access_token=%s&media_id=%s", weComAPIBase, token, url)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return nil, fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil, nil
}

func (c *WecomChannel) downloadFileToBuffer(ctx context.Context, url string) ([]byte, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	
	reqURL := fmt.Sprintf("%s/media/get?access_token=%s&media_id=%s", weComAPIBase, token, url)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		return nil, fmt.Errorf("WeCom API error: %v - %s", errcode, result["errmsg"])
	}
	
	return nil, nil
}

func (c *WecomChannel) processMixedMessageContent(msg *WecomMessage) error {
	if msg.Mixed != nil {
		for _, item := range msg.Mixed.MsgItem {
			if item.Text != nil {
				log.Printf("WeCom: Mixed message text: %s", item.Text.Content)
			}
			if item.Image != nil {
				log.Printf("WeCom: Mixed message image")
			}
		}
	}
	return nil
}

func (c *WecomChannel) processQuoteContent(msg *WecomMessage) error {
	if msg.Quote != nil {
		if msg.Quote.Text != nil {
			log.Printf("WeCom: Quote text: %s", msg.Quote.Text.Content)
		}
		if msg.Quote.Image != nil {
			log.Printf("WeCom: Quote image")
		}
		if msg.Quote.File != nil {
			log.Printf("WeCom: Quote file")
		}
	}
	return nil
}

func (c *WecomChannel) processVoiceContent(msg *WecomMessage) error {
	if msg.Voice != nil {
		log.Printf("WeCom: Voice content: %s", msg.Voice.Content)
	}
	return nil
}

func (c *WecomChannel) processFileContent(msg *WecomMessage) error {
	if msg.File != nil {
		log.Printf("WeCom: File URL: %s", msg.File.URL)
	}
	return nil
}

func (c *WecomChannel) processImageContent(msg *WecomMessage) error {
	if msg.Image != nil {
		log.Printf("WeCom: Image URL: %s", msg.Image.URL)
	}
	return nil
}

func (c *WecomChannel) processTextContent(msg *WecomMessage) error {
	if msg.Text != nil {
		log.Printf("WeCom: Text content: %s", msg.Text.Content)
	}
	return nil
}

func (c *WecomChannel) processAllContent(msg *WecomMessage) error {
	if err := c.processTextContent(msg); err != nil {
		return err
	}
	if err := c.processImageContent(msg); err != nil {
		return err
	}
	if err := c.processVoiceContent(msg); err != nil {
		return err
	}
	if err := c.processFileContent(msg); err != nil {
		return err
	}
	if err := c.processMixedMessageContent(msg); err != nil {
		return err
	}
	if err := c.processQuoteContent(msg); err != nil {
		return err
	}
	return nil
}

func (c *WecomChannel) processMessageToChannel(msg *WecomMessage) error {
	if !c.isUserAllowed(msg.From.UserID) {
		log.Printf("WeCom: User %s not allowed", msg.From.UserID)
		return nil
	}
	
	if msg.ChatType == "group" {
		if !c.isGroupAllowed(msg.ChatID) {
			log.Printf("WeCom: Group %s not allowed", msg.ChatID)
			return nil
		}
		
		if !c.isGroupSenderAllowed(msg.ChatID, msg.From.UserID) {
			log.Printf("WeCom: Sender %s not allowed in group %s", msg.From.UserID, msg.ChatID)
			return nil
		}
	}
	
	text, _, err := c.parseMessageContent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}
	
	if text == "" {
		log.Printf("WeCom: Empty message content")
		return nil
	}
	
	channelMsg := types.ChannelMessage{
		ID:        msg.MsgID,
		Sender:    msg.From.UserID,
		ReplyTarget: msg.ChatID,
		Content:   text,
		Channel:   "wecom",
		Timestamp: uint64(time.Now().Unix()),
	}
	
	select {
	case c.wsClient.msgChan <- channelMsg:
		log.Printf("WeCom: Message processed: %s", msg.MsgID)
	default:
		return fmt.Errorf("message channel full")
	}
	
	return nil
}
