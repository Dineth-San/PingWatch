"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }
    setLoading(true);
    try {
      await api.register(email, password);
      router.push("/dashboard");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{
      minHeight: "100vh",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      padding: "0 24px",
      background: "var(--bg)",
    }}>
      <div style={{ width: "100%", maxWidth: 400 }}>
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <span style={{
            display: "inline-block",
            width: 48,
            height: 48,
            borderRadius: "50%",
            background: "var(--primary-light)",
            marginBottom: 16,
            lineHeight: "48px",
            fontSize: 24,
          }}>📡</span>
          <h1 style={{ fontSize: 24, fontWeight: 800, color: "var(--text)", marginBottom: 4 }}>
            PingWatch
          </h1>
          <p style={{ color: "var(--muted)", fontSize: 14 }}>Create your free account</p>
        </div>

        <div className="card">
          <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 18 }}>
            <div className="form-group">
              <label>Email</label>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="you@example.com"
                required
                autoFocus
              />
            </div>
            <div className="form-group">
              <label>Password</label>
              <input
                type="password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="Min. 8 characters"
                required
                minLength={8}
              />
            </div>
            {error && <p className="error-msg">{error}</p>}
            <button
              type="submit"
              className="btn btn-primary"
              disabled={loading}
              style={{ width: "100%", marginTop: 4 }}
            >
              {loading ? "Creating account…" : "Create account"}
            </button>
          </form>
        </div>

        <p style={{ marginTop: 20, textAlign: "center", color: "var(--muted)", fontSize: 14 }}>
          Already have an account?{" "}
          <Link href="/login" style={{ fontWeight: 600 }}>Sign in</Link>
        </p>
      </div>
    </div>
  );
}
