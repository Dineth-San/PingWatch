const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error ?? res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  register: (email: string, password: string) =>
    request("/api/auth/register", { method: "POST", body: JSON.stringify({ email, password }) }),

  login: (email: string, password: string) =>
    request("/api/auth/login", { method: "POST", body: JSON.stringify({ email, password }) }),

  logout: () => request("/api/auth/logout", { method: "POST" }),

  getMonitors: () => request<Monitor[]>("/api/monitors"),

  createMonitor: (data: { name: string; url: string; interval_seconds: number }) =>
    request<Monitor>("/api/monitors", { method: "POST", body: JSON.stringify(data) }),

  getMonitor: (id: string) => request<Monitor>(`/api/monitors/${id}`),

  updateMonitor: (id: string, data: { name: string; url: string; interval_seconds: number; is_active: boolean }) =>
    request<Monitor>(`/api/monitors/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  deleteMonitor: (id: string) =>
    request(`/api/monitors/${id}`, { method: "DELETE" }),

  getChecks: (id: string, params?: string) =>
    request<Check[]>(`/api/monitors/${id}/checks${params ? `?${params}` : ""}`),

  getIncidents: (id: string) => request<Incident[]>(`/api/monitors/${id}/incidents`),

  getStats: (id: string) =>
    request<Stats>(`/api/monitors/${id}/stats`),
};

export interface Monitor {
  ID: string;
  UserID: string;
  Name: string;
  URL: string;
  IntervalSeconds: number;
  IsActive: boolean;
  CreatedAt: string;
  // Present only when returned from the list endpoint (ListMonitorSummaries)
  IsUp?: boolean | null;
  ResponseTimeMs?: number | null;
  Uptime30d?: number;
}

export interface Check {
  ID: string;
  MonitorID: string;
  CheckedAt: string;
  StatusCode: number | null;
  ResponseTimeMs: number | null;
  IsUp: boolean;
  ErrorMessage: string | null;
}

export interface Incident {
  ID: string;
  MonitorID: string;
  StartedAt: string;
  ResolvedAt: string | null;
  DurationSeconds: number | null;
}

export interface Stats {
  uptime_1d: number;
  uptime_7d: number;
  uptime_30d: number;
  avg_response_ms: number;
  p95_response_ms: number;
}
