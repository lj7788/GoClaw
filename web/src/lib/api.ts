import type {
  StatusResponse,
  ToolSpec,
  CronJob,
  Integration,
  DiagResult,
  MemoryEntry,
  CostSummary,
  CliTool,
  HealthSnapshot,
} from '../types/api';
import { clearToken, getToken, setToken } from './auth';

export class UnauthorizedError extends Error {
  constructor() {
    super('Unauthorized');
    this.name = 'UnauthorizedError';
  }
}

export async function apiFetch<T = unknown>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const token = getToken();
  const headers = new Headers(options.headers);

  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  if (
    options.body &&
    typeof options.body === 'string' &&
    !headers.has('Content-Type')
  ) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(path, { ...options, headers });

  if (response.status === 401) {
    clearToken();
    window.dispatchEvent(new Event('zeroclaw-unauthorized'));
    throw new UnauthorizedError();
  }

  if (!response.ok) {
    const text = await response.text().catch(() => '');
    throw new Error(`API ${response.status}: ${text || response.statusText}`);
  }

  if (response.status === 204) {
    return undefined as unknown as T;
  }

  return response.json() as Promise<T>;
}

function unwrapField<T>(value: T | Record<string, T>, key: string): T {
  if (value !== null && typeof value === 'object' && !Array.isArray(value) && key in value) {
    const unwrapped = (value as Record<string, T | undefined>)[key];
    if (unwrapped !== undefined) {
      return unwrapped;
    }
  }
  return value as T;
}

export async function pair(code: string): Promise<{ token: string }> {
  const response = await fetch('/api/pair', {
    method: 'POST',
    headers: { 'X-Pairing-Code': code },
  });

  if (!response.ok) {
    const text = await response.text().catch(() => '');
    throw new Error(`Pairing failed (${response.status}): ${text || response.statusText}`);
  }

  const data = (await response.json()) as { token: string };
  setToken(data.token);
  return data;
}

export async function getPublicHealth(): Promise<{ require_pairing: boolean; paired: boolean }> {
  const response = await fetch('/api/health');
  if (!response.ok) {
    throw new Error(`Health check failed (${response.status})`);
  }
  return response.json() as Promise<{ require_pairing: boolean; paired: boolean }>;
}

export function getStatus(): Promise<StatusResponse> {
  return apiFetch<StatusResponse>('/api/status');
}

export function getHealth(): Promise<HealthSnapshot> {
  return apiFetch<HealthSnapshot | { health: HealthSnapshot }>('/api/health').then((data) =>
    unwrapField(data, 'health'),
  );
}

export function getConfig(): Promise<string> {
  return apiFetch<string | { format?: string; content: string }>('/api/config').then((data) =>
    typeof data === 'string' ? data : data.content,
  );
}

export function putConfig(toml: string): Promise<void> {
  return apiFetch<void>('/api/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/toml' },
    body: toml,
  });
}

export function getTools(): Promise<ToolSpec[]> {
  return apiFetch<ToolSpec[] | { tools: ToolSpec[] }>('/api/tools').then((data) =>
    unwrapField(data, 'tools'),
  );
}

export function getCronJobs(): Promise<CronJob[]> {
  return apiFetch<CronJob[] | { jobs: CronJob[] }>('/api/cron').then((data) =>
    unwrapField(data, 'jobs'),
  );
}

export function addCronJob(body: {
  name?: string;
  command: string;
  schedule: string;
  enabled?: boolean;
}): Promise<CronJob> {
  return apiFetch<CronJob | { status: string; job: CronJob }>('/api/cron', {
    method: 'POST',
    body: JSON.stringify(body),
  }).then((data) => (typeof (data as { job?: CronJob }).job === 'object' ? (data as { job: CronJob }).job : (data as CronJob)));
}

export function deleteCronJob(id: string): Promise<void> {
  return apiFetch<void>(`/api/cron/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export function getIntegrations(): Promise<Integration[]> {
  return apiFetch<Integration[] | { integrations: Integration[] }>('/api/integrations').then(
    (data) => unwrapField(data, 'integrations'),
  );
}

export function runDoctor(): Promise<DiagResult[]> {
  return apiFetch<DiagResult[] | { results: DiagResult[]; summary?: unknown }>('/api/doctor', {
    method: 'POST',
    body: JSON.stringify({}),
  }).then((data) => (Array.isArray(data) ? data : data.results));
}

export function getMemory(
  query?: string,
  category?: string,
): Promise<MemoryEntry[]> {
  const params = new URLSearchParams();
  if (query) params.set('query', query);
  if (category) params.set('category', category);
  const qs = params.toString();
  return apiFetch<MemoryEntry[] | { entries: MemoryEntry[] }>(`/api/memory${qs ? `?${qs}` : ''}`).then(
    (data) => {
      data.entries = data.entries || [];
      return unwrapField(data, 'entries') || [];
    },
  );
}

export function storeMemory(
  key: string,
  content: string,
  category?: string,
): Promise<void> {
  return apiFetch<unknown>('/api/memory', {
    method: 'POST',
    body: JSON.stringify({ key, content, category }),
  }).then(() => undefined);
}

export function deleteMemory(key: string): Promise<{ status: string; deleted: boolean }> {
  return apiFetch<{ status: string; deleted: boolean }>(`/api/memory/${encodeURIComponent(key)}`, {
    method: 'DELETE',
  });
}

export function getCost(): Promise<CostSummary> {
  return apiFetch<CostSummary | { cost: CostSummary }>('/api/cost').then((data) =>
    unwrapField(data, 'cost'),
  );
}

export function getCliTools(): Promise<CliTool[]> {
  return apiFetch<CliTool[] | { cli_tools: CliTool[] }>('/api/cli-tools').then((data) =>
    unwrapField(data, 'cli_tools'),
  );
}

// WeChat login APIs
export async function getWechatLoginURL(): Promise<{ login_url: string; state: string }> {
  return apiFetch<{ login_url: string; state: string }>('/api/auth/wechat/login');
}

export async function getWechatUserInfo(): Promise<{ status: string; user: any }> {
  return apiFetch<{ status: string; user: any }>('/api/auth/wechat/user');
}

// User APIs
export async function getUserInfo(): Promise<{ status: string; user: any }> {
  return apiFetch<{ status: string; user: any }>('/api/auth/user/info');
}

export async function updateUserInfo(data: { email?: string; avatar?: string }): Promise<{ status: string; user: any }> {
  return apiFetch<{ status: string; user: any }>('/api/auth/user/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// Admin APIs
export async function adminLogin(data: { username: string; password: string }): Promise<{ status: string; token: string; admin: any }> {
  return apiFetch<{ status: string; token: string; admin: any }>('/api/auth/admin/login', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function getAdminUsers(): Promise<{ status: string; users: any[] }> {
  return apiFetch<{ status: string; users: any[] }>('/api/auth/admin/users');
}

export async function approveUser(data: { user_id: number; status: number }): Promise<{ status: string; user: any }> {
  return apiFetch<{ status: string; user: any }>('/api/auth/admin/users/approve', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function changeAdminPassword(data: { old_password: string; new_password: string }): Promise<{ status: string; message: string }> {
  return apiFetch<{ status: string; message: string }>('/api/auth/admin/password', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}
