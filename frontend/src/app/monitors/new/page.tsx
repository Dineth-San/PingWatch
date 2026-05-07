"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";

type FieldErrors = {
  name?: string;
  url?: string;
  interval?: string;
  general?: string;
};

function parseApiError(message: string): FieldErrors {
  if (message.includes("name")) return { name: message };
  if (message.includes("url") || message.includes("URL")) return { url: message };
  if (message.includes("interval")) return { interval: message };
  return { general: message };
}

function validateUrl(value: string): string {
  if (!value) return "URL is required";
  try {
    const u = new URL(value);
    if (u.protocol !== "http:" && u.protocol !== "https:") {
      return "URL must start with http:// or https://";
    }
  } catch {
    return "Enter a valid URL starting with http:// or https://";
  }
  return "";
}

export default function NewMonitorPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [url, setUrl] = useState("https://");
  const [interval, setIntervalVal] = useState(60);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [loading, setLoading] = useState(false);

  function clearFieldError(field: keyof FieldErrors) {
    setErrors(prev => ({ ...prev, [field]: undefined }));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();

    const urlError = validateUrl(url);
    if (!name.trim() || urlError) {
      setErrors({
        name: !name.trim() ? "Name is required" : undefined,
        url: urlError || undefined,
      });
      return;
    }

    setErrors({});
    setLoading(true);
    try {
      const m = await api.createMonitor({ name: name.trim(), url, interval_seconds: interval });
      router.push(`/monitors/${m.ID}`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to create monitor";
      setErrors(parseApiError(msg));
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <nav className="nav">
        <span className="nav-brand">📡 PingWatch</span>
        <Link href="/dashboard" className="btn btn-ghost">← Dashboard</Link>
      </nav>

      <div className="container" style={{ maxWidth: 520 }}>
        <div className="page-header">
          <h1 className="page-title">Add monitor</h1>
        </div>

        <div className="card">
          <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 20 }}>

            <div className="form-group">
              <label>Name</label>
              <input
                type="text"
                value={name}
                onChange={e => { setName(e.target.value); clearFieldError("name"); }}
                placeholder="My Website"
                autoFocus
                style={errors.name ? { borderColor: "var(--danger)" } : {}}
              />
              {errors.name && <span className="error-msg" style={{ padding: "4px 0", background: "none" }}>{errors.name}</span>}
            </div>

            <div className="form-group">
              <label>URL</label>
              <input
                type="text"
                value={url}
                onChange={e => { setUrl(e.target.value); clearFieldError("url"); }}
                placeholder="https://example.com"
                style={errors.url ? { borderColor: "var(--danger)" } : {}}
              />
              {errors.url && <span className="error-msg" style={{ padding: "4px 0", background: "none" }}>{errors.url}</span>}
            </div>

            <div className="form-group">
              <label>Check interval</label>
              <select
                value={interval}
                onChange={e => { setIntervalVal(Number(e.target.value)); clearFieldError("interval"); }}
                style={errors.interval ? { borderColor: "var(--danger)" } : {}}
              >
                <option value={60}>Every 1 minute</option>
                <option value={120}>Every 2 minutes</option>
                <option value={300}>Every 5 minutes</option>
                <option value={600}>Every 10 minutes</option>
              </select>
              {errors.interval && <span className="error-msg" style={{ padding: "4px 0", background: "none" }}>{errors.interval}</span>}
            </div>

            {errors.general && <p className="error-msg">{errors.general}</p>}

            <div style={{ display: "flex", gap: 12, marginTop: 4 }}>
              <button type="submit" className="btn btn-primary" disabled={loading}>
                {loading ? "Creating…" : "Create monitor"}
              </button>
              <Link href="/dashboard" className="btn btn-ghost">Cancel</Link>
            </div>

          </form>
        </div>
      </div>
    </>
  );
}
