import { apiFetch } from './api'

// Types
export interface ChannelStatus {
  name: string
  connected: boolean
  account_id?: string
  message?: string
}

export interface ChannelsStatusResponse {
  channels: Record<string, ChannelStatus>
}

export interface DingTalkConfigRequest {
  client_id: string
  client_secret: string
  allowed_users?: string[]
}

export interface WeixinQRCodeResponse {
  status: string
  qrcode_url: string
  session_key: string
}

export interface WeixinStatusResponse {
  status: boolean | string  // boolean for overall status, string for session status ("waiting", "confirmed", "expired")
  account_id?: string
  message?: string
}

// Get all channels status
export async function getChannelsStatus(): Promise<ChannelsStatusResponse> {
  return apiFetch<ChannelsStatusResponse>('/api/channels')
}

// DingTalk APIs
export async function saveDingTalkConfig(config: DingTalkConfigRequest): Promise<{ status: string; message: string }> {
  return apiFetch<{ status: string; message: string }>('/api/channels/dingtalk/config', {
    method: 'POST',
    body: JSON.stringify(config),
  })
}

export async function connectDingTalk(): Promise<{ status: string; message: string }> {
  return apiFetch<{ status: string; message: string }>('/api/channels/dingtalk/connect', {
    method: 'POST',
  })
}

export async function disconnectDingTalk(): Promise<{ status: string; message: string }> {
  return apiFetch<{ status: string; message: string }>('/api/channels/dingtalk/disconnect', {
    method: 'POST',
  })
}

// WeChat APIs
export async function getWeixinQRCode(): Promise<WeixinQRCodeResponse> {
  console.log('[getWeixinQRCode] Calling /api/channels/weixin/qrcode')
  try {
    const result = await apiFetch<WeixinQRCodeResponse>('/api/channels/weixin/qrcode')
    console.log('[getWeixinQRCode] Result:', result)
    return result
  } catch (err) {
    console.error('[getWeixinQRCode] Error:', err)
    throw err
  }
}

export async function getWeixinStatus(sessionKey?: string): Promise<WeixinStatusResponse> {
  const url = sessionKey
    ? `/api/channels/weixin/status?session_key=${encodeURIComponent(sessionKey)}`
    : '/api/channels/weixin/status'
  return apiFetch<WeixinStatusResponse>(url)
}

export async function disconnectWeixin(): Promise<{ status: string; message: string }> {
  return apiFetch<{ status: string; message: string }>('/api/channels/weixin/disconnect', {
    method: 'POST',
  })
}