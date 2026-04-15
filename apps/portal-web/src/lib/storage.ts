import { AuthSession, ClientSession } from "../types";

const STORAGE_KEY = "geek-hub-room-session";
const AUTH_STORAGE_KEY = "geek-hub-auth";

export const readSession = (): ClientSession | null => {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as ClientSession) : null;
  } catch {
    return null;
  }
};

export const writeSession = (session: ClientSession): void => {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
};

export const clearSession = (): void => {
  window.localStorage.removeItem(STORAGE_KEY);
};

export const readAuthSession = (): AuthSession | null => {
  try {
    const raw = window.localStorage.getItem(AUTH_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as AuthSession) : null;
  } catch {
    return null;
  }
};

export const writeAuthSession = (session: AuthSession): void => {
  window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session));
};

export const clearAuthSession = (): void => {
  window.localStorage.removeItem(AUTH_STORAGE_KEY);
};
