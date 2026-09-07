import { useMemo, useState } from "react";
import type { FormEvent } from "react";

import type { AuthResponse } from "../../entities/user/model/user";
import { apiRequest, saveAuthTokens } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";
import { AuthLayout } from "../../shared/ui/AuthLayout";

export function PasswordResetPage() {
  const token = useMemo(() => new URLSearchParams(window.location.search).get("token"), []);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [password2, setPassword2] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setMessage(null);
    setError(null);
    setLoading(true);

    try {
      if (!token) {
        const result = await apiRequest<{ message: string }>("/password-reset-requests", {
          method: "POST",
          body: JSON.stringify({ email }),
        });
        setMessage(result.message || "Если аккаунт существует, ссылка отправлена на почту.");
        return;
      }

      if (password !== password2) {
        setError("Пароли должны совпадать");
        return;
      }

      const result = await apiRequest<AuthResponse>("/password-resets", {
        method: "POST",
        body: JSON.stringify({ token, new_password: password }),
      });
      saveAuthTokens(result);
      setMessage("Пароль изменён. Открываем профиль...");
      window.setTimeout(() => navigate("/profile/me"), 500);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось восстановить доступ");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLayout
      title={token ? "Новый пароль" : "Восстановление пароля"}
      aside={token ? "Придумайте новый пароль. После сохранения вы сразу войдёте в аккаунт." : "Укажите email аккаунта — мы отправим на него ссылку для восстановления доступа."}
    >
      <form onSubmit={submit}>
        {!token ? (
          <label>
            Email
            <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" autoComplete="email" placeholder="you@example.com" required />
          </label>
        ) : (
          <>
            <label>
              Новый пароль
              <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" autoComplete="new-password" required />
            </label>
            <label>
              Повтор пароля
              <input value={password2} onChange={(event) => setPassword2(event.target.value)} type="password" autoComplete="new-password" required />
            </label>
          </>
        )}
        {message && <p className="success">{message}</p>}
        {error && <p className="error">{error}</p>}
        <button disabled={loading}>{loading ? "Подождите..." : token ? "Изменить пароль" : "Отправить ссылку"}</button>
        <button type="button" className="auth-back-link" onClick={() => navigate("/login")}>Вернуться ко входу</button>
      </form>
    </AuthLayout>
  );
}
