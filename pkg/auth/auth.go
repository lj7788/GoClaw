package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userManager   *UserManager
	tokens        map[string]*Token
	WechatClient *WechatClient
}

type Token struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	IsAdmin   bool      `json:"is_admin"`
}

func NewAuthService(userManager *UserManager, wechatAppID, wechatAppSecret, wechatCallbackURL string) *AuthService {
	authService := &AuthService{
		userManager: userManager,
		tokens:      make(map[string]*Token),
	}

	// 初始化微信客户端
	if wechatAppID != "" && wechatAppSecret != "" && wechatCallbackURL != "" {
		authService.WechatClient = NewWechatClient(wechatAppID, wechatAppSecret, wechatCallbackURL)
	}

	return authService
}

// 密码哈希
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// 验证密码
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// 生成随机token
func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// 管理员登录
func (s *AuthService) AdminLogin(username, password string) (*Token, error) {
	admin, err := s.userManager.GetAdminByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}
	if admin == nil {
		return nil, fmt.Errorf("admin not found")
	}

	if !CheckPasswordHash(password, admin.Password) {
		return nil, fmt.Errorf("invalid password")
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	authToken := &Token{
		UserID:    admin.ID,
		Username:  admin.Username,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsAdmin:   true,
	}

	s.tokens[token] = authToken

	return authToken, nil
}

// 验证管理员token
func (s *AuthService) ValidateAdminToken(token string) (*Token, error) {
	authToken, exists := s.tokens[token]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if time.Now().After(authToken.ExpiresAt) {
		delete(s.tokens, token)
		return nil, fmt.Errorf("token expired")
	}

	if !authToken.IsAdmin {
		return nil, fmt.Errorf("not an admin token")
	}

	return authToken, nil
}

// 用户登录（微信登录后验证）
func (s *AuthService) UserLogin(openid string) (*Token, error) {
	user, err := s.userManager.GetUserByOpenID(openid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 检查用户状态
	if user.Status != 1 {
		return nil, fmt.Errorf("user not approved")
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	authToken := &Token{
		UserID:    user.ID,
		Username:  user.Nickname,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsAdmin:   false,
	}

	s.tokens[token] = authToken

	return authToken, nil
}

// 验证用户token
func (s *AuthService) ValidateUserToken(token string) (*Token, error) {
	authToken, exists := s.tokens[token]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if time.Now().After(authToken.ExpiresAt) {
		delete(s.tokens, token)
		return nil, fmt.Errorf("token expired")
	}

	return authToken, nil
}

// 刷新token
func (s *AuthService) RefreshToken(token string) (*Token, error) {
	authToken, err := s.ValidateUserToken(token)
	if err != nil {
		// 尝试验证管理员token
		authToken, err = s.ValidateAdminToken(token)
		if err != nil {
			return nil, err
		}
	}

	// 生成新token
	newToken, err := generateToken()
	if err != nil {
		return nil, err
	}

	// 创建新token
	newAuthToken := &Token{
		UserID:    authToken.UserID,
		Username:  authToken.Username,
		Token:     newToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsAdmin:   authToken.IsAdmin,
	}

	// 删除旧token
	delete(s.tokens, token)
	// 添加新token
	s.tokens[newToken] = newAuthToken

	return newAuthToken, nil
}

// 注销token
func (s *AuthService) Logout(token string) error {
	_, exists := s.tokens[token]
	if !exists {
		return fmt.Errorf("token not found")
	}

	delete(s.tokens, token)
	return nil
}

// 获取token信息
func (s *AuthService) GetTokenInfo(token string) (*Token, error) {
	authToken, exists := s.tokens[token]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if time.Now().After(authToken.ExpiresAt) {
		delete(s.tokens, token)
		return nil, fmt.Errorf("token expired")
	}

	return authToken, nil
}

// 清理过期token
func (s *AuthService) CleanExpiredTokens() {
	now := time.Now()
	for token, authToken := range s.tokens {
		if now.After(authToken.ExpiresAt) {
			delete(s.tokens, token)
		}
	}
}