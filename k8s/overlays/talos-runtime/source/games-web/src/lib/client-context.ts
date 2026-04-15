export interface ClientTelemetry {
  deviceType: string;
  platform: string;
  browser: string;
  region: string;
}

const UNKNOWN = "unknown";

const normalize = (value: string | null | undefined): string => {
  const normalized = value?.trim().toLowerCase();
  return normalized ? normalized.replace(/[^a-z0-9_-]+/g, "_") : UNKNOWN;
};

const detectDeviceType = (userAgent: string): string => {
  const width = window.innerWidth || window.screen?.width || 0;
  const isTablet =
    /ipad|tablet/.test(userAgent) ||
    (/android/.test(userAgent) && !/mobile/.test(userAgent)) ||
    (navigator.maxTouchPoints > 1 && /macintosh/.test(userAgent));

  if (isTablet) {
    return "tablet";
  }

  if (/mobile|iphone|ipod|android/.test(userAgent) || (width > 0 && width < 768)) {
    return "mobile";
  }

  return "desktop";
};

const detectPlatform = (userAgent: string): string => {
  if (/android/.test(userAgent)) {
    return "android";
  }
  if (/iphone|ipad|ipod/.test(userAgent)) {
    return "ios";
  }
  if (/windows/.test(userAgent)) {
    return "windows";
  }
  if (/macintosh|mac os x/.test(userAgent)) {
    return "macos";
  }
  if (/cros/.test(userAgent)) {
    return "chromeos";
  }
  if (/linux/.test(userAgent)) {
    return "linux";
  }
  return UNKNOWN;
};

const detectBrowser = (userAgent: string): string => {
  if (/edg\//.test(userAgent)) {
    return "edge";
  }
  if (/firefox\//.test(userAgent)) {
    return "firefox";
  }
  if (/safari\//.test(userAgent) && !/chrome\//.test(userAgent) && !/chromium\//.test(userAgent)) {
    return "safari";
  }
  if (/chrome\//.test(userAgent) || /chromium\//.test(userAgent)) {
    return "chrome";
  }
  return "other";
};

const detectRegion = (): string => {
  const locale = navigator.language;
  if (!locale) {
    return UNKNOWN;
  }

  try {
    const intlLocale = new Intl.Locale(locale);
    if (intlLocale.region) {
      return normalize(intlLocale.region);
    }
  } catch {
    // ignore and fall back to locale parsing below
  }

  const localeParts = locale.split(/[-_]/);
  return normalize(localeParts[1]);
};

export const getClientTelemetry = (): ClientTelemetry => {
  if (typeof window === "undefined" || typeof navigator === "undefined") {
    return {
      deviceType: UNKNOWN,
      platform: UNKNOWN,
      browser: UNKNOWN,
      region: UNKNOWN
    };
  }

  const userAgent = navigator.userAgent.toLowerCase();

  return {
    deviceType: normalize(detectDeviceType(userAgent)),
    platform: normalize(detectPlatform(userAgent)),
    browser: normalize(detectBrowser(userAgent)),
    region: detectRegion()
  };
};
