import { useEffect, useState, type ReactNode } from "react";

import type { ProfilePrivacy } from "../../entities/profile/model/profile";
import { apiRequest } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";

type PrivacyKey = keyof ProfilePrivacy;
type Visibility = "everyone" | "friends" | "friends_of_friends" | "nobody";

const visibilityOptions: Array<{ value: Visibility; label: string; icon: string }> = [
  { value: "everyone", label: "Все пользователи", icon: "◉" },
  { value: "friends", label: "Только друзья", icon: "♢" },
  { value: "friends_of_friends", label: "Друзья друзей", icon: "◈" },
  { value: "nobody", label: "Никто", icon: "—" },
];

const defaultPrivacy: ProfilePrivacy = {
  profile_visibility: "everyone",
  friend_request_policy: "everyone",
  message_policy: "everyone",
  phone_visibility: "everyone",
  birth_date_visibility: "everyone",
  gender_visibility: "everyone",
  location_visibility: "everyone",
  status_visibility: "everyone",
  friends_visibility: "friends",
  posts_visibility: "everyone",
  music_visibility: "friends_of_friends",
};

/** Показывает настройки профиля и приватности пользователя. */
export function SettingsPage() {
  const [privacy, setPrivacy] = useState<ProfilePrivacy>(defaultPrivacy);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    apiRequest<ProfilePrivacy>("/v1/profiles/me/privacy")
      .then((result) => setPrivacy({ ...defaultPrivacy, ...result }))
      .catch((err) => setError(err instanceof Error ? err.message : "Не удалось загрузить настройки"))
      .finally(() => setLoading(false));
  }, []);

  function update(key: PrivacyKey, value: Visibility) {
    setPrivacy((current) => ({ ...current, [key]: value }));
    setSaved(false);
  }

  async function save() {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      await apiRequest<ProfilePrivacy>("/v1/profiles/me/privacy", {
        method: "PUT",
        body: JSON.stringify(privacy),
      });
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить настройки");
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <p className="muted">Загружаем настройки...</p>;

  return (
    <section className="panel privacy-page">
      <header className="privacy-header">
        <div className="privacy-heading">
          <span className="privacy-heading-icon" aria-hidden="true">⌕</span>
          <div>
            <span className="eyebrow">Настройки аккаунта</span>
            <h1>Профиль и приватность</h1>
            <p>Вы сами решаете, что видят другие люди и как они могут с вами общаться.</p>
          </div>
        </div>
        <div className="privacy-header-actions">
          <button type="button" className="secondary" onClick={() => navigate("/profile/me/edit")}>Изменить профиль</button>
          <button type="button" className="privacy-profile-link" onClick={() => navigate("/profile/me")}>Открыть профиль <span aria-hidden="true">↗</span></button>
        </div>
      </header>

      <div className="privacy-tip">
        <span className="privacy-tip-icon" aria-hidden="true">✦</span>
        <div><strong>Настройки применяются сразу</strong><span>Изменения влияют на профиль, заявки в друзья, сообщения и разделы с контентом.</span></div>
      </div>

      <div className="privacy-sections">
        <PrivacySection icon="◎" title="Общение и контакты" description="Управляйте тем, кто может связаться с вами.">
          <PrivacySelect label="Видимость профиля" hint="Кто может открыть вашу страницу" value={privacy.profile_visibility as Visibility} onChange={(value) => update("profile_visibility", value)} />
          <PrivacySelect label="Сообщения" hint="Кто может начать диалог" value={privacy.message_policy as Visibility} onChange={(value) => update("message_policy", value)} />
          <PrivacySelect label="Заявки в друзья" hint="Кто может отправить вам заявку" value={privacy.friend_request_policy as Visibility} onChange={(value) => update("friend_request_policy", value)} />
          <PrivacySelect label="Список друзей" hint="Кто видит список ваших друзей" value={privacy.friends_visibility as Visibility} onChange={(value) => update("friends_visibility", value)} />
        </PrivacySection>

        <PrivacySection icon="◌" title="Личная информация" description="Выберите аудиторию для каждого поля профиля.">
          <PrivacySelect label="Номер телефона" hint="Контактный номер в профиле" value={privacy.phone_visibility as Visibility} onChange={(value) => update("phone_visibility", value)} />
          <PrivacySelect label="Дата рождения" hint="День и год рождения" value={privacy.birth_date_visibility as Visibility} onChange={(value) => update("birth_date_visibility", value)} />
          <PrivacySelect label="Пол" hint="Указанный в профиле пол" value={privacy.gender_visibility as Visibility} onChange={(value) => update("gender_visibility", value)} />
          <PrivacySelect label="Местоположение" hint="Страна, город и улица" value={privacy.location_visibility as Visibility} onChange={(value) => update("location_visibility", value)} />
          <PrivacySelect label="Статус" hint="Короткая подпись под именем" value={privacy.status_visibility as Visibility} onChange={(value) => update("status_visibility", value)} />
        </PrivacySection>

        <PrivacySection icon="▦" title="Разделы профиля" description="Настройте видимость публикаций и интересов.">
          <PrivacySelect label="Посты" hint="Ваши записи и публикации" value={privacy.posts_visibility as Visibility} onChange={(value) => update("posts_visibility", value)} />
          <PrivacySelect label="Музыка" hint="Загруженные треки и плейлист" value={privacy.music_visibility as Visibility} onChange={(value) => update("music_visibility", value)} />
        </PrivacySection>
      </div>

      {error && <p className="error">{error}</p>}
      <div className="privacy-footer">
        <div className={saved ? "privacy-save-status saved" : "privacy-save-status"}>
          <span className="privacy-save-dot" aria-hidden="true" />
          {saved ? "Изменения сохранены" : "Не забудьте сохранить изменения"}
        </div>
        <button type="button" disabled={saving} onClick={() => void save()}>{saving ? "Сохраняем..." : "Сохранить изменения"}</button>
      </div>
    </section>
  );
}

type PrivacySelectProps = {
  label: string;
  hint: string;
  value: Visibility;
  onChange: (value: Visibility) => void;
};

function PrivacySelect({ label, hint, value, onChange }: PrivacySelectProps) {
  const selectedOption = visibilityOptions.find((option) => option.value === value) ?? visibilityOptions[0];

  return (
    <label className="privacy-row">
      <span className="privacy-row-copy"><strong>{label}</strong><small>{hint}</small></span>
      <span className="privacy-select-wrap">
        <span className="privacy-select-icon" aria-hidden="true">{selectedOption.icon}</span>
      <select value={value} onChange={(event) => onChange(event.target.value as Visibility)}>
          {visibilityOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
      </span>
    </label>
  );
}

type PrivacySectionProps = {
  icon: string;
  title: string;
  description: string;
  children: ReactNode;
};

function PrivacySection({ icon, title, description, children }: PrivacySectionProps) {
  return (
    <section className="privacy-section">
      <header className="privacy-section-header">
        <span className="privacy-section-icon" aria-hidden="true">{icon}</span>
        <div><h2>{title}</h2><p>{description}</p></div>
      </header>
      <div className="privacy-rows">{children}</div>
    </section>
  );
}
