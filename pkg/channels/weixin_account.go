package channels

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type WeixinAccountManager struct {
	mu          sync.RWMutex
	stateDir    string
	accountsDir string
	indexFile   string
}

func NewWeixinAccountManager(stateDir string) *WeixinAccountManager {
	if stateDir == "" {
		homeDir, _ := os.UserHomeDir()
		stateDir = filepath.Join(homeDir, ".goclaw")
	}
	
	weixinDir := filepath.Join(stateDir, "weixin")
	accountsDir := filepath.Join(weixinDir, "accounts")
	
	return &WeixinAccountManager{
		stateDir:    stateDir,
		accountsDir: accountsDir,
		indexFile:   filepath.Join(weixinDir, "accounts.json"),
	}
}

func (m *WeixinAccountManager) ListAccountIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, err := os.ReadFile(m.indexFile)
	if err == nil {
		var ids []string
		if err := json.Unmarshal(data, &ids); err == nil && len(ids) > 0 {
			return ids
		}
	}
	
	if err := os.MkdirAll(m.accountsDir, 0755); err != nil {
		return nil
	}
	
	entries, err := os.ReadDir(m.accountsDir)
	if err != nil {
		return nil
	}
	
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			id := strings.TrimSuffix(entry.Name(), ".json")
			ids = append(ids, id)
		}
	}
	
	return ids
}

func (m *WeixinAccountManager) RegisterAccountID(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registerAccountIDUnlocked(accountID)
}

func (m *WeixinAccountManager) registerAccountIDUnlocked(accountID string) error {
	ids := m.listAccountIDsUnlocked()
	for _, id := range ids {
		if id == accountID {
			return nil
		}
	}

	ids = append(ids, accountID)
	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(m.indexFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(m.indexFile, data, 0644)
}

func (m *WeixinAccountManager) listAccountIDsUnlocked() []string {
	data, err := os.ReadFile(m.indexFile)
	if err != nil {
		return nil
	}
	
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil
	}
	
	return ids
}

func (m *WeixinAccountManager) LoadAccount(accountID string) (*WeixinAccountData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	filePath := m.accountFilePath(accountID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			rawID := DeriveRawAccountID(accountID)
			if rawID != "" {
				rawPath := m.accountFilePath(rawID)
				rawData, rawErr := os.ReadFile(rawPath)
				if rawErr == nil {
					var account WeixinAccountData
					if err := json.Unmarshal(rawData, &account); err == nil {
						return &account, nil
					}
				}
			}
		}
		return nil, err
	}
	
	var account WeixinAccountData
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}
	
	return &account, nil
}

func (m *WeixinAccountManager) SaveAccount(accountID string, account *WeixinAccountData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if err := os.MkdirAll(m.accountsDir, 0755); err != nil {
		return err
	}
	
	account.SavedAt = time.Now().Format(time.RFC3339)
	
	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}
	
	filePath := m.accountFilePath(accountID)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return err
	}

	return m.registerAccountIDUnlocked(accountID)
}

// DeleteAccount deletes an account file and removes it from the index
func (m *WeixinAccountManager) DeleteAccount(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Delete the account file
	filePath := m.accountFilePath(accountID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove from index
	ids := m.listAccountIDsUnlocked()
	newIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != accountID {
			newIDs = append(newIDs, id)
		}
	}

	if len(newIDs) == 0 {
		// No accounts left, delete the index file
		os.Remove(m.indexFile)
		return nil
	}

	data, err := json.MarshalIndent(newIDs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.indexFile, data, 0644)
}

func (m *WeixinAccountManager) accountFilePath(accountID string) string {
	return filepath.Join(m.accountsDir, accountID+".json")
}

func DeriveRawAccountID(normalizedID string) string {
	if strings.HasSuffix(normalizedID, "-im-bot") {
		return normalizedID[:len(normalizedID)-7] + "@im.bot"
	}
	if strings.HasSuffix(normalizedID, "-im-wechat") {
		return normalizedID[:len(normalizedID)-10] + "@im.wechat"
	}
	return ""
}

func NormalizeAccountID(rawID string) string {
	rawID = strings.TrimSpace(rawID)
	rawID = strings.ReplaceAll(rawID, "@im.bot", "-im-bot")
	rawID = strings.ReplaceAll(rawID, "@im.wechat", "-im-wechat")
	return rawID
}

type QRLoginManager struct {
	mu          sync.RWMutex
	activeLogins map[string]*ActiveLogin
	httpClient  *WeixinChannel
}

type ActiveLogin struct {
	SessionKey string
	ID         string
	QRCode     string
	QRCodeURL  string
	StartedAt  int64
	BotToken   string
	Status     string
	Error      string
}

const ActiveLoginTTLMs = 5 * 60 * 1000
const QRLongPollTimeoutMs = 35000

func NewQRLoginManager(httpClient *WeixinChannel) *QRLoginManager {
	return &QRLoginManager{
		activeLogins: make(map[string]*ActiveLogin),
		httpClient:   httpClient,
	}
}

type QRStartResult struct {
	QRCodeURL  string `json:"qrcodeUrl,omitempty"`
	QRCode     string `json:"qrcode,omitempty"`     // The raw QR code string for polling status
	Message    string `json:"message"`
	SessionKey string `json:"sessionKey"`
}

type QRWaitResult struct {
	Connected bool   `json:"connected"`
	BotToken  string `json:"botToken,omitempty"`
	AccountID string `json:"accountId,omitempty"`
	BaseURL   string `json:"baseUrl,omitempty"`
	UserID    string `json:"userId,omitempty"`
	Message   string `json:"message"`
}

func (m *QRLoginManager) StartLogin(ctx context.Context, accountID, apiBaseURL string, force bool) (*QRStartResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.purgeExpiredLogins()
	
	sessionKey := accountID
	if sessionKey == "" {
		sessionKey = generateSessionKey()
	}
	
	if !force {
		if existing, ok := m.activeLogins[sessionKey]; ok {
			if isLoginFresh(existing) && existing.QRCodeURL != "" {
				return &QRStartResult{
					QRCodeURL:  existing.QRCodeURL,
					Message:    "二维码已就绪，请使用微信扫描。",
					SessionKey: sessionKey,
				}, nil
			}
		}
	}
	
	qrResp, err := m.fetchQRCode(ctx, apiBaseURL)
	if err != nil {
		return &QRStartResult{
			Message:    fmt.Sprintf("获取二维码失败: %v", err),
			SessionKey: sessionKey,
		}, err
	}
	
	login := &ActiveLogin{
		SessionKey: sessionKey,
		ID:         generateSessionKey(),
		QRCode:     qrResp.QRCode,
		QRCodeURL:  qrResp.QRCodeImgContent,
		StartedAt:  time.Now().UnixMilli(),
	}

	m.activeLogins[sessionKey] = login

	// Use QRCodeImgContent if available, otherwise construct URL from QRCode
	qrcodeURL := qrResp.QRCodeImgContent
	if qrcodeURL == "" && qrResp.QRCode != "" {
		// Construct a URL that can be used to display the QR code
		qrcodeURL = "https://api.qrserver.com/v1/create-qr-code/?size=256x256&data=" + qrResp.QRCode
	}

	return &QRStartResult{
		QRCodeURL:  qrcodeURL,
		QRCode:     qrResp.QRCode,
		Message:    "使用微信扫描以下二维码，以完成连接。",
		SessionKey: sessionKey,
	}, nil
}

func (m *QRLoginManager) WaitForLogin(ctx context.Context, sessionKey, apiBaseURL string, timeoutMs int) (*QRWaitResult, error) {
	m.mu.Lock()
	activeLogin, ok := m.activeLogins[sessionKey]
	if !ok {
		m.mu.Unlock()
		return &QRWaitResult{
			Connected: false,
			Message:   "当前没有进行中的登录，请先发起登录。",
		}, nil
	}
	
	if !isLoginFresh(activeLogin) {
		delete(m.activeLogins, sessionKey)
		m.mu.Unlock()
		return &QRWaitResult{
			Connected: false,
			Message:   "二维码已过期，请重新生成。",
		}, nil
	}
	m.mu.Unlock()
	
	if timeoutMs <= 0 {
		timeoutMs = 480000
	}
	
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	qrRefreshCount := 1
	maxQRRefreshCount := 3
	
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		statusResp, err := m.PollQRStatus(ctx, apiBaseURL, activeLogin.QRCode)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		
		m.mu.Lock()
		if login, ok := m.activeLogins[sessionKey]; ok {
			login.Status = statusResp.Status
		}
		m.mu.Unlock()
		
		switch statusResp.Status {
		case "wait":
			time.Sleep(2 * time.Second)
			
		case "scaned":
			time.Sleep(1 * time.Second)
			
		case "expired":
			qrRefreshCount++
			if qrRefreshCount > maxQRRefreshCount {
				m.mu.Lock()
				delete(m.activeLogins, sessionKey)
				m.mu.Unlock()
				return &QRWaitResult{
					Connected: false,
					Message:   "登录超时：二维码多次过期，请重新开始登录流程。",
				}, nil
			}
			
			newQR, err := m.fetchQRCode(ctx, apiBaseURL)
			if err != nil {
				m.mu.Lock()
				delete(m.activeLogins, sessionKey)
				m.mu.Unlock()
				return &QRWaitResult{
					Connected: false,
					Message:   fmt.Sprintf("刷新二维码失败: %v", err),
				}, nil
			}
			
			m.mu.Lock()
			if login, ok := m.activeLogins[sessionKey]; ok {
				login.QRCode = newQR.QRCode
				login.QRCodeURL = newQR.QRCodeImgContent
				login.StartedAt = time.Now().UnixMilli()
				activeLogin = login
			}
			m.mu.Unlock()
			
		case "confirmed":
			m.mu.Lock()
			delete(m.activeLogins, sessionKey)
			m.mu.Unlock()
			
			if statusResp.ILinkBotID == "" {
				return &QRWaitResult{
					Connected: false,
					Message:   "登录失败：服务器未返回 ilink_bot_id。",
				}, nil
			}
			
			return &QRWaitResult{
				Connected: true,
				BotToken:  statusResp.BotToken,
				AccountID: statusResp.ILinkBotID,
				BaseURL:   statusResp.BaseURL,
				UserID:    statusResp.ILinkUserID,
				Message:   "✅ 与微信连接成功！",
			}, nil
		}
	}
	
	m.mu.Lock()
	delete(m.activeLogins, sessionKey)
	m.mu.Unlock()
	
	return &QRWaitResult{
		Connected: false,
		Message:   "登录超时，请重试。",
	}, nil
}

func (m *QRLoginManager) fetchQRCode(ctx context.Context, apiBaseURL string) (*QRCodeResponse, error) {
	base := apiBaseURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	url := base + "ilink/bot/get_bot_qrcode?bot_type=" + DefaultBotType
	log.Printf("[Weixin] Fetching QR code from: %s", url)

	req, err := createRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Weixin] Failed to fetch QR code: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[Weixin] QR code response status: %d, body length: %d", resp.StatusCode, len(body))

	var result QRCodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[Weixin] Failed to decode QR code response: %v", err)
		return nil, err
	}

	log.Printf("[Weixin] QRCode length: %d, QRCodeImgContent length: %d", len(result.QRCode), len(result.QRCodeImgContent))

	return &result, nil
}

func (m *QRLoginManager) PollQRStatus(ctx context.Context, apiBaseURL, qrCode string) (*QRStatusResponse, error) {
	base := apiBaseURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	
	url := base + "ilink/bot/get_qrcode_status?qrcode=" + qrCode
	
	req, err := createRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("iLink-App-ClientVersion", "1")
	
	client := &http.Client{Timeout: time.Duration(QRLongPollTimeoutMs) * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result QRStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

func (m *QRLoginManager) purgeExpiredLogins() {
	now := time.Now().UnixMilli()
	for key, login := range m.activeLogins {
		if now-login.StartedAt > ActiveLoginTTLMs {
			delete(m.activeLogins, key)
		}
	}
}

func isLoginFresh(login *ActiveLogin) bool {
	return time.Now().UnixMilli()-login.StartedAt < ActiveLoginTTLMs
}

func generateSessionKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func createRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}
