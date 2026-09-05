import { useState } from "react";
import type { FormEvent } from "react";

import type { MusicTrack } from "../../entities/music/model/music";
import { apiRequest } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";

/** Отдельная форма загрузки музыкального трека. */
export function AddMusicPage() {
  const [title, setTitle] = useState("");
  const [artist, setArtist] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!file) {
      setError("Сначала выберите аудиофайл.");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      const body = new FormData();
      body.append("file", file);
      body.append("title", title);
      body.append("artist", artist);
      await apiRequest<{ track: MusicTrack }>("/music", { method: "POST", body });
      navigate("/music");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить трек");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="music-page music-add-page">
      <header className="page-header music-hero">
        <div>
          <span className="eyebrow">Аудиотека</span>
          <h1>Добавить трек</h1>
          <p>Заполните информацию о треке — так его будет проще найти в вашей коллекции.</p>
        </div>
        <div className="music-hero-orb" aria-hidden="true">♫</div>
      </header>

      <form className="panel music-add-form" onSubmit={submit}>
        <div className="music-add-form-heading">
          <span className="music-add-icon" aria-hidden="true">♫</span>
          <div><h2>Информация о треке</h2><p>Автор и название будут видны рядом с аудиозаписью.</p></div>
        </div>
        <label>Название трека<input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Например, Ночной город" maxLength={200} /></label>
        <label>Автор / исполнитель<input value={artist} onChange={(event) => setArtist(event.target.value)} placeholder="Например, Алексей" maxLength={120} /></label>
        <label className="music-add-file-picker">
          <span className="music-add-file-icon" aria-hidden="true">↑</span>
          <span><strong>{file?.name ?? "Выберите аудиофайл"}</strong><small>MP3, WAV или другой поддерживаемый формат</small></span>
          <input type="file" accept="audio/*" onChange={(event) => setFile(event.target.files?.[0] ?? null)} />
        </label>
        {error && <p className="error">{error}</p>}
        <div className="form-actions">
          <button type="submit" disabled={saving || !file}>{saving ? "Загружаем..." : "Добавить трек"}</button>
          <button type="button" className="secondary" onClick={() => navigate("/music")}>Отмена</button>
        </div>
      </form>
    </section>
  );
}
