const AUTH_TOKEN_KEY = "token";

// getAuthToken 输入为空，输出为当前登录 token，目的在于集中管理后台鉴权凭证读取。
export function getAuthToken() {
  return window.localStorage.getItem(AUTH_TOKEN_KEY) || "";
}

// setAuthToken 输入 token 字符串，输出为空，目的在于在登录成功后保存当前会话凭证。
export function setAuthToken(tokenValue) {
  window.localStorage.setItem(AUTH_TOKEN_KEY, tokenValue);
}

// clearAuthToken 输入为空，输出为空，目的在于在退出登录或 token 失效时清理本地凭证。
export function clearAuthToken() {
  window.localStorage.removeItem(AUTH_TOKEN_KEY);
}

// breakOutOfIframe 输入为空，输出为顶层页面跳转结果，目的在于避免后台页被嵌入旧 iframe 壳中。
export function breakOutOfIframe() {
  if (window.top !== window.self) {
    window.top.location.href = window.location.href;
  }
}

// buildVisitorCacheKey 输入会话入口标识，输出为访客缓存键，目的在于在同一入口链路下复用访客身份。
export function buildVisitorCacheKey(entryKey) {
  return `visitor_${entryKey || "default"}`;
}

// loadVisitorCache 输入缓存键，输出为访客缓存对象，目的在于尽量复用已分配的 visitor_id。
export function loadVisitorCache(cacheKey) {
  try {
    const rawValue = window.localStorage.getItem(cacheKey);
    return rawValue ? JSON.parse(rawValue) : null;
  } catch (error) {
    return null;
  }
}

// saveVisitorCache 输入缓存键和访客对象，输出为空，目的在于持久化访客身份信息以减少重复建档。
export function saveVisitorCache(cacheKey, visitorInfo) {
  window.localStorage.setItem(cacheKey, JSON.stringify(visitorInfo));
}
