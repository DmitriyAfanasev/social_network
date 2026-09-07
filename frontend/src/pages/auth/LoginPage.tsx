import { useState } from "react";
import type { FormEvent } from "react";

import type { AuthResponse } from "../../entities/user/model/user";
import { apiRequest, saveAuthTokens } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";
import { AuthLayout } from "../../shared/ui/AuthLayout";

export function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const result = await apiRequest<AuthResponse>("/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      saveAuthTokens(result);
      navigate("/profile/me");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка входа");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLayout
      title="Вход"
      aside="Войдите через email после подтверждения почты. Access-токен используется для запросов к Go API."
    >
      <form onSubmit={submit}>
        <label>
          Email
          <input
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            type="email"
            autoComplete="email"
            placeholder="you@example.com"
          />
        </label>
        <label>
          Пароль
          <input
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            type="password"
            autoComplete="current-password"
            placeholder="Введите пароль"
          />
        </label>
        <button type="button" className="auth-forgot-link" onClick={() => navigate("/password-reset")}>Забыли пароль?</button>
        {error && <p className="error">{error}</p>}
        <button disabled={loading}>{loading ? "Входим..." : "Войти"}</button>
      </form>
    </AuthLayout>
  );
}
