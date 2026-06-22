import http from "k6/http";
import { check } from "k6";

export function jsonHeaders(token) {
  const headers = {
    "Content-Type": "application/json",
    "X-Benchmark": "zfeed",
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

export function jsonField(res, path, fallback = undefined) {
  try {
    const value = res.json(path);
    return value === null || value === undefined ? fallback : value;
  } catch (_) {
    return fallback;
  }
}

export function postJSON(baseURL, path, payload, token, tags) {
  return http.post(`${baseURL}${path}`, JSON.stringify(payload || {}), {
    headers: jsonHeaders(token),
    tags,
  });
}

export function getJSON(baseURL, path, token, tags) {
  return http.get(`${baseURL}${path}`, {
    headers: jsonHeaders(token),
    tags,
  });
}

export function deleteJSON(baseURL, path, payload, token, tags) {
  return http.del(`${baseURL}${path}`, JSON.stringify(payload || {}), {
    headers: jsonHeaders(token),
    tags,
  });
}

export function check2xx(res, name) {
  const ok = check(res, {
    [`${name} status is 2xx`]: (r) => r.status >= 200 && r.status < 300,
  });
  logFailedCheck(ok, res, name);
  return ok;
}

export function checkNon5xx(res, name) {
  const ok = check(res, {
    [`${name} status is not 5xx`]: (r) => r.status < 500,
  });
  logFailedCheck(ok, res, name);
  return ok;
}

function logFailedCheck(ok, res, name) {
  if (ok || __ENV.BENCH_DEBUG_FAILURES !== "1") {
    return;
  }

  console.error(
    `bench_check_failed name=${name} status=${res.status} request_id=${responseHeader(
      res,
      "X-Request-Id",
    )} body=${responseBodySummary(res)}`,
  );
}

function responseHeader(res, name) {
  if (!res || !res.headers) {
    return "-";
  }
  return res.headers[name] || res.headers[name.toLowerCase()] || res.headers[name.toUpperCase()] || "-";
}

function responseBodySummary(res) {
  const body = String((res && res.body) || "")
    .replace(/\s+/g, " ")
    .trim();
  if (body.length <= 300) {
    return body || "-";
  }
  return `${body.slice(0, 300)}...`;
}
