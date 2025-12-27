import { useState } from "react";

interface LoginProps {
  onLogin: (token: string, username: string) => void;
}

export default function Login({ onLogin }: LoginProps) {
  const [isLogin, setIsLogin] = useState(true);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    const endpoint = isLogin ? "/api/auth/login" : "/api/auth/register";

    try {
      const res = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      if (!res.ok) {
        const text = await res.text();
        setError(text || "Authentication failed");
        setLoading(false);
        return;
      }

      const data = await res.json();
      onLogin(data.token, data.username);
    } catch (err) {
      setError("Network error. Please try again.");
      setLoading(false);
    }
  }

  return (
    <div style={{
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      minHeight: "100vh",
      background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
    }}>
      <div style={{
        background: "white",
        padding: "40px",
        borderRadius: "12px",
        boxShadow: "0 10px 40px rgba(0,0,0,0.2)",
        width: "100%",
        maxWidth: "400px",
      }}>
        <h1 style={{
          textAlign: "center",
          marginBottom: "30px",
          color: "#333",
          fontSize: "28px",
          fontWeight: "600",
        }}>
          Torrent Seeker
        </h1>

        <div style={{
          display: "flex",
          marginBottom: "30px",
          borderRadius: "8px",
          overflow: "hidden",
          border: "1px solid #e0e0e0",
        }}>
          <button
            onClick={() => setIsLogin(true)}
            style={{
              flex: 1,
              padding: "12px",
              border: "none",
              background: isLogin ? "#667eea" : "white",
              color: isLogin ? "white" : "#666",
              cursor: "pointer",
              fontSize: "16px",
              fontWeight: "500",
              transition: "all 0.3s",
            }}
          >
            Login
          </button>
          <button
            onClick={() => setIsLogin(false)}
            style={{
              flex: 1,
              padding: "12px",
              border: "none",
              background: !isLogin ? "#667eea" : "white",
              color: !isLogin ? "white" : "#666",
              cursor: "pointer",
              fontSize: "16px",
              fontWeight: "500",
              transition: "all 0.3s",
            }}
          >
            Register
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom: "20px" }}>
            <label style={{
              display: "block",
              marginBottom: "8px",
              color: "#555",
              fontSize: "14px",
              fontWeight: "500",
            }}>
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              minLength={3}
              style={{
                width: "100%",
                padding: "12px",
                border: "1px solid #ddd",
                borderRadius: "6px",
                fontSize: "16px",
                boxSizing: "border-box",
              }}
              placeholder="Enter username"
            />
          </div>

          <div style={{ marginBottom: "20px" }}>
            <label style={{
              display: "block",
              marginBottom: "8px",
              color: "#555",
              fontSize: "14px",
              fontWeight: "500",
            }}>
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={6}
              style={{
                width: "100%",
                padding: "12px",
                border: "1px solid #ddd",
                borderRadius: "6px",
                fontSize: "16px",
                boxSizing: "border-box",
              }}
              placeholder="Enter password"
            />
          </div>

          {error && (
            <div style={{
              padding: "12px",
              background: "#fee",
              color: "#c33",
              borderRadius: "6px",
              marginBottom: "20px",
              fontSize: "14px",
            }}>
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            style={{
              width: "100%",
              padding: "14px",
              background: loading ? "#999" : "#667eea",
              color: "white",
              border: "none",
              borderRadius: "6px",
              fontSize: "16px",
              fontWeight: "600",
              cursor: loading ? "not-allowed" : "pointer",
              transition: "background 0.3s",
            }}
          >
            {loading ? "Please wait..." : isLogin ? "Login" : "Register"}
          </button>
        </form>

        <p style={{
          marginTop: "20px",
          textAlign: "center",
          color: "#888",
          fontSize: "13px",
        }}>
          {isLogin ? "New user? " : "Already have an account? "}
          <a
            href="#"
            onClick={(e) => {
              e.preventDefault();
              setIsLogin(!isLogin);
              setError("");
            }}
            style={{
              color: "#667eea",
              textDecoration: "none",
              fontWeight: "500",
            }}
          >
            {isLogin ? "Register here" : "Login here"}
          </a>
        </p>
      </div>
    </div>
  );
}
