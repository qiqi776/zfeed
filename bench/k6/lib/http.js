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
  return check(res, {
    [`${name} status is 2xx`]: (r) => r.status >= 200 && r.status < 300,
  });
}

export function checkNon5xx(res, name) {
  return check(res, {
    [`${name} status is not 5xx`]: (r) => r.status < 500,
  });
}

