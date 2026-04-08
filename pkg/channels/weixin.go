package channels

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

const (
	DefaultBaseURL           = "https://ilinkai.weixin.qq.com"
	CDNBaseURL               = "https://novac2c.cdn.weixin.qq.com/c2c"
	DefaultBotType           = "3"
	SessionExpiredErrCode    = -14
	MaxConsecutiveFailures   = 3
	BackoffDelayMs           = 30000
	RetryDelayMs             = 2000
	DefaultLongPollTimeoutMs = 35000
	DefaultAPITimeoutMs      = 15000
)

type WeixinChannel struct {
	botToken     string
	baseURL      string
	cdnBaseURL   string
	accountID    string
	allowedUsers []string
	httpClient   *http.Client

	mu            sync.Mutex
	contextTokens map[string]string
	running       bool
	abortChan     chan struct{}

	accountManager *WeixinAccountManager
}

type WeixinConfig struct {
	Token        string
	BaseURL      string
	CDNBaseURL   string
	AccountID    string
	AllowedUsers []string
	StateDir     string
}

func NewWeixinChannel(cfg *WeixinConfig) *WeixinChannel {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	cdnBaseURL := cfg.CDNBaseURL
	if cdnBaseURL == "" {
		cdnBaseURL = CDNBaseURL
	}

	stateDir := cfg.StateDir
	if stateDir == "" {
		homeDir, _ := os.UserHomeDir()
		stateDir = filepath.Join(homeDir, ".goclaw")
	}

	ch := &WeixinChannel{
		botToken:       cfg.Token,
		baseURL:        baseURL,
		cdnBaseURL:     cdnBaseURL,
		accountID:      cfg.AccountID,
		allowedUsers:   cfg.AllowedUsers,
		httpClient:     &http.Client{Timeout: 60 * time.Second},
		contextTokens:  make(map[string]string),
		abortChan:      make(chan struct{}),
		accountManager: NewWeixinAccountManager(stateDir),
	}

	if cfg.Token == "" {
		ch.autoLoadAccount()
	}

	return ch
}

func (c *WeixinChannel) autoLoadAccount() {
	accountIDs := c.accountManager.ListAccountIDs()
	if len(accountIDs) == 0 {
		return
	}

	accountID := accountIDs[0]
	if c.accountID != "" {
		for _, id := range accountIDs {
			if id == c.accountID {
				accountID = id
				break
			}
		}
	}

	account, err := c.accountManager.LoadAccount(accountID)
	if err != nil {
		return
	}

	if account.Token != "" {
		c.botToken = account.Token
		c.accountID = accountID
		if account.BaseURL != "" {
			c.baseURL = account.BaseURL
		}
	}
}

func (c *WeixinChannel) Name() string {
	return "weixin"
}

func (c *WeixinChannel) Send(ctx context.Context, message *types.SendMessage) error {
	if c.botToken == "" {
		return fmt.Errorf("weixin: 未登录，请先运行登录命令扫描二维码")
	}

	to := message.Recipient
	if to == "" {
		return fmt.Errorf("weixin: recipient is required")
	}

	userID := to
	if idx := strings.Index(to, ":"); idx > 0 {
		userID = to[:idx]
	}

	contextToken := c.getContextToken(c.accountID, userID)
	if contextToken == "" {
		log.Printf("[weixin] 未找到 context token, accountID=%s, userID=%s", c.accountID, userID)
		return fmt.Errorf("weixin: context token is required for sending messages")
	}

	clientID := generateClientID()
	req := &SendMessageReq{
		Msg: &WeixinMessage{
			FromUserID:  "",
			ToUserID:    userID,
			ClientID:    clientID,
			MessageType: MessageTypeBot,
			MessageState: MessageStateFinish,
			ItemList: []*MessageItem{
				{
					Type:     MessageItemTypeText,
					TextItem: &TextItem{Text: message.Content},
				},
			},
			ContextToken: contextToken,
		},
	}

	return c.sendMessageAPI(ctx, req)
}

func (c *WeixinChannel) Listen(ctx context.Context, msgChan chan<- types.ChannelMessage) error {
	if c.botToken == "" {
		return fmt.Errorf("weixin: 未登录，请先运行登录命令扫描二维码")
	}

	log.Printf("[weixin] Listen 启动, accountID=%s, baseURL=%s", c.accountID, c.baseURL)

	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		log.Printf("[weixin] Listen 停止")
	}()

	var getUpdatesBuf string
	consecutiveFailures := 0
	timeoutMs := DefaultLongPollTimeoutMs
	pollCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.abortChan:
			return nil
		default:
		}

		pollCount++
		if pollCount%10 == 1 {
			log.Printf("[weixin] 轮询中... (第 %d 次, buf=%s)", pollCount, getUpdatesBuf)
		}

		resp, err := c.getUpdates(ctx, getUpdatesBuf, timeoutMs)
		if err != nil {
			log.Printf("[weixin] getUpdates 错误: %v", err)
			consecutiveFailures++
			if consecutiveFailures >= MaxConsecutiveFailures {
				log.Printf("[weixin] 连续失败 %d 次，等待重试", consecutiveFailures)
				time.Sleep(BackoffDelayMs * time.Millisecond)
				consecutiveFailures = 0
			} else {
				time.Sleep(RetryDelayMs * time.Millisecond)
			}
			continue
		}

		consecutiveFailures = 0

		if resp.LongPollingTimeoutMs > 0 {
			timeoutMs = resp.LongPollingTimeoutMs
		}

		if resp.GetUpdatesBuf != "" {
			getUpdatesBuf = resp.GetUpdatesBuf
		}

		if len(resp.Msgs) > 0 {
			log.Printf("[weixin] 收到 %d 条消息", len(resp.Msgs))
		}

		for _, msg := range resp.Msgs {
			channelMsg := c.parseMessage(msg)
			if channelMsg != nil {
				if msg.ContextToken != "" {
					c.setContextToken(c.accountID, msg.FromUserID, msg.ContextToken)
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case msgChan <- *channelMsg:
				}
			}
		}
	}
}

func (c *WeixinChannel) HealthCheck(ctx context.Context) error {
	if c.botToken == "" {
		return fmt.Errorf("weixin: 未登录")
	}
	return nil
}

func (c *WeixinChannel) StartTyping(ctx context.Context, recipient string) error {
	return nil
}

func (c *WeixinChannel) StopTyping(ctx context.Context, recipient string) error {
	return nil
}

func (c *WeixinChannel) SupportsDraftUpdates() bool {
	return false
}

func (c *WeixinChannel) SendDraft(ctx context.Context, message *types.SendMessage) (string, error) {
	return "", fmt.Errorf("draft updates not supported")
}

func (c *WeixinChannel) UpdateDraft(ctx context.Context, recipient, messageID, text string) (string, error) {
	return "", fmt.Errorf("draft updates not supported")
}

func (c *WeixinChannel) FinalizeDraft(ctx context.Context, recipient, messageID, text string) error {
	return fmt.Errorf("draft updates not supported")
}

func (c *WeixinChannel) CancelDraft(ctx context.Context, recipient, messageID string) error {
	return fmt.Errorf("draft updates not supported")
}

func (c *WeixinChannel) AddReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return fmt.Errorf("reactions not supported")
}

func (c *WeixinChannel) RemoveReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return fmt.Errorf("reactions not supported")
}

func (c *WeixinChannel) getUpdates(ctx context.Context, buf string, timeoutMs int) (*GetUpdatesResp, error) {
	req := &GetUpdatesReq{
		GetUpdatesBuf: buf,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBody, err := c.apiFetch(ctx, "ilink/bot/getupdates", body, timeoutMs)
	if err != nil {
		return nil, err
	}

	var result GetUpdatesResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if result.ErrCode == SessionExpiredErrCode {
		return nil, fmt.Errorf("session expired")
	}

	return &result, nil
}

func (c *WeixinChannel) sendMessageAPI(ctx context.Context, req *SendMessageReq) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = c.apiFetch(ctx, "ilink/bot/sendmessage", body, DefaultAPITimeoutMs)
	return err
}

func (c *WeixinChannel) apiFetch(ctx context.Context, endpoint string, body []byte, timeoutMs int) ([]byte, error) {
	base := c.baseURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	url := base + endpoint
	log.Printf("[weixin] API 请求: %s", url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	if c.botToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.botToken)
	}

	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[weixin] API 请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("[weixin] API 响应: status=%d, body=%s", resp.StatusCode, string(respBody))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *WeixinChannel) parseMessage(msg *WeixinMessage) *types.ChannelMessage {
	if msg == nil {
		return nil
	}

	text := extractTextBody(msg.ItemList)
	if text == "" {
		return nil
	}

	from := msg.FromUserID
	if from == "" {
		return nil
	}

	return &types.ChannelMessage{
		ID:          fmt.Sprintf("%d", msg.MessageID),
		Sender:      from,
		ReplyTarget: from + ":" + fmt.Sprintf("%d", msg.MessageID),
		Content:     text,
		Channel:     "weixin",
		Timestamp:   uint64(msg.CreateTimeMs / 1000),
	}
}

func (c *WeixinChannel) setContextToken(accountID, userID, token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := accountID + ":" + userID
	c.contextTokens[key] = token
}

func (c *WeixinChannel) getContextToken(accountID, userID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := accountID + ":" + userID
	return c.contextTokens[key]
}

func (c *WeixinChannel) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		close(c.abortChan)
		c.abortChan = make(chan struct{})
	}
}

// DeleteAccount deletes the saved account data
func (c *WeixinChannel) DeleteAccount(accountID string) error {
	return c.accountManager.DeleteAccount(accountID)
}

func (c *WeixinChannel) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.botToken = token
}

func (c *WeixinChannel) IsConfigured() bool {
	return c.botToken != ""
}

func (c *WeixinChannel) GetAccountID() string {
	return c.accountID
}

// SetBotToken sets the bot token and account ID after successful login
func (c *WeixinChannel) SetBotToken(token, accountID string) {
	c.botToken = token
	c.accountID = accountID
}

func extractTextBody(items []*MessageItem) string {
	for _, item := range items {
		if item.Type == MessageItemTypeText && item.TextItem != nil {
			text := item.TextItem.Text
			if item.RefMsg != nil {
				refText := extractTextBody([]*MessageItem{item.RefMsg.MessageItem})
				if refText != "" {
					return fmt.Sprintf("[引用: %s]\n%s", refText, text)
				}
			}
			return text
		}
		if item.Type == MessageItemTypeVoice && item.VoiceItem != nil && item.VoiceItem.Text != "" {
			return item.VoiceItem.Text
		}
	}
	return ""
}

func generateClientID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("weixin_%x", b[:8])
}

func (c *WeixinChannel) SendMedia(ctx context.Context, to, text, filePath string) error {
	if c.botToken == "" {
		return fmt.Errorf("weixin: 未登录，请先运行登录命令扫描二维码")
	}

	contextToken := c.getContextToken(c.accountID, to)
	if contextToken == "" {
		return fmt.Errorf("weixin: context token is required for sending media")
	}

	uploaded, err := c.uploadFile(ctx, filePath, to)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	mimeType := getMimeType(filePath)
	var req *SendMessageReq

	if strings.HasPrefix(mimeType, "image/") {
		req = &SendMessageReq{
			Msg: &WeixinMessage{
				ToUserID:     to,
				ClientID:     generateClientID(),
				MessageType:  MessageTypeBot,
				MessageState: MessageStateFinish,
				ItemList: []*MessageItem{
					{
						Type: MessageItemTypeImage,
						ImageItem: &ImageItem{
							Media: &CDNMedia{
								EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
								AesKey:            base64.StdEncoding.EncodeToString(uploaded.AesKey),
								EncryptType:       1,
							},
							MidSize: uploaded.FileSizeCiphertext,
						},
					},
				},
				ContextToken: contextToken,
			},
		}
	} else if strings.HasPrefix(mimeType, "video/") {
		req = &SendMessageReq{
			Msg: &WeixinMessage{
				ToUserID:     to,
				ClientID:     generateClientID(),
				MessageType:  MessageTypeBot,
				MessageState: MessageStateFinish,
				ItemList: []*MessageItem{
					{
						Type: MessageItemTypeVideo,
						VideoItem: &VideoItem{
							Media: &CDNMedia{
								EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
								AesKey:            base64.StdEncoding.EncodeToString(uploaded.AesKey),
								EncryptType:       1,
							},
							VideoSize: uploaded.FileSizeCiphertext,
						},
					},
				},
				ContextToken: contextToken,
			},
		}
	} else {
		fileName := filepath.Base(filePath)
		req = &SendMessageReq{
			Msg: &WeixinMessage{
				ToUserID:     to,
				ClientID:     generateClientID(),
				MessageType:  MessageTypeBot,
				MessageState: MessageStateFinish,
				ItemList: []*MessageItem{
					{
						Type: MessageItemTypeFile,
						FileItem: &FileItem{
							Media: &CDNMedia{
								EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
								AesKey:            base64.StdEncoding.EncodeToString(uploaded.AesKey),
								EncryptType:       1,
							},
							FileName: fileName,
							Len:      fmt.Sprintf("%d", uploaded.FileSize),
						},
					},
				},
				ContextToken: contextToken,
			},
		}
	}

	if text != "" {
		textItem := &MessageItem{
			Type:     MessageItemTypeText,
			TextItem: &TextItem{Text: text},
		}
		req.Msg.ItemList = append([]*MessageItem{textItem}, req.Msg.ItemList...)
	}

	return c.sendMessageAPI(ctx, req)
}

type UploadedFileInfo struct {
	DownloadEncryptedQueryParam string
	AesKey                      []byte
	FileSize                    int
	FileSizeCiphertext          int
	FileKey                     string
}

func (c *WeixinChannel) uploadFile(ctx context.Context, filePath, toUserID string) (*UploadedFileInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	aesKey := make([]byte, 16)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, err
	}

	encrypted, err := aesEncrypt(data, aesKey)
	if err != nil {
		return nil, err
	}

	fileKey := generateFileKey()
	rawMD5 := md5.Sum(data)
	rawFileMD5 := hex.EncodeToString(rawMD5[:])

	uploadReq := &GetUploadUrlReq{
		FileKey:    fileKey,
		MediaType:  UploadMediaTypeFile,
		ToUserID:   toUserID,
		RawSize:    len(data),
		RawFileMD5: rawFileMD5,
		FileSize:   len(encrypted),
		AesKey:     hex.EncodeToString(aesKey),
	}

	uploadURLResp, err := c.getUploadURL(ctx, uploadReq)
	if err != nil {
		return nil, err
	}

	if err := c.uploadToCDN(ctx, uploadURLResp.UploadParam, encrypted); err != nil {
		return nil, err
	}

	return &UploadedFileInfo{
		DownloadEncryptedQueryParam: uploadURLResp.UploadParam,
		AesKey:                      aesKey,
		FileSize:                    len(data),
		FileSizeCiphertext:          len(encrypted),
		FileKey:                     fileKey,
	}, nil
}

func (c *WeixinChannel) getUploadURL(ctx context.Context, req *GetUploadUrlReq) (*GetUploadUrlResp, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBody, err := c.apiFetch(ctx, "ilink/bot/getuploadurl", body, DefaultAPITimeoutMs)
	if err != nil {
		return nil, err
	}

	var result GetUploadUrlResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *WeixinChannel) uploadToCDN(ctx context.Context, uploadParam string, data []byte) error {
	uploadURL := c.cdnBaseURL + "/upload?param=" + uploadParam

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CDN upload failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func generateFileKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func aesEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := append(plaintext, bytes.Repeat([]byte{byte(padding)}, padding)...)

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, key)
	mode.CryptBlocks(ciphertext, padded)

	return ciphertext, nil
}

func getMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

func (c *WeixinChannel) StartQRLogin(ctx context.Context) (*QRStartResult, error) {
	qrManager := NewQRLoginManager(c)
	return qrManager.StartLogin(ctx, "", c.baseURL, false)
}

// PollQRStatus polls the QR code login status directly using the qrCode string
func (c *WeixinChannel) PollQRStatus(ctx context.Context, qrCode string) (*QRStatusResponse, error) {
	qrManager := NewQRLoginManager(c)
	return qrManager.PollQRStatus(ctx, c.baseURL, qrCode)
}

// SaveAccountFromLogin saves account data after successful QR login
func (c *WeixinChannel) SaveAccountFromLogin(accountID, token, baseURL, userID string) error {
	normalizedID := NormalizeAccountID(accountID)
	account := &WeixinAccountData{
		Token:   token,
		BaseURL: baseURL,
		UserID:  userID,
	}
	return c.accountManager.SaveAccount(normalizedID, account)
}

func (c *WeixinChannel) WaitForQRLogin(ctx context.Context, sessionKey string, timeoutMs int) error {
	qrManager := NewQRLoginManager(c)
	result, err := qrManager.WaitForLogin(ctx, sessionKey, c.baseURL, timeoutMs)
	if err != nil {
		return err
	}

	if !result.Connected {
		return fmt.Errorf("login failed: %s", result.Message)
	}

	accountID := NormalizeAccountID(result.AccountID)
	account := &WeixinAccountData{
		Token:   result.BotToken,
		BaseURL: result.BaseURL,
		UserID:  result.UserID,
	}

	if err := c.accountManager.SaveAccount(accountID, account); err != nil {
		return fmt.Errorf("save account failed: %w", err)
	}

	c.botToken = result.BotToken
	c.accountID = accountID
	if result.BaseURL != "" {
		c.baseURL = result.BaseURL
	}

	return nil
}

func (c *WeixinChannel) LoginWithQR(ctx context.Context) error {
	qrManager := NewQRLoginManager(c)
	
	result, err := qrManager.StartLogin(ctx, "", c.baseURL, false)
	if err != nil {
		return err
	}

	if result.QRCodeURL == "" {
		return fmt.Errorf("failed to get QR code: %s", result.Message)
	}

	fmt.Println("\n请使用微信扫描二维码登录...")
	fmt.Printf("二维码链接: %s\n", result.QRCodeURL)
	fmt.Println("正在打开浏览器...")

	if err := openBrowser(result.QRCodeURL); err != nil {
		fmt.Printf("无法自动打开浏览器，请手动访问: %s\n", result.QRCodeURL)
	}

	waitResult, err := qrManager.WaitForLogin(ctx, result.SessionKey, c.baseURL, 480000)
	if err != nil {
		return err
	}

	if !waitResult.Connected {
		return fmt.Errorf("login failed: %s", waitResult.Message)
	}

	accountID := NormalizeAccountID(waitResult.AccountID)
	account := &WeixinAccountData{
		Token:   waitResult.BotToken,
		BaseURL: waitResult.BaseURL,
		UserID:  waitResult.UserID,
	}

	if err := c.accountManager.SaveAccount(accountID, account); err != nil {
		return fmt.Errorf("save account failed: %w", err)
	}

	c.botToken = waitResult.BotToken
	c.accountID = accountID
	if waitResult.BaseURL != "" {
		c.baseURL = waitResult.BaseURL
	}

	fmt.Println("\n✅ 微信登录成功！")
	return nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
