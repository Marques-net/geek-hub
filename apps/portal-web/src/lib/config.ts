const trimTrailingSlash = (value: string): string => value.replace(/\/+$/, "");

export const getBackendUrl = (): string => {
  const configured = import.meta.env.VITE_BACKEND_URL?.trim();

  if (configured) {
    return trimTrailingSlash(configured);
  }

  return trimTrailingSlash(window.location.origin);
};

export const getGoogleClientId = (): string => import.meta.env.VITE_GOOGLE_CLIENT_ID?.trim() ?? "";
