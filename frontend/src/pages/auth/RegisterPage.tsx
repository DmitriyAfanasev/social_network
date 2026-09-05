import { useState } from "react";
import type { FormEvent } from "react";

import { apiRequest } from "../../shared/api/http";
import { AuthLayout } from "../../shared/ui/AuthLayout";

export function RegisterPage() {
  const [form, setForm] = useState({
    email: "",
    password: "",
  });
  const [password2, setPassword2] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function update(field: keyof typeof form, value: string) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setMessage(null);
    setError(null);
    setLoading(true);

    if (form.password !== password2) {
      setError("Пароли должны совпадать");
      setLoading(false);
      return;
    }

    try {
      const result = await apiRequest<{ message: string }>("/register", {
        method: "POST",
        body: JSON.stringify({ email: form.email, password: form.password }),
      });
      setMessage(result.message);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка регистрации");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLayout
      title="Регистрация"
      aside="Создайте учетную запись, подтвердите email в Mailpit, затем заполните профиль."
    >
      <form onSubmit={submit}>
        <label>
          Email
          <input value={form.email} onChange={(event) => update("email", event.target.value)} type="email" />
        </label>
        <label>
          Пароль
          <input value={form.password} onChange={(event) => update("password", event.target.value)} type="password" />
        </label>
        <label>
          Повтор пароля
          <input value={password2} onChange={(event) => setPassword2(event.target.value)} type="password" />
        </label>
        {message && <p className="success">{message}</p>}
        {error && <p className="error">{error}</p>}
        <button disabled={loading}>{loading ? "Отправляем..." : "Создать аккаунт"}</button>
      </form>
    </AuthLayout>
  );
}
