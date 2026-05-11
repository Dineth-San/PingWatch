"use client";
import useSWR from "swr";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, Monitor } from "@/lib/api";

export default function DashboardPage() {
  const router = useRouter();
  const { data: monitors, error, isLoading } = useSWR<Monitor[]>(
    "monitors",
    () => api.getMonitors(),
    { refreshInterval: 30_000 }
  );

  async function handleLogout() {
    await api.logout();
    router.push("/login");
  }

  if (error?.message?.includes("unauthorized")) {
    router.push("/login");
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", minHeight: "100vh" }}>
        <p style={{ color: "var(--muted)" }}>Redirecting…</p>
      </div>
    );
  }

  // Sort: down first, then pending, then up
  const sorted = monitors
    ? [...monitors].sort((a, b) => {
        const rank = (m: Monitor) =>
          m.IsUp === false ? 0 : m.IsUp == null ? 1 : 2;
        return rank(a) - rank(b);
      })
    : [];

  const upCount   = monitors?.filter(m => m.IsUp === true).length  ?? 0;
  const downCount = monitors?.filter(m => m.IsUp === false).length ?? 0;

  return (
    <>
      <nav className="nav">
        <span className="nav-brand">📡 PingWatch</span>
        <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
          <Link href="/monitors/new" className="btn btn-primary">+ Add monitor</Link>
          <button className="btn btn-ghost" onClick={handleLogout}>Logout</button>
        </div>
      </nav>

      <div className="container">
        {/* Header row */}
        <div className="page-header">
          <div>
            <h1 className="page-title">Your monitors</h1>
            {monitors && monitors.length > 0 && (
              <div style={{ display: "flex", gap: 8, marginTop: 6, alignItems: "center" }}>
                {downCount > 0 && (
                  <span style={{
                    fontSize: 12, fontWeight: 700,
                    padding: "2px 10px", borderRadius: 9999,
                    background: "var(--danger-light)", color: "#dc2626",
                  }}>
                    {downCount} down
                  </span>
                )}
                {upCount > 0 && (
                  <span style={{
                    fontSize: 12, fontWeight: 700,
                    padding: "2px 10px", borderRadius: 9999,
                    background: "var(--success-light)", color: "#16a34a",
                  }}>
                    {upCount} up
                  </span>
                )}
                <span style={{ fontSize: 12, color: "var(--muted)" }}>
                  · refreshes every 30s
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Loading skeletons */}
        {isLoading && (
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            {[1, 2, 3].map(i => (
              <div key={i} style={{
                height: 112,
                background: "var(--surface)",
                border: "1.5px solid var(--border)",
                borderRadius: "var(--radius)",
                opacity: 0.5,
                animation: "pulse 1.5s ease infinite",
              }} />
            ))}
          </div>
        )}

        {/* Empty state */}
        {!isLoading && monitors?.length === 0 && (
          <div className="card animate-in" style={{ textAlign: "center", padding: "64px 24px" }}>
            <div style={{ fontSize: 44, marginBottom: 14 }}>📭</div>
            <p style={{ fontWeight: 700, fontSize: 17, marginBottom: 6 }}>No monitors yet</p>
            <p style={{ color: "var(--muted)", marginBottom: 24, fontSize: 14 }}>
              Add your first URL to start tracking uptime and response times.
            </p>
            <Link href="/monitors/new" className="btn btn-primary">Add your first monitor</Link>
          </div>
        )}

        {/* Monitor list */}
        {sorted.length > 0 && (
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }} className="animate-in">
            {sorted.map(m => (
              <MonitorCard key={m.ID} monitor={m} />
            ))}
          </div>
        )}
      </div>
    </>
  );
}

function MonitorCard({ monitor: m }: { monitor: Monitor }) {
  const isUp      = m.IsUp;
  const isPending = isUp == null;

  const accentColor = isPending ? "#cbd5e1" : isUp ? "var(--success)" : "var(--danger)";
  const badgeBg     = isPending ? "var(--primary-light)" : isUp ? "var(--success-light)" : "var(--danger-light)";
  const badgeColor  = isPending ? "var(--muted)" : isUp ? "#16a34a" : "#dc2626";
  const statusLabel = isPending ? "Pending" : isUp ? "Up" : "Down";
  const dotClass    = !isPending && !isUp ? "dot-pulse" : "";

  const uptime  = m.Uptime30d ?? 0;
  const uptimeColor = uptime >= 99 ? "#16a34a" : uptime >= 95 ? "#ca8a04" : "#dc2626";
  const displayUrl  = m.URL.replace(/^https?:\/\//, "").replace(/\/$/, "");

  return (
    <Link href={`/monitors/${m.ID}`} style={{ textDecoration: "none" }}>
      <div
        style={{
          background: "var(--surface)",
          border: "1.5px solid var(--border)",
          borderRadius: "var(--radius)",
          boxShadow: `var(--shadow), inset 4px 0 0 ${accentColor}`,
          padding: "18px 22px",
          cursor: "pointer",
          transition: "transform 0.15s, box-shadow 0.15s",
          display: "grid",
          gridTemplateColumns: "1fr auto",
          gap: "10px 24px",
          alignItems: "center",
        }}
        onMouseEnter={e => {
          const el = e.currentTarget as HTMLDivElement;
          el.style.transform = "translateY(-2px) translateX(2px)";
          el.style.boxShadow = `0 8px 24px rgba(100,120,220,0.14), inset 4px 0 0 ${accentColor}`;
        }}
        onMouseLeave={e => {
          const el = e.currentTarget as HTMLDivElement;
          el.style.transform = "";
          el.style.boxShadow = `var(--shadow), inset 4px 0 0 ${accentColor}`;
        }}
      >
        {/* Left: name + URL + uptime bar */}
        <div style={{ minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 3 }}>
            <span
              className={dotClass}
              style={{
                position: "relative",
                display: "inline-block",
                width: 8,
                height: 8,
                borderRadius: "50%",
                background: accentColor,
                flexShrink: 0,
              }}
            />
            <span style={{
              fontWeight: 700,
              fontSize: 15,
              color: "var(--text)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}>
              {m.Name}
            </span>
          </div>
          <div style={{
            color: "var(--muted)",
            fontSize: 12,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            marginBottom: 10,
            paddingLeft: 18,
          }}>
            {displayUrl}
          </div>
          {/* Uptime progress bar */}
          {!isPending && (
            <div style={{ paddingLeft: 18 }}>
              <div style={{
                height: 3,
                background: "var(--border)",
                borderRadius: 99,
                overflow: "hidden",
                maxWidth: 320,
              }}>
                <div style={{
                  height: "100%",
                  width: `${Math.min(100, uptime)}%`,
                  background: uptimeColor,
                  borderRadius: 99,
                  transition: "width 0.6s ease",
                }} />
              </div>
            </div>
          )}
        </div>

        {/* Right: status badge + metrics */}
        <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end", gap: 10 }}>
          <span style={{
            fontSize: 12, fontWeight: 700,
            padding: "4px 12px",
            borderRadius: 9999,
            background: badgeBg,
            color: badgeColor,
            flexShrink: 0,
          }}>
            {statusLabel}
          </span>
          <div style={{ display: "flex", gap: 20 }}>
            <MiniStat label="30d uptime"  value={isPending ? "—" : `${uptime.toFixed(1)}%`} color={isPending ? undefined : uptimeColor} />
            <MiniStat label="Last ping"   value={m.ResponseTimeMs != null ? `${m.ResponseTimeMs}ms` : "—"} />
            <MiniStat label="Every"       value={formatInterval(m.IntervalSeconds)} />
          </div>
        </div>
      </div>
    </Link>
  );
}

function MiniStat({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end", gap: 1 }}>
      <span style={{ fontSize: 10, color: "var(--muted)", fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.06em" }}>
        {label}
      </span>
      <span style={{ fontSize: 14, fontWeight: 700, color: color ?? "var(--text)" }}>
        {value}
      </span>
    </div>
  );
}

function formatInterval(seconds: number): string {
  if (seconds === 60)  return "1 min";
  if (seconds === 120) return "2 min";
  if (seconds < 3600)  return `${seconds / 60} min`;
  return `${seconds / 3600}h`;
}
