import { useState, useEffect } from "react";
import Login from "./Login";
import App from "./AppContent";

export default function AuthWrapper() {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem("authToken"));
  const [username, setUsername] = useState<string | null>(() => localStorage.getItem("username"));

  function handleLogin(newToken: string, newUsername: string) {
    setToken(newToken);
    setUsername(newUsername);
    localStorage.setItem("authToken", newToken);
    localStorage.setItem("username", newUsername);
  }

  function handleLogout() {
    setToken(null);
    setUsername(null);
    localStorage.removeItem("authToken");
    localStorage.removeItem("username");
  }

  if (!token) {
    return <Login onLogin={handleLogin} />;
  }

  return <App token={token} username={username} onLogout={handleLogout} />;
}
