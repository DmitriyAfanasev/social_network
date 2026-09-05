import { useEffect, useState } from "react";

import type { MusicResponse, MusicTrack } from "../../entities/music/model/music";
import { apiRequest } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { formatDate } from "../../shared/lib/date";
import { navigate } from "../../shared/lib/navigation";
import { MediaPlayer } from "../../shared/ui/MediaPlayer";


export function MusicPage() {
  const [tracks, setTracks] = useState<MusicTrack[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    try {
      const result = await apiRequest<MusicResponse>("/music");
      setTracks(result.tracks);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить музыку");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  async function remove(trackId: string) {
    if (!window.confirm("Удалить этот трек из аудиотеки?")) return;
    try {
      await apiRequest(`/music/${trackId}`, { method: "DELETE" });
      setTracks((current) => current.filter((track) => track.id !== trackId));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить трек");
    }
  }

  return (
    <section className="music-page">
      <header className="page-header music-hero">
        <div>
          <span className="eyebrow">Личная аудиотека</span>
          <div className="music-title-row"><h1>Музыка</h1><span className="music-count-badge">{tracks.length}</span></div>
          <p>Соберите треки, которые хочется слушать рядом с историями и друзьями.</p>
        </div>
        <button type="button" className="music-add-button" onClick={() => navigate("/music/add")}><span aria-hidden="true">＋</span> Добавить трек</button>
      </header>

      {error && <p className="error">{error}</p>}
      {loading && <p className="muted">Загружаем аудиотеку...</p>}
      {!loading && tracks.length === 0 && <div className="empty-state"><h2>Пока нет треков</h2><p>Добавьте первый аудиофайл — он появится здесь с названием и исполнителем.</p></div>}
      {!loading && tracks.length > 0 && (
        <section className="panel music-library">
          <div className="section-title"><div><span className="eyebrow">Ваша коллекция</span><h2>Все треки</h2></div><span>{tracks.length}</span></div>
          <div className="music-library-list">
            {tracks.map((track) => (
              <article className="music-library-track" key={track.id}>
                <div className="music-track-icon" aria-hidden="true">♫</div>
                <div className="music-track-copy"><strong>{track.title}</strong><span>{track.artist || "Исполнитель не указан"}</span><small>добавлено {formatDate(track.created_at)}</small></div>
                <MediaPlayer kind="audio" src={`${API_BASE_URL}/v1/media/${track.media_id}/content`} title={`${track.title} — ${track.artist || "трек"}`} preload="none" />
                <button type="button" className="secondary music-library-delete" onClick={() => void remove(track.id)}>Удалить</button>
              </article>
            ))}
          </div>
        </section>
      )}
    </section>
  );
}
