package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type WechatClient struct {
	AppID     string
	AppSecret string
	RedirectURI string
}

type WechatTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID      string `json:"openid"`
	Scope       string `json:"scope"`
	UnionID     string `json:"unionid,omitempty"`
	ErrCode     int    `json:"errcode,omitempty"`
	ErrMsg      string `json:"errmsg,omitempty"`
}

type WechatUserInfoResponse struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid,omitempty"`
	ErrCode    int    `json:"errcode,omitempty"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

func NewWechatClient(appID, appSecret, redirectURI string) *WechatClient {
	return &WechatClient{
		AppID:     appID,
		AppSecret: appSecret,
		RedirectURI: redirectURI,
	}
}

// GetAuthURL 生成微信授权URL（使用公众号网页授权）
func (c *WechatClient) GetAuthURL(state string) string {
	params := url.Values{}
	params.Add("appid", c.AppID)
	params.Add("redirect_uri", c.RedirectURI)
	params.Add("response_type", "code")
	params.Add("scope", "snsapi_userinfo")
	params.Add("state", state)
	return "https://open.weixin.qq.com/connect/oauth2/authorize?" + params.Encode() + "#wechat_redirect"
}

// GetAccessToken 通过code获取access_token
func (c *WechatClient) GetAccessToken(code string) (*WechatTokenResponse, error) {
	params := url.Values{}
	params.Add("appid", c.AppID)
	params.Add("secret", c.AppSecret)
	params.Add("code", code)
	params.Add("grant_type", "authorization_code")

	url := "https://api.weixin.qq.com/sns/oauth2/access_token?" + params.Encode()
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp WechatTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d - %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	return &tokenResp, nil
}

// RefreshAccessToken 刷新access_token
func (c *WechatClient) RefreshAccessToken(refreshToken string) (*WechatTokenResponse, error) {
	params := url.Values{}
	params.Add("appid", c.AppID)
	params.Add("grant_type", "refresh_token")
	params.Add("refresh_token", refreshToken)

	url := "https://api.weixin.qq.com/sns/oauth2/refresh_token?" + params.Encode()
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp WechatTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d - %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	return &tokenResp, nil
}

// GetUserInfo 获取用户信息
func (c *WechatClient) GetUserInfo(accessToken, openID string) (*WechatUserInfoResponse, error) {
	params := url.Values{}
	params.Add("access_token", accessToken)
	params.Add("openid", openID)
	params.Add("lang", "zh_CN")

	url := "https://api.weixin.qq.com/sns/userinfo?" + params.Encode()
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var userInfoResp WechatUserInfoResponse
	if err := json.Unmarshal(body, &userInfoResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if userInfoResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d - %s", userInfoResp.ErrCode, userInfoResp.ErrMsg)
	}

	return &userInfoResp, nil
}

// ValidateAccessToken 验证access_token是否有效
func (c *WechatClient) ValidateAccessToken(accessToken, openID string) (bool, error) {
	params := url.Values{}
	params.Add("access_token", accessToken)
	params.Add("openid", openID)

	url := "https://api.weixin.qq.com/sns/auth?" + params.Encode()
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to validate access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	var validateResp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &validateResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	return validateResp.ErrCode == 0, nil
}