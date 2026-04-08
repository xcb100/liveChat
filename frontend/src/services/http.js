import { clearAuthToken, getAuthToken } from "./session";

// createTimeoutHandle 输入超时毫秒数，输出为包含 AbortController 的控制句柄，目的在于为 fetch 请求统一注入超时控制。
function createTimeoutHandle(timeoutMilliseconds) {
  const controller = new AbortController();
  const timeoutIdentifier = window.setTimeout(() => {
    controller.abort();
  }, timeoutMilliseconds);

  return {
    controller,
    timeoutIdentifier,
  };
}

// clearTimeoutHandle 输入超时控制句柄，输出为空，目的在于在请求结束后及时释放超时定时器。
function clearTimeoutHandle(timeoutHandle) {
  if (!timeoutHandle) {
    return;
  }

  window.clearTimeout(timeoutHandle.timeoutIdentifier);
}

// createFormData 输入普通对象，输出为 FormData，目的在于兼容当前后端大量 multipart/form-data 接口。
export function createFormData(values) {
  const formData = new FormData();

  Object.entries(values).forEach(([key, value]) => {
    if (value === undefined || value === null) {
      return;
    }
    formData.append(key, value);
  });

  return formData;
}

// requestJsonWithTimeout 输入请求地址、配置和超时毫秒数，输出为 JSON 数据，目的在于统一处理前端请求超时和网络异常。
export async function requestJsonWithTimeout(url, options = {}, timeoutMilliseconds = 15000) {
  const timeoutHandle = createTimeoutHandle(timeoutMilliseconds);

  try {
    const response = await fetch(url, {
      ...options,
      signal: timeoutHandle.controller.signal,
    });

    return await response.json();
  } catch (error) {
    if (error.name === "AbortError") {
      throw new Error("请求超时，请稍后重试");
    }
    throw error;
  } finally {
    clearTimeoutHandle(timeoutHandle);
  }
}

// redirectToLogin 输入目标地址，输出为页面跳转结果，目的在于在 token 失效时统一返回登录页。
export function redirectToLogin(targetPath = "/login") {
  window.location.href = targetPath;
}

// createAuthorizedOptions 输入请求方法、消息体与附加请求头，输出为鉴权请求配置，目的在于统一附带当前登录 token。
export function createAuthorizedOptions(method = "GET", body = null, extraHeaders = {}) {
  const headers = {
    token: getAuthToken(),
    ...extraHeaders,
  };

  if (!body) {
    return { method, headers };
  }

  return {
    method,
    headers,
    body,
  };
}

// normalizePayloadResult 输入后端响应数据，输出为 result 字段或原始结果，目的在于统一兼容当前接口响应结构。
export function normalizePayloadResult(payload) {
  if (payload && payload.result !== undefined) {
    return payload.result;
  }
  return payload;
}

// requestJson 输入请求地址和配置，输出为业务结果，目的在于统一根据 code 字段抛出接口错误。
export async function requestJson(url, options = {}, timeoutMilliseconds = 15000) {
  const payload = await requestJsonWithTimeout(url, options, timeoutMilliseconds);
  const payloadCode = Number(payload?.code ?? 200);
  const payloadMessage = payload?.message || payload?.msg || "请求失败";

  if (payloadCode === 400 && /login|token|认证|验证/i.test(String(payloadMessage))) {
    clearAuthToken();
    redirectToLogin();
    throw new Error(payloadMessage);
  }

  if (payloadCode !== 200) {
    throw new Error(payloadMessage);
  }

  return normalizePayloadResult(payload);
}
