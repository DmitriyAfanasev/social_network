import { useEffect, useState } from "react";
import type { FormEvent } from "react";

import type { ProfileForm } from "../../entities/profile/model/profile";
import { emptyStringsToNull } from "../../entities/profile/model/profile";
import { apiRequest } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";

type ProfileDTO = { handle?: string | null; bio: string } & Partial<ProfileForm>;
type EditProfilePageProps = { id: string };

const emptyForm: ProfileForm = {
  first_name: "", last_name: "", middle_name: "", birth_date: "", gender: "",
  phone_number: "", country: "", city: "", street: "", bio: "", status: "",
};

/** Форма редактирования полей, которые поддерживает новый profiles API. */
export function EditProfilePage({ id }: EditProfilePageProps) {
  const [handle, setHandle] = useState("");
  const [form, setForm] = useState<ProfileForm>(emptyForm);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    apiRequest<ProfileDTO>("/v1/profiles/me")
      .then((profile) => {
        setHandle(profile.handle ?? "");
        setForm({
          first_name: profile.first_name ?? "", last_name: profile.last_name ?? "", middle_name: profile.middle_name ?? "",
          birth_date: profile.birth_date ?? "", gender: profile.gender ?? "", phone_number: profile.phone_number ?? "",
          country: profile.country ?? "", city: profile.city ?? "", street: profile.street ?? "",
          bio: profile.bio ?? "", status: profile.status ?? "",
        });
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Не удалось загрузить профиль"))
      .finally(() => setLoading(false));
  }, [id]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError(null);
    try {
      await apiRequest("/v1/profiles/me", {
        method: "PATCH",
        body: JSON.stringify(emptyStringsToNull(form)),
      });
      if (handle.trim()) {
        await apiRequest("/v1/profiles/me/handle", {
          method: "PUT",
          body: JSON.stringify({ handle: handle.trim().toLowerCase() }),
        });
      }
      navigate("/profile/me");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить профиль");
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <p className="muted">Загружаем профиль...</p>;

  function updateField(field: keyof ProfileForm, value: string) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  return (
    <section className="panel edit-panel">
      <header className="page-header">
        <div><span className="eyebrow">Профиль</span><h1>Редактирование</h1></div>
        <button type="button" className="secondary" onClick={() => navigate("/profile/me")}>Назад</button>
      </header>
      <form className="profile-form" onSubmit={submit}>
        <div className="form-section-heading"><strong>Основное</strong><span>Эти данные можно скрыть настройками приватности.</span></div>
        <label>Handle<input value={handle} onChange={(event) => setHandle(event.target.value)} placeholder="например, alex" pattern="[a-z0-9_]{3,32}" /></label>
        <label>Имя<input value={form.first_name} onChange={(event) => updateField("first_name", event.target.value)} maxLength={80} /></label>
        <label>Фамилия<input value={form.last_name} onChange={(event) => updateField("last_name", event.target.value)} maxLength={80} /></label>
        <label>Отчество<input value={form.middle_name} onChange={(event) => updateField("middle_name", event.target.value)} maxLength={80} /></label>
        <label>Дата рождения<input type="date" value={form.birth_date} onChange={(event) => updateField("birth_date", event.target.value)} /></label>
        <label>Пол<input value={form.gender} onChange={(event) => updateField("gender", event.target.value)} maxLength={32} /></label>
        <label>Телефон<input value={form.phone_number} onChange={(event) => updateField("phone_number", event.target.value)} maxLength={32} /></label>
        <label>Страна<input value={form.country} onChange={(event) => updateField("country", event.target.value)} maxLength={80} /></label>
        <label>Город<input value={form.city} onChange={(event) => updateField("city", event.target.value)} maxLength={80} /></label>
        <label>Улица<input value={form.street} onChange={(event) => updateField("street", event.target.value)} maxLength={160} /></label>
        <label>Статус<input value={form.status} onChange={(event) => updateField("status", event.target.value)} maxLength={160} /></label>
        <label className="form-field-wide">О себе<textarea value={form.bio} onChange={(event) => updateField("bio", event.target.value)} maxLength={2000} rows={6} /></label>
        {error && <p className="error">{error}</p>}
        <div className="form-actions">
          <button disabled={saving}>{saving ? "Сохраняем..." : "Сохранить"}</button>
          <button type="button" className="secondary" onClick={() => navigate("/profile/me")}>Отмена</button>
        </div>
      </form>
    </section>
  );
}
