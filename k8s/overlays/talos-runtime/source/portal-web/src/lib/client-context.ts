export interface ClientTelemetry {
  deviceType: string;
  platform: string;
  platformVersion: string;
  browser: string;
  browserVersion: string;
  region: string;
  deviceModel: string;
  rawUserAgent: string;
}

const UNKNOWN = "unknown";

const normalize = (value: string | null | undefined): string => {
  const normalized = value?.trim().toLowerCase();
  return normalized ? normalized.replace(/[^a-z0-9_-]+/g, "_") : UNKNOWN;
};

const cleanText = (value: string | null | undefined): string => {
  const normalized = value?.trim();
  return normalized ? normalized : UNKNOWN;
};

const normalizeVersion = (value: string | null | undefined): string => {
  const normalized = value?.trim().replace(/_/g, ".");
  return normalized ? normalized : UNKNOWN;
};

const detectDeviceType = (userAgent: string, userAgentData?: NavigatorUAData): string => {
  if (userAgentData?.mobile) {
    return "mobile";
  }

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
  if (/samsungbrowser\//.test(userAgent)) {
    return "samsung_internet";
  }
  if (/firefox\//.test(userAgent)) {
    return "firefox";
  }
  if (/opr\//.test(userAgent)) {
    return "opera";
  }
  if (/safari\//.test(userAgent) && !/chrome\//.test(userAgent) && !/chromium\//.test(userAgent)) {
    return "safari";
  }
  if (/chrome\//.test(userAgent) || /chromium\//.test(userAgent)) {
    return "chrome";
  }
  return "other";
};

const detectBrowserVersion = (userAgent: string): string => {
  const versionPatterns: RegExp[] = [
    /edg\/([0-9._]+)/,
    /samsungbrowser\/([0-9._]+)/,
    /firefox\/([0-9._]+)/,
    /opr\/([0-9._]+)/,
    /version\/([0-9._]+)/,
    /chrome\/([0-9._]+)/,
    /chromium\/([0-9._]+)/
  ];

  for (const pattern of versionPatterns) {
    const match = userAgent.match(pattern);
    if (match?.[1]) {
      return normalizeVersion(match[1]);
    }
  }

  return UNKNOWN;
};

const detectPlatformVersion = (userAgent: string): string => {
  const versionPatterns: RegExp[] = [
    /android ([0-9._]+)/,
    /os ([0-9_]+) like mac os x/,
    /windows nt ([0-9._]+)/,
    /mac os x ([0-9_]+)/,
    /cros [^ ]+ ([0-9._]+)/
  ];

  for (const pattern of versionPatterns) {
    const match = userAgent.match(pattern);
    if (match?.[1]) {
      return normalizeVersion(match[1]);
    }
  }

  return UNKNOWN;
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

const matchBrowserVersionFromBrands = (
  brands: NavigatorUABrandVersion[] | undefined,
  browser: string
): string => {
  if (!brands?.length) {
    return UNKNOWN;
  }

  const browserMatchers: Record<string, RegExp[]> = {
    edge: [/microsoft edge/i],
    chrome: [/google chrome/i, /chromium/i],
    opera: [/opera/i],
    safari: [/safari/i],
    samsung_internet: [/samsung internet/i],
    firefox: [/firefox/i]
  };

  for (const matcher of browserMatchers[browser] ?? []) {
    const brand = brands.find((candidate) => matcher.test(candidate.brand));
    if (brand?.version) {
      return normalizeVersion(brand.version);
    }
  }

  return UNKNOWN;
};

export const getClientTelemetry = async (): Promise<ClientTelemetry> => {
  if (typeof window === "undefined" || typeof navigator === "undefined") {
    return {
      deviceType: UNKNOWN,
      platform: UNKNOWN,
      platformVersion: UNKNOWN,
      browser: UNKNOWN,
      browserVersion: UNKNOWN,
      region: UNKNOWN,
      deviceModel: UNKNOWN,
      rawUserAgent: UNKNOWN
    };
  }

  const rawUserAgent = navigator.userAgent || "";
  const userAgent = rawUserAgent.toLowerCase();
  const userAgentData = navigator.userAgentData;

  const telemetry: ClientTelemetry = {
    deviceType: normalize(detectDeviceType(userAgent, userAgentData)),
    platform: normalize(detectPlatform(userAgent)),
    platformVersion: detectPlatformVersion(userAgent),
    browser: normalize(detectBrowser(userAgent)),
    browserVersion: detectBrowserVersion(userAgent),
    region: detectRegion(),
    deviceModel: UNKNOWN,
    rawUserAgent: cleanText(rawUserAgent)
  };

  if (!userAgentData?.getHighEntropyValues) {
    return telemetry;
  }

  try {
    const highEntropyValues = await userAgentData.getHighEntropyValues([
      "model",
      "platformVersion",
      "fullVersionList"
    ]);

    const deviceModel = cleanText(highEntropyValues.model);
    if (deviceModel !== UNKNOWN) {
      telemetry.deviceModel = deviceModel;
    }

    const platformVersion = normalizeVersion(highEntropyValues.platformVersion);
    if (platformVersion !== UNKNOWN) {
      telemetry.platformVersion = platformVersion;
    }

    const browserVersion = matchBrowserVersionFromBrands(
      highEntropyValues.fullVersionList ?? userAgentData.brands,
      telemetry.browser
    );
    if (browserVersion !== UNKNOWN) {
      telemetry.browserVersion = browserVersion;
    }
  } catch {
    // Browser blocked high-entropy client hints or does not support them.
  }

  return telemetry;
};
