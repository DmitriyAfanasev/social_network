import { useEffect, useState } from "react";

import type { MusicResponse, MusicTrack } from "../../entities/music/model/music";
import { apiRequest } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { formatDate } from "../../shared/lib/date";
import { navigate } from "../../shared/lib/navigation";
import { MusicTrackControl } from "../../shared/ui/GlobalMusicPlayer";

type MusicOwner = {
  id: string;
  username: string;
  full_name: string;
  avatar: string | null;
};

type MusicOwnerSearchResponse = {
  users: MusicOwner[];
  posts: unknown[];
  videos: unknown[];
};

export function MusicPage() {
  const [tracks, setTracks] = useState<MusicTrack[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [owners, setOwners] = useState<MusicOwner[]>([]);
  const [ownersLoading, setOwnersLoading] = useState(false);
  const [ownersError, setOwnersError] = useState<string | null>(null);

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

  useEffect(() => {
    const query = search.trim();
    if (query.length < 2) {
      setOwners([]);
      setOwnersError(null);
      setOwnersLoading(false);
      return;
    }

    let active = true;
    const timer = window.setTimeout(() => {
      setOwnersLoading(true);
      void apiRequest<MusicOwnerSearchResponse>(`/search?q=${encodeURIComponent(query)}`)
        .then((result) => {
          if (active) {
            setOwners(result.users);
            setOwnersError(null);
          }
        })
        .catch((err: unknown) => {
          if (active) {
            setOwnersError(err instanceof Error ? err.message : "Не удалось найти пользователей");
          }
        })
        .finally(() => {
          if (active) setOwnersLoading(false);
        });
    }, 280);

    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [search]);

  async function remove(trackId: string) {
    const track = tracks.find((item) => item.id === trackId);
    const saved = Boolean(track?.is_saved);
    if (!window.confirm(saved ? "Убрать этот трек из плейлиста?" : "Удалить этот трек из аудиотеки?")) return;
    try {
      await apiRequest(`/music/${trackId}${saved ? "/save" : ""}`, { method: "DELETE" });
      setTracks((current) => current.filter((track) => track.id !== trackId));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить трек");
    }
  }

  const visibleTracks = tracks.filter((track) => {
    const needle = search.trim().toLocaleLowerCase();
    if (!needle) return true;
    return `${track.title} ${track.artist}`.toLocaleLowerCase().includes(needle);
  });
  return (
    <section className="music-page">
      <section className="music-workspace">
        <nav className="music-tabs" aria-label="Разделы музыки">
          <span className="music-tab active">Главная</span>
          <span className="music-tab">Моя музыка</span>
          <span className="music-tab">Обзор</span>
          <span className="music-tab">Радио</span>
          <span className="music-tab">Обновления</span>
          <span className="music-tab-count">{tracks.length} треков</span>
          <button type="button" className="music-add-button" onClick={() => navigate("/music/add")}>+ Добавить трек</button>
        </nav>

        <label className="music-search">
          <span aria-hidden="true">⌕</span>
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Поиск трека или пользователя" />
        </label>

        {search.trim().length >= 2 && (
          <section className="music-owner-results" aria-label="Плейлисты пользователей">
            <div className="music-owner-results-heading">
              <div><span className="eyebrow">Музыка друзей и авторов</span><h2>Плейлисты пользователей</h2></div>
              <span>Откройте профиль, чтобы слушать всю аудиотеку</span>
            </div>
            {ownersLoading && <p className="muted">Ищем пользователей...</p>}
            {ownersError && <p className="error">{ownersError}</p>}
            {!ownersLoading && !ownersError && owners.length === 0 && <p className="music-filter-empty">Пользователей с таким именем не найдено.</p>}
            {!ownersLoading && owners.length > 0 && (
              <div className="music-owner-list">
                {owners.map((owner) => (
                  <button type="button" className="music-owner-card" key={owner.id} onClick={() => navigate(`/profile/${owner.id}`)}>
                    <span className="music-owner-avatar">
                      {owner.avatar ? <img src={owner.avatar.startsWith("http") ? owner.avatar : `${API_BASE_URL}${owner.avatar}`} alt="" /> : owner.full_name.slice(0, 1).toUpperCase()}
                    </span>
                    <span className="music-owner-copy"><strong>{owner.full_name}</strong><small>{owner.username ? `@${owner.username}` : "Пользователь"}</small></span>
                    <span className="music-owner-link">Открыть плейлист →</span>
                  </button>
                ))}
              </div>
            )}
          </section>
        )}

        <section className="music-mix-card">
          <div className="music-mix-copy">
            <span className="eyebrow">Персональная подборка</span>
            <h1>Слушать музыку<br />под своё настроение</h1>
            <p>Собирайте любимые треки в одном месте и возвращайтесь к ним в любой момент.</p>
            <button type="button" className="music-mix-button" onClick={() => document.querySelector(".music-library")?.scrollIntoView({ behavior: "smooth" })}>
              <span aria-hidden="true">▶</span> Слушать микс
            </button>
          </div>
          <div className="music-mix-art" aria-hidden="true"><span>♫</span><i /><i /><i /></div>
        </section>

        {error && <p className="error">{error}</p>}
        {loading && <p className="muted">Загружаем аудиотеку...</p>}
        {!loading && tracks.length === 0 && <div className="empty-state music-empty"><h2>Пока нет треков</h2><p>Загрузите аудиофайл на странице добавления — он появится здесь с названием и исполнителем.</p></div>}
        {!loading && tracks.length > 0 && (
          <section className="music-library">
            <div className="music-library-heading"><div><span className="eyebrow">Ваша коллекция</span><h2>Мои треки</h2></div><span>{visibleTracks.length} из {tracks.length}</span></div>
            {visibleTracks.length > 0 ? (
              <div className="music-library-list">
                {visibleTracks.map((track) => (
                  <article className="music-library-track" key={track.id}>
                    <div className="music-library-track-main">
                      <MusicTrackControl track={track} queue={visibleTracks} />
                      <div className="music-album-art" aria-hidden="true"><span>♫</span></div>
                      <div className="music-track-copy"><strong>{track.title}</strong><span>{track.artist || "Исполнитель не указан"}</span><small>{track.is_saved ? "сохранено в плейлист" : `добавлено ${formatDate(track.created_at)}`}</small></div>
                      <button type="button" className="secondary music-library-delete" onClick={() => void remove(track.id)}>{track.is_saved ? "Убрать" : "Удалить"}</button>
                    </div>
                  </article>
                ))}
              </div>
            ) : <p className="music-filter-empty">По запросу ничего не найдено.</p>}
          </section>
        )}
      </section>
    </section>
  );
}
