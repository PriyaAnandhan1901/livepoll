const BASE = (import.meta.env.VITE_API_URL || "http://localhost:8081").replace(/\/$/, "");
export const API = `${BASE}/api`;

function read(key) {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}
function write(key, value) {
  try {
    localStorage.setItem(key, value);
  } catch {
    /* storage unavailable: ignore */
  }
}

export async function request(path, { method = "GET", body, auth = false } = {}) {
  const headers = {};
  if (body) headers["Content-Type"] = "application/json";
  const token = read("lp_token");
  if (auth && token) headers.Authorization = `Bearer ${token}`;

  let res;
  try {
    res = await fetch(API + path, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new Error("Can't reach the server. Check your connection and try again.");
  }
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const err = new Error(data.error || "Something went wrong. Try again.");
    err.status = res.status;
    throw err;
  }
  return data;
}

export const streamUrl = (id) => `${API}/polls/${id}/stream`;

// Anonymous voter identity: one vote per browser per poll.
export function getVoterId() {
  let v = read("lp_voter");
  if (!v) {
    v =
      typeof crypto !== "undefined" && crypto.randomUUID
        ? crypto.randomUUID()
        : Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join("");
    write("lp_voter", v);
  }
  return v;
}

export function getVoted(pollId) {
  const v = read(`lp_voted_${pollId}`);
  return v === null ? null : parseInt(v, 10);
}
export function setVoted(pollId, index) {
  write(`lp_voted_${pollId}`, String(index));
}
