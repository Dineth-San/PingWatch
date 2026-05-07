import Link from "next/link";

export default function LandingPage() {
  return (
    <>
      {/* ─── Navbar ─────────────────────────────────────────────── */}
      <nav style={{
        background: "#1a1e36",
        borderBottom: "1px solid rgba(255,255,255,0.07)",
        position: "sticky",
        top: 0,
        zIndex: 50,
        padding: "0 32px",
        height: 60,
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
      }}>
        <span style={{ fontWeight: 800, fontSize: 18, color: "#fff", letterSpacing: "-0.02em" }}>
          📡 PingWatch
        </span>
        <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
          <Link href="/login" style={{
            color: "rgba(255,255,255,0.7)",
            fontWeight: 500,
            fontSize: 14,
            padding: "6px 14px",
            borderRadius: 10,
            textDecoration: "none",
          }}>
            Sign in
          </Link>
          <Link href="/register" style={{
            background: "var(--primary)",
            color: "#fff",
            fontWeight: 600,
            fontSize: 14,
            padding: "8px 18px",
            borderRadius: 10,
            textDecoration: "none",
            boxShadow: "0 2px 8px rgba(123,158,248,0.35)",
          }}>
            Get started
          </Link>
        </div>
      </nav>

      {/* ─── Hero ───────────────────────────────────────────────── */}
      <section style={{
        background: "linear-gradient(160deg, #eef2ff 0%, #f5f0ff 60%, #f2f5ff 100%)",
        padding: "96px 24px 80px",
        textAlign: "center",
      }}>
        <div style={{ maxWidth: 660, margin: "0 auto" }}>
          <div style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 8,
            background: "white",
            border: "1.5px solid var(--border)",
            borderRadius: 9999,
            padding: "4px 14px",
            marginBottom: 28,
            fontSize: 13,
            fontWeight: 600,
            color: "var(--primary)",
            boxShadow: "0 1px 4px rgba(123,158,248,0.12)",
          }}>
            <span style={{ width: 7, height: 7, borderRadius: "50%", background: "var(--success)", display: "inline-block" }} />
            Free to start — no credit card required
          </div>

          <h1 style={{
            fontSize: "clamp(36px, 5vw, 54px)",
            fontWeight: 900,
            lineHeight: 1.15,
            color: "#1a1e36",
            marginBottom: 20,
            letterSpacing: "-0.03em",
          }}>
            Know the moment your<br />
            <span style={{ color: "var(--primary)" }}>website goes down</span>
          </h1>

          <p style={{
            fontSize: 18,
            color: "var(--muted)",
            marginBottom: 40,
            lineHeight: 1.65,
            maxWidth: 500,
            margin: "0 auto 40px",
          }}>
            PingWatch checks your URLs every minute, records response times,
            and alerts you the instant something goes wrong.
          </p>

          <div style={{ display: "flex", gap: 14, justifyContent: "center", flexWrap: "wrap" }}>
            <Link href="/register" style={{
              background: "var(--primary)",
              color: "#fff",
              fontWeight: 700,
              fontSize: 16,
              padding: "14px 32px",
              borderRadius: 14,
              textDecoration: "none",
              boxShadow: "0 4px 16px rgba(123,158,248,0.4)",
            }}>
              Start monitoring free →
            </Link>
            <a href="#features" style={{
              background: "#fff",
              color: "var(--text)",
              fontWeight: 600,
              fontSize: 16,
              padding: "14px 32px",
              borderRadius: 14,
              textDecoration: "none",
              border: "1.5px solid var(--border)",
              boxShadow: "var(--shadow)",
            }}>
              See how it works
            </a>
          </div>
        </div>
      </section>

      {/* ─── Features ───────────────────────────────────────────── */}
      <section id="features" style={{ padding: "80px 24px", background: "var(--bg)" }}>
        <div style={{ maxWidth: 960, margin: "0 auto" }}>
          <h2 style={{
            textAlign: "center",
            fontSize: 30,
            fontWeight: 800,
            color: "var(--text)",
            marginBottom: 8,
            letterSpacing: "-0.02em",
          }}>
            Everything you need to stay on top of uptime
          </h2>
          <p style={{ textAlign: "center", color: "var(--muted)", marginBottom: 48, fontSize: 16 }}>
            Simple, focused, and fast.
          </p>

          <div style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(260px, 1fr))",
            gap: 20,
          }}>
            <FeatureCard
              icon="🔔"
              title="Instant alerts"
              body="Get notified the moment your site goes down — not minutes later. Every second of downtime counts."
              accent="#7b9ef8"
            />
            <FeatureCard
              icon="⚡"
              title="Response time tracking"
              body="See exactly how fast your pages load over time. Spot slowdowns before they frustrate your users."
              accent="#a78bfa"
            />
            <FeatureCard
              icon="📋"
              title="Incident history"
              body="Every outage is logged with start time, end time, and duration. No guessing what happened or when."
              accent="#34d399"
            />
          </div>
        </div>
      </section>

      {/* ─── How it works ───────────────────────────────────────── */}
      <section style={{
        padding: "80px 24px",
        background: "linear-gradient(160deg, #f5f0ff 0%, #eef2ff 100%)",
      }}>
        <div style={{ maxWidth: 720, margin: "0 auto", textAlign: "center" }}>
          <h2 style={{
            fontSize: 30,
            fontWeight: 800,
            color: "var(--text)",
            marginBottom: 8,
            letterSpacing: "-0.02em",
          }}>
            Up and running in seconds
          </h2>
          <p style={{ color: "var(--muted)", marginBottom: 56, fontSize: 16 }}>
            Three steps. No configuration files. No infrastructure.
          </p>

          <div style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
            gap: 16,
            position: "relative",
          }}>
            <Step number={1} title="Add your URL" body="Paste any http or https URL and pick a check interval." />
            <Step number={2} title="We check it" body="PingWatch pings your URL on schedule and records the result." />
            <Step number={3} title="Get alerted" body="If your site goes down, you know instantly. When it's back, you know that too." />
          </div>
        </div>
      </section>

      {/* ─── Final CTA ──────────────────────────────────────────── */}
      <section style={{
        padding: "80px 24px",
        textAlign: "center",
        background: "var(--bg)",
      }}>
        <div style={{ maxWidth: 520, margin: "0 auto" }}>
          <h2 style={{
            fontSize: 32,
            fontWeight: 900,
            color: "#1a1e36",
            marginBottom: 12,
            letterSpacing: "-0.02em",
          }}>
            Start monitoring in 30 seconds
          </h2>
          <p style={{ color: "var(--muted)", marginBottom: 36, fontSize: 16 }}>
            Free to start. No credit card. Cancel any time.
          </p>
          <Link href="/register" style={{
            display: "inline-block",
            background: "var(--primary)",
            color: "#fff",
            fontWeight: 700,
            fontSize: 16,
            padding: "14px 36px",
            borderRadius: 14,
            textDecoration: "none",
            boxShadow: "0 4px 16px rgba(123,158,248,0.4)",
          }}>
            Create your free account →
          </Link>
        </div>
      </section>

      {/* ─── Footer ─────────────────────────────────────────────── */}
      <footer style={{
        background: "#1a1e36",
        padding: "24px 32px",
        textAlign: "center",
        color: "rgba(255,255,255,0.35)",
        fontSize: 13,
      }}>
        © {new Date().getFullYear()} PingWatch — built to keep you informed.
      </footer>
    </>
  );
}

function FeatureCard({ icon, title, body, accent }: {
  icon: string;
  title: string;
  body: string;
  accent: string;
}) {
  return (
    <div style={{
      background: "#fff",
      border: "1.5px solid var(--border)",
      borderRadius: 18,
      padding: "28px 24px",
      boxShadow: "var(--shadow)",
      display: "flex",
      flexDirection: "column",
      gap: 12,
    }}>
      <span style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        width: 48,
        height: 48,
        borderRadius: 14,
        background: `${accent}18`,
        fontSize: 24,
      }}>
        {icon}
      </span>
      <h3 style={{ fontSize: 17, fontWeight: 800, color: "var(--text)", margin: 0 }}>{title}</h3>
      <p style={{ fontSize: 14, color: "var(--muted)", lineHeight: 1.65, margin: 0 }}>{body}</p>
    </div>
  );
}

function Step({ number, title, body }: { number: number; title: string; body: string }) {
  return (
    <div style={{
      background: "#fff",
      border: "1.5px solid var(--border)",
      borderRadius: 18,
      padding: "28px 20px",
      boxShadow: "var(--shadow)",
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      gap: 12,
      textAlign: "center",
    }}>
      <span style={{
        width: 40,
        height: 40,
        borderRadius: "50%",
        background: "var(--primary)",
        color: "#fff",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontWeight: 900,
        fontSize: 16,
        boxShadow: "0 3px 10px rgba(123,158,248,0.35)",
      }}>
        {number}
      </span>
      <h3 style={{ fontSize: 16, fontWeight: 800, color: "var(--text)", margin: 0 }}>{title}</h3>
      <p style={{ fontSize: 13, color: "var(--muted)", lineHeight: 1.65, margin: 0 }}>{body}</p>
    </div>
  );
}
