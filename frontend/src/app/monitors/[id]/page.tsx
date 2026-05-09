"use client";
import { useState } from "react";
import useSWR, { mutate as globalMutate } from "swr";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, Incident } from "@/lib/api";
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from "recharts";

export default function MonitorDetailPage({ params }: { params: { id: string } }) {
  const { id } = params;
  const router = useRouter();
  const [editing, setEditing] = useState(false);

  const { data: monitor, error, mutate: mutateMonitor } = useSWR(
    `monitor-${id}`,
    () => api.getMonitor(id),
  );
  const { data: stats } = useSWR(
    `stats-${id}`,
    () => api.getStats(id),
    { refreshInterval: 60_000 },
  );
  const { data: checks } = useSWR(
    `checks-${id}`,
    () => api.getChecks(id, "limit=288"),
    { refreshInterval: 60_000 },
  );
  const { data: incidents } = useSWR(
    `incidents-${id}`,
    () => api.getIncidents(id),
    { refreshInterval: 60_000 },
  );

  async function handleDelete() {
    if (!confirm(`Delete "${monitor?.Name}"? This cannot be undone.`)) return;
    await api.deleteMonitor(id);
    await globalMutate("monitors");
    router.push("/dashboard");
  }

  if (error) return (
    <>
      <Nav />
      <div className="container">
        <p className="error-msg" style={{ marginBottom: 12 }}>Monitor not found.</p>
        <Link href="/dashboard" className="btn btn-ghost">← Dashboard</Link>
      </div>
    </>
  );

  if (!monitor) return (
    <>
      <Nav />
      <div className="container">
        <p style={{ color: "var(--muted)" }}>Loading…</p>
      </div>
    </>
  );

  const latest = checks?.[0];
  const isUp = latest?.IsUp ?? null;
  const isPending = isUp === null;
  const dotColor = isPending ? "var(--muted)" : isUp ? "var(--success)" : "var(--danger)";
  const dotClass = !isPending && !isUp ? "dot-pulse" : "";
  const statusLabel = isPending ? "Pending" : isUp ? "Up" : "Down";

  const chartData = [...(checks ?? [])].reverse().map(c => ({
    time: new Date(c.CheckedAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
    ms: c.ResponseTimeMs,
    isUp: c.IsUp,
  }));

  return (
    <>
      <Nav />
      <div className="container">

        {/* Header card */}
        <div className="card" style={{ marginBottom: 20 }}>
          <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 16, flexWrap: "wrap" }}>
            <div style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
              <span
                className={dotClass}
                style={{
                  position: "relative",
                  display: "inline-block",
                  width: 12,
                  height: 12,
                  borderRadius: "50%",
                  background: dotColor,
                  flexShrink: 0,
                  marginTop: 5,
                }}
              />
              <div>
                <h1 style={{ fontSize: 20, fontWeight: 800, color: "var(--text)", marginBottom: 4 }}>
                  {monitor.Name}
                </h1>
                <a
                  href={monitor.URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ color: "var(--muted)", fontSize: 13, wordBreak: "break-all" }}
                >
                  {monitor.URL}
                </a>
              </div>
            </div>
            <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
              <span style={{
                fontSize: 12,
                fontWeight: 700,
                padding: "4px 12px",
                borderRadius: 9999,
                background: isPending ? "var(--primary-light)" : isUp ? "var(--success-light)" : "var(--danger-light)",
                color: isPending ? "var(--muted)" : isUp ? "#16a34a" : "#dc2626",
              }}>
                {statusLabel}
              </span>
              <button
                className="btn btn-ghost"
                style={{ padding: "6px 14px", fontSize: 13 }}
                onClick={() => setEditing(e => !e)}
              >
                {editing ? "Cancel" : "Edit"}
              </button>
              <button
                className="btn btn-danger"
                style={{ padding: "6px 14px", fontSize: 13 }}
                onClick={handleDelete}
              >
                Delete
              </button>
            </div>
          </div>
        </div>

        {/* Inline edit form */}
        {editing && (
          <EditForm
            name={monitor.Name}
            url={monitor.URL}
            intervalSeconds={monitor.IntervalSeconds}
            isActive={monitor.IsActive}
            onSave={async (data) => {
              await mutateMonitor(api.updateMonitor(id, data));
              setEditing(false);
            }}
            onCancel={() => setEditing(false)}
          />
        )}

        {/* Stats row */}
        <div className="stat-grid" style={{ marginBottom: 20 }}>
          <StatCard label="Uptime 1d"     value={stats ? `${stats.uptime_1d.toFixed(1)}%`      : "—"} />
          <StatCard label="Uptime 7d"     value={stats ? `${stats.uptime_7d.toFixed(1)}%`      : "—"} />
          <StatCard label="Uptime 30d"    value={stats ? `${stats.uptime_30d.toFixed(1)}%`     : "—"} />
          <StatCard label="Avg response"  value={stats ? `${Math.round(stats.avg_response_ms)}ms` : "—"} />
          <StatCard label="p95 response"  value={stats ? `${Math.round(stats.p95_response_ms)}ms` : "—"} />
        </div>

        {/* Response time chart */}
        <div className="card" style={{ marginBottom: 20 }}>
          <p style={{ fontWeight: 700, fontSize: 14, marginBottom: 16 }}>
            Response time — last 24h
          </p>
          {chartData.length === 0 ? (
            <p style={{ color: "var(--muted)", fontSize: 13 }}>No data yet.</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={chartData} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
                <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 11, fill: "var(--muted)" }}
                  interval="preserveStartEnd"
                />
                <YAxis
                  tick={{ fontSize: 11, fill: "var(--muted)" }}
                  unit="ms"
                  width={54}
                />
                <Tooltip
                  contentStyle={{
                    background: "var(--surface)",
                    border: "1.5px solid var(--border)",
                    borderRadius: 10,
                    fontSize: 12,
                  }}
                  labelStyle={{ color: "var(--muted)", marginBottom: 4 }}
                  formatter={(v) => [`${v}ms`, "Response time"]}
                />
                <Line
                  type="monotone"
                  dataKey="ms"
                  stroke="var(--primary)"
                  strokeWidth={2}
                  connectNulls
                  dot={(props: {cx?: number; cy?: number; payload?: {isUp: boolean; ms: number}}) => {
                    const { cx = 0, cy = 0, payload } = props;
                    const down = payload?.isUp === false;
                    return (
                      <circle
                        key={`dot-${cx}-${cy}`}
                        cx={cx}
                        cy={cy}
                        r={down ? 4 : 2}
                        fill={down ? "var(--danger)" : "var(--primary)"}
                        stroke={down ? "var(--danger-light)" : "none"}
                        strokeWidth={down ? 2 : 0}
                      />
                    );
                  }}
                  activeDot={{ r: 5, strokeWidth: 0 }}
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Incident history */}
        <div className="card">
          <p style={{ fontWeight: 700, fontSize: 14, marginBottom: 16 }}>Incident history</p>
          {!incidents || incidents.length === 0 ? (
            <p style={{ color: "var(--muted)", fontSize: 13 }}>No incidents recorded — looking good! 🎉</p>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Started at</th>
                  <th>Resolved at</th>
                  <th>Duration</th>
                </tr>
              </thead>
              <tbody>
                {incidents.map(i => <IncidentRow key={i.ID} incident={i} />)}
              </tbody>
            </table>
          )}
        </div>

      </div>
    </>
  );
}

function Nav() {
  return (
    <nav className="nav">
      <span className="nav-brand">📡 PingWatch</span>
      <Link href="/dashboard" className="btn btn-ghost">← Dashboard</Link>
    </nav>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat-card">
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
    </div>
  );
}

function EditForm({
  name: initName,
  url: initUrl,
  intervalSeconds: initInterval,
  isActive: initActive,
  onSave,
  onCancel,
}: {
  name: string;
  url: string;
  intervalSeconds: number;
  isActive: boolean;
  onSave: (data: { name: string; url: string; interval_seconds: number; is_active: boolean }) => Promise<void>;
  onCancel: () => void;
}) {
  const [name, setName] = useState(initName);
  const [url, setUrl] = useState(initUrl);
  const [interval, setIntervalVal] = useState(initInterval);
  const [isActive, setIsActive] = useState(initActive);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await onSave({ name, url, interval_seconds: interval, is_active: isActive });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Update failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="card" style={{ marginBottom: 20, borderColor: "var(--primary)" }}>
      <p style={{ fontWeight: 700, fontSize: 14, marginBottom: 16 }}>Edit monitor</p>
      <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
          <div className="form-group">
            <label>Name</label>
            <input value={name} onChange={e => setName(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>URL</label>
            <input type="url" value={url} onChange={e => setUrl(e.target.value)} required />
          </div>
        </div>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, alignItems: "end" }}>
          <div className="form-group">
            <label>Interval</label>
            <select value={interval} onChange={e => setIntervalVal(Number(e.target.value))}>
              <option value={60}>Every 1 minute</option>
              <option value={120}>Every 2 minutes</option>
              <option value={300}>Every 5 minutes</option>
              <option value={600}>Every 10 minutes</option>
            </select>
          </div>
          <div className="form-group">
            <label>Status</label>
            <select value={isActive ? "active" : "paused"} onChange={e => setIsActive(e.target.value === "active")}>
              <option value="active">Active</option>
              <option value="paused">Paused</option>
            </select>
          </div>
        </div>
        {error && <p className="error-msg">{error}</p>}
        <div style={{ display: "flex", gap: 10 }}>
          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? "Saving…" : "Save changes"}
          </button>
          <button type="button" className="btn btn-ghost" onClick={onCancel}>
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}

function IncidentRow({ incident }: { incident: Incident }) {
  return (
    <tr>
      <td style={{ fontSize: 13 }}>{new Date(incident.StartedAt).toLocaleString()}</td>
      <td style={{ fontSize: 13 }}>
        {incident.ResolvedAt
          ? new Date(incident.ResolvedAt).toLocaleString()
          : <span style={{ color: "var(--danger)", fontWeight: 600 }}>Ongoing</span>
        }
      </td>
      <td style={{ fontSize: 13 }}>
        {incident.DurationSeconds != null ? formatDuration(incident.DurationSeconds) : "—"}
      </td>
    </tr>
  );
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${(seconds / 3600).toFixed(1)}h`;
}
