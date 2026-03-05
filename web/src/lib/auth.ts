export const TOKEN_STORAGE_KEY = 'zeroclaw_token';
export const USER_STORAGE_KEY = 'zeroclaw_user';

let inMemoryToken: string | null = null;
let inMemoryUser: any = null;

function readStorage(key: string): string | null {
  try {
    return sessionStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeStorage(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value);
  } catch {
    // sessionStorage may be unavailable in some browser privacy modes
  }
}

function removeStorage(key: string): void {
  try {
    sessionStorage.removeItem(key);
  } catch {
    // Ignore
  }
}

function clearLegacyLocalStorageToken(key: string): void {
  try {
    localStorage.removeItem(key);
  } catch {
    // Ignore
  }
}

export function getToken(): string | null {
  if (inMemoryToken && inMemoryToken.length > 0) {
    return inMemoryToken;
  }

  const sessionToken = readStorage(TOKEN_STORAGE_KEY);
  if (sessionToken && sessionToken.length > 0) {
    inMemoryToken = sessionToken;
    return sessionToken;
  }

  try {
    const legacy = localStorage.getItem(TOKEN_STORAGE_KEY);
    if (legacy && legacy.length > 0) {
      inMemoryToken = legacy;
      writeStorage(TOKEN_STORAGE_KEY, legacy);
      localStorage.removeItem(TOKEN_STORAGE_KEY);
      return legacy;
    }
  } catch {
    // Ignore
  }

  return null;
}

export function setToken(token: string): void {
  inMemoryToken = token;
  writeStorage(TOKEN_STORAGE_KEY, token);
  clearLegacyLocalStorageToken(TOKEN_STORAGE_KEY);
}

export function clearToken(): void {
  inMemoryToken = null;
  removeStorage(TOKEN_STORAGE_KEY);
  clearLegacyLocalStorageToken(TOKEN_STORAGE_KEY);
}

export function isAuthenticated(): boolean {
  const token = getToken();
  return token !== null && token.length > 0;
}

export function getUser(): any {
  if (inMemoryUser) {
    return inMemoryUser;
  }

  try {
    const userStr = sessionStorage.getItem(USER_STORAGE_KEY);
    if (userStr) {
      inMemoryUser = JSON.parse(userStr);
      return inMemoryUser;
    }
  } catch {
    // Ignore
  }

  return null;
}

export function setUser(user: any): void {
  inMemoryUser = user;
  try {
    sessionStorage.setItem(USER_STORAGE_KEY, JSON.stringify(user));
  } catch {
    // sessionStorage may be unavailable in some browser privacy modes
  }
}

export function clearUser(): void {
  inMemoryUser = null;
  try {
    sessionStorage.removeItem(USER_STORAGE_KEY);
  } catch {
    // Ignore
  }
}
