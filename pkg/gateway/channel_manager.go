package gateway

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/zeroclaw-labs/goclaw/pkg/channels"
	"github.com/zeroclaw-labs/goclaw/pkg/config"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

// ChannelStatus represents the status of a channel
type ChannelStatus struct {
	Name        string `json:"name"`
	Connected   bool   `json:"connected"`
	AccountID   string `json:"account_id,omitempty"`
	Message     string `json:"message,omitempty"`
}

// WeixinLoginSession represents an active WeChat login session
type WeixinLoginSession struct {
	SessionKey string `json:"session_key"`
	QRCodeURL  string `json:"qrcode_url"`
	QRCode     string `json:"-"` // Internal use, not exposed to frontend
	Status     string `json:"status"` // "waiting", "scanned", "confirmed", "expired"
}

// ChannelManager manages channel connections
type ChannelManager struct {
	mu sync.RWMutex

	config          *config.Config
	dingtalkChannel *channels.DingTalkChannel
	weixinChannel   *channels.WeixinChannel

	// WeChat login sessions
	weixinSessions map[string]*WeixinLoginSession

	// Message channel for agent integration
	msgChan chan types.ChannelMessage
}

// NewChannelManager creates a new channel manager
func NewChannelManager(cfg *config.Config) *ChannelManager {
	return &ChannelManager{
		config:        cfg,
		weixinSessions: make(map[string]*WeixinLoginSession),
		msgChan:       make(chan types.ChannelMessage, 100),
	}
}

// GetStatus returns the status of all channels
func (m *ChannelManager) GetStatus() map[string]ChannelStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]ChannelStatus)

	// DingTalk status
	dtStatus := ChannelStatus{Name: "dingtalk"}
	if m.dingtalkChannel != nil {
		dtStatus.Connected = true
		dtStatus.Message = "已连接"
	} else {
		// Check if configured
		if dtCfg := m.config.GetDingTalkConfig(); dtCfg != nil && dtCfg.ClientID != "" {
			dtStatus.Message = "已配置，未连接"
		} else {
			dtStatus.Message = "未配置"
		}
	}
	status["dingtalk"] = dtStatus

	// Weixin status
	wxStatus := ChannelStatus{Name: "weixin"}
	if m.weixinChannel != nil && m.weixinChannel.IsConfigured() {
		wxStatus.Connected = true
		wxStatus.AccountID = m.weixinChannel.GetAccountID()
		wxStatus.Message = "已绑定"
	} else {
		// Check if there's saved account
		if weixinCfg := m.config.GetWeixinConfig(); weixinCfg != nil && weixinCfg.Token != "" {
			wxStatus.Message = "有保存的账号，未连接"
		} else {
			wxStatus.Message = "未绑定"
		}
	}
	status["weixin"] = wxStatus

	return status
}

// ConfigureDingTalk configures DingTalk channel
func (m *ChannelManager) ConfigureDingTalk(clientID, clientSecret string, allowedUsers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Disconnect existing channel first
	if m.dingtalkChannel != nil {
		m.dingtalkChannel = nil
	}

	// Save config
	if err := m.config.SaveDingTalkConfig(clientID, clientSecret, allowedUsers); err != nil {
		return err
	}

	return nil
}

// ConnectDingTalk starts DingTalk connection
func (m *ChannelManager) ConnectDingTalk(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dtCfg := m.config.GetDingTalkConfig()
	if dtCfg == nil || dtCfg.ClientID == "" {
		return ErrDingTalkNotConfigured
	}

	// Create new channel
	m.dingtalkChannel = channels.NewDingTalkChannel(
		dtCfg.ClientID,
		dtCfg.ClientSecret,
		dtCfg.AllowedUsers,
	)

	// Start listening in background
	go func() {
		if err := m.dingtalkChannel.Listen(ctx, m.msgChan); err != nil {
			log.Printf("DingTalk listen error: %v", err)
			m.mu.Lock()
			m.dingtalkChannel = nil
			m.mu.Unlock()
		}
	}()

	return nil
}

// DisconnectDingTalk stops DingTalk connection
func (m *ChannelManager) DisconnectDingTalk() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dingtalkChannel != nil {
		// DingTalk channel doesn't have explicit Stop, just clear reference
		m.dingtalkChannel = nil
	}
}

// StartWeixinLogin starts WeChat QR code login flow
func (m *ChannelManager) StartWeixinLogin(ctx context.Context) (*WeixinLoginSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a temporary weixin channel for login
	weixinCfg := m.config.GetWeixinConfig()
	if weixinCfg == nil {
		weixinCfg = &config.WeixinConfig{}
	}

	tempChannel := channels.NewWeixinChannel(&channels.WeixinConfig{
		Token:      weixinCfg.Token,
		BaseURL:    weixinCfg.BaseURL,
		CDNBaseURL: weixinCfg.CDNBaseURL,
		AccountID:  weixinCfg.AccountID,
	})

	// Start QR login
	result, err := tempChannel.StartQRLogin(ctx)
	if err != nil {
		return nil, err
	}

	if result.QRCodeURL == "" {
		return nil, &ChannelError{Message: "获取二维码失败"}
	}

	sessionKey := result.SessionKey

	session := &WeixinLoginSession{
		SessionKey: sessionKey,
		QRCodeURL:  result.QRCodeURL,
		QRCode:     result.QRCode,
		Status:     "waiting",
	}
	m.weixinSessions[sessionKey] = session
	log.Printf("[Weixin] Session stored with key: %s, ptr: %p", sessionKey, session)

	// Start waiting for login in background using direct polling
	// Use context.Background() to avoid cancellation when the HTTP request ends
	go func(session *WeixinLoginSession, sessionKey string, qrCode string) {
		ctx := context.Background()
		deadline := time.Now().Add(5 * time.Minute)
		log.Printf("[Weixin] Starting background polling, QRCode: %s, session ptr: %p", qrCode, session)
		for time.Now().Before(deadline) {
			statusResp, err := tempChannel.PollQRStatus(ctx, qrCode)
			if err != nil {
				log.Printf("[Weixin] Poll error: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Printf("[Weixin] Poll status: %s", statusResp.Status)

			switch statusResp.Status {
			case "wait":
				time.Sleep(2 * time.Second)
			case "scaned":
				log.Printf("[Weixin] QR code scanned, waiting for confirmation...")
				m.mu.Lock()
				session.Status = "scanned"
				m.mu.Unlock()
				time.Sleep(1 * time.Second)
			case "expired":
				log.Printf("[Weixin] QR code expired")
				m.mu.Lock()
				session.Status = "expired"
				m.mu.Unlock()
				return
			case "confirmed":
				log.Printf("[Weixin] Login confirmed! ILinkBotID: %s, BotToken: %s, BaseURL: %s", statusResp.ILinkBotID, statusResp.BotToken, statusResp.BaseURL)
				// Login successful - save account
				accountID := channels.NormalizeAccountID(statusResp.ILinkBotID)
				log.Printf("[Weixin] Saving account with ID: %s...", accountID)
				saveErr := tempChannel.SaveAccountFromLogin(accountID, statusResp.BotToken, statusResp.BaseURL, statusResp.ILinkUserID)
				log.Printf("[Weixin] SaveAccountFromLogin returned: %v", saveErr)
				if saveErr != nil {
					log.Printf("[Weixin] Failed to save account: %v", saveErr)
					m.mu.Lock()
					session.Status = "expired"
					m.mu.Unlock()
					return
				}
				log.Printf("[Weixin] Account saved successfully")

				// Set the botToken on the channel so Listen can work
				tempChannel.SetBotToken(statusResp.BotToken, accountID)

				m.mu.Lock()
				session.Status = "confirmed"
				log.Printf("[Weixin] Set session.Status = confirmed, ptr: %p", session)
				m.weixinChannel = tempChannel
				m.weixinSessions[sessionKey] = session
				m.mu.Unlock()

				log.Printf("[Weixin] Session status updated to confirmed, ptr: %p, status: %s", session, session.Status)

				// Start listening if msgChan is set (shared with daemon)
				if m.msgChan != nil {
					go func() {
						log.Printf("[Weixin] Starting Listen for account: %s", accountID)
						if err := m.weixinChannel.Listen(ctx, m.msgChan); err != nil {
							log.Printf("Weixin listen error: %v", err)
						}
					}()
				}
				return
			default:
				log.Printf("[Weixin] Unknown status: %s", statusResp.Status)
				time.Sleep(2 * time.Second)
			}
		}

		// Timeout
		log.Printf("[Weixin] Login timeout")
		m.mu.Lock()
		session.Status = "expired"
		m.mu.Unlock()
	}(session, sessionKey, result.QRCode)

	return session, nil
}

// GetWeixinLoginStatus returns the status of a WeChat login session
func (m *ChannelManager) GetWeixinLoginStatus(sessionKey string) *WeixinLoginSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.weixinSessions[sessionKey]
}

// DisconnectWeixin disconnects WeChat
func (m *ChannelManager) DisconnectWeixin() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.weixinChannel != nil {
		m.weixinChannel.Stop()
		// Delete saved account
		accountID := m.weixinChannel.GetAccountID()
		if accountID != "" {
			if err := m.weixinChannel.DeleteAccount(accountID); err != nil {
				log.Printf("[Weixin] Failed to delete account: %v", err)
			}
		}
		m.weixinChannel = nil
	}

	// Clear all sessions
	m.weixinSessions = make(map[string]*WeixinLoginSession)
}

// GetMessageChannel returns the message channel for agent integration
func (m *ChannelManager) GetMessageChannel() <-chan types.ChannelMessage {
	return m.msgChan
}

// GetMessageSendChannel returns the message channel for sending (used by Listen)
func (m *ChannelManager) GetMessageSendChannel() chan<- types.ChannelMessage {
	return m.msgChan
}

// SetMessageChannel sets a shared message channel
func (m *ChannelManager) SetMessageChannel(ch chan types.ChannelMessage) {
	m.msgChan = ch
}

// SendToDingTalk sends a message through DingTalk
func (m *ChannelManager) SendToDingTalk(ctx context.Context, recipient, content string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.dingtalkChannel == nil {
		return ErrDingTalkNotConnected
	}

	return m.dingtalkChannel.Send(ctx, &types.SendMessage{
		Recipient: recipient,
		Content:   content,
	})
}

// SendToWeixin sends a message through WeChat
func (m *ChannelManager) SendToWeixin(ctx context.Context, recipient, content string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.weixinChannel == nil {
		return ErrWeixinNotConnected
	}

	return m.weixinChannel.Send(ctx, &types.SendMessage{
		Recipient: recipient,
		Content:   content,
	})
}

// IsDingTalkConnected returns true if DingTalk is connected
func (m *ChannelManager) IsDingTalkConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dingtalkChannel != nil
}

// IsWeixinConnected returns true if WeChat is connected
func (m *ChannelManager) IsWeixinConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.weixinChannel != nil && m.weixinChannel.IsConfigured()
}

// GetWeixinChannel returns the WeChat channel (may be nil if not connected)
func (m *ChannelManager) GetWeixinChannel() *channels.WeixinChannel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.weixinChannel
}

// BroadcastStatus sends channel status to WebSocket clients
func (m *ChannelManager) BroadcastStatus(broadcast func(msg []byte)) {
	status := m.GetStatus()
	data, err := json.Marshal(map[string]interface{}{
		"type":     "channel_status",
		"channels": status,
	})
	if err == nil {
		broadcast(data)
	}
}

// Error definitions
var (
	ErrDingTalkNotConfigured = &ChannelError{Message: "钉钉未配置"}
	ErrDingTalkNotConnected  = &ChannelError{Message: "钉钉未连接"}
	ErrWeixinNotConnected    = &ChannelError{Message: "微信未连接"}
)

// ChannelError represents a channel error
type ChannelError struct {
	Message string
}

func (e *ChannelError) Error() string {
	return e.Message
}