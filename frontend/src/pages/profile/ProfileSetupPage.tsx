import { useEffect, useState } from "react";
import type { FormEvent } from "react";

import { apiRequest } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";
import { AuthLayout } from "../../shared/ui/AuthLayout";

type ProfileSetupResponse = {
  first_name?: string | null;
  last_name?: string | null;
};

export function ProfileSetupPage() {
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiRequest<ProfileSetupResponse>("/v1/profiles/me")
      .then((profile) => {
        setFirstName(profile.first_name ?? "");
        setLastName(profile.last_name ?? "");
      })
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const first = firstName.trim();
    const last = lastName.trim();
    if (!first || !last) {
      setError("Введите имя и фамилию, чтобы продолжить.");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await apiRequest("/v1/profiles/me", {
        method: "PATCH",
        body: JSON.stringify({ first_name: first, last_name: last }),
      });
      navigate("/profile/me");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить профиль");
    } finally {
      setSaving(false);
    }
  }

  return (
    <AuthLayout
      title="Почти готово"
      aside="Укажите имя и фамилию — они будут отображаться в постах, друзьях и сообщениях вместо email. Остальные данные можно добавить позже."
    >
      {loading ? <p className="muted">Загружаем профиль...</p> : (
        <form onSubmit={submit} className="profile-setup-form">
          <label>
            Имя
            <input value={firstName} onChange={(event) => setFirstName(event.target.value)} autoComplete="given-name" maxLength={80} required autoFocus />
          </label>
          <label>
            Фамилия
            <input value={lastName} onChange={(event) => setLastName(event.target.value)} autoComplete="family-name" maxLength={80} required />
          </label>
          {error && <p className="error">{error}</p>}
          <button disabled={saving}>{saving ? "Сохраняем..." : "Продолжить"}</button>
        </form>
      )}
    </AuthLayout>
  );
}
