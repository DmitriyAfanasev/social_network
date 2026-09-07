import { useEffect, useRef, useState } from "react";

import { apiRequest } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { navigate, usePath } from "../../shared/lib/navigation";
import { MediaPlayer } from "../../shared/ui/MediaPlayer";

type Video = {
  video_id: string | number;
  media_id: string | number;
  title: string;
  owner_id: string | number | null;
  owner_name: string;
  status: string;
  error: string | null;
  original_filename: string;
  created_at: string | null;
  duration: number | null;
  views_count: number;
  likes_count: number;
  is_liked_by_current: boolean;
  is_bookmarked_by_current: boolean;
  is_favorited_by_current: boolean;
};

type Album = { id: string | number; title: string; videos: Video[] };
type VideoResponse = { owner_name?: string; albums: Album[] };
type Tab = "uploaded" | "favorite" | "viewed" | "bookmarked";
type VideosPageProps = { profileId?: string };

const videoTabs: Array<{ value: Tab; label: string }> = [
  { value: "uploaded", label: "Загруженные" },
  { value: "favorite", label: "Избранное" },
  { value: "viewed", label: "Просмотренные" },
  { value: "bookmarked", label: "Смотреть позже" },
];

function formatDuration(duration: number | null) {
  if (duration == null) return "Длительность уточняется";
  const totalSeconds = Math.max(0, Math.floor(duration));
  return `${Math.floor(totalSeconds / 60)}:${String(totalSeconds % 60).padStart(2, "0")}`;
}

function formatDate(createdAt: string | null) {
  return createdAt ? new Date(createdAt).toLocaleDateString("ru-RU") : "Дата уточняется";
}

export function VideosPage({ profileId }: VideosPageProps) {
  const path = usePath();
  const [albums, setAlbums] = useState<Album[]>([]);
  const [albumTitle, setAlbumTitle] = useState("");
  const [uploadTitle, setUploadTitle] = useState("");
  const [ownerName, setOwnerName] = useState("пользователя");
  const [viewerId, setViewerId] = useState<string | number | null>(null);
  const [tab, setTab] = useState<Tab>("uploaded");
  const [search, setSearch] = useState("");
  const [selectedVideo, setSelectedVideo] = useState<Video | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyAlbum, setBusyAlbum] = useState<string | number | null>(null);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const videoSessionID = useRef<string | null>(null);
  const reportedProgress = useRef(0);
  const playbackRef = useRef({ currentTime: 0, duration: 0 });

  const isOwner = viewerId !== null && (!profileId || profileId === "me" || String(profileId) === String(viewerId));
  const videos = albums.flatMap((album) => album.videos);
  const otherVideos = selectedVideo ? videos.filter((video) => video.video_id !== selectedVideo.video_id) : [];

  function updateVideo(videoId: string | number, update: Partial<Video>) {
    setAlbums((current) => current.map((album) => ({
      ...album,
      videos: album.videos.map((video) => video.video_id === videoId ? { ...video, ...update } : video),
    })));
    setSelectedVideo((current) => current?.video_id === videoId ? { ...current, ...update } : current);
  }

  async function load() {
    setLoading(true);
    try {
      const query = new URLSearchParams({ tab, q: search.trim() });
      const viewerProfileRequest = apiRequest<{ user_id: string }>("/v1/profiles/me");
      const viewerProfile = await viewerProfileRequest;
      const ownerId = profileId === "me" ? viewerProfile.user_id : profileId;
      const videosPath = ownerId ? `/v1/media/videos?owner_id=${encodeURIComponent(ownerId)}&${query}` : `/videos?${query}`;
      const [result, feed] = await Promise.all([
        apiRequest<VideoResponse>(videosPath),
        viewerProfileRequest,
      ]);
      setAlbums(result.albums);
      setViewerId(feed.user_id);
      setOwnerName(result.owner_name || result.albums.flatMap((album) => album.videos)[0]?.owner_name || "пользователя");
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить видео");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [profileId, tab, search]);

  useEffect(() => {
    const queryTab = new URLSearchParams(path.split("?")[1] ?? "").get("tab");
    if (queryTab === "uploaded" || queryTab === "favorite" || queryTab === "viewed" || queryTab === "bookmarked") setTab(queryTab);
  }, [path]);

  async function createAlbum() {
    if (!albumTitle.trim()) return;
    try {
      const body = new FormData();
      body.append("title", albumTitle.trim());
      await apiRequest("/videos/albums", { method: "POST", body });
      setAlbumTitle("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось создать альбом");
    }
  }

  async function upload(albumId: string | number, file: File) {
    setBusyAlbum(albumId);
    try {
      const body = new FormData();
      body.append("file", file);
      body.append("album_id", String(albumId));
      if (uploadTitle.trim()) body.append("title", uploadTitle.trim());
      await apiRequest("/videos", { method: "POST", body });
      setUploadTitle("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить видео");
    } finally {
      setBusyAlbum(null);
    }
  }

  async function deleteVideo(videoId: string | number) {
    if (!window.confirm("Удалить это видео?")) return;
    try {
      await apiRequest(`/videos/${videoId}`, { method: "DELETE" });
      setSelectedVideo(null);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить видео");
    }
  }

  async function deleteAlbum(albumId: string | number) {
    if (!window.confirm("Удалить альбом и все видео внутри?")) return;
    try {
      await apiRequest(`/videos/albums/${albumId}`, { method: "DELETE" });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить альбом");
    }
  }

  async function recordView(video: Video) {
    if (selectedVideo && selectedVideo.video_id !== video.video_id) {
      reportVideoProgress(playbackRef.current.currentTime, playbackRef.current.duration, false, true);
    }
    setSelectedVideo(video);
    videoSessionID.current = createSessionID();
    reportedProgress.current = 0;
    playbackRef.current = { currentTime: 0, duration: 0 };
    try {
      const result = await apiRequest<{ views_count: number }>(`/videos/${video.video_id}/view`, {
        method: "POST",
        body: JSON.stringify({ session_id: videoSessionID.current }),
      });
      updateVideo(video.video_id, { views_count: result.views_count });
    } catch {
      // Просмотр не должен блокировать открытие плеера.
    }
  }

  function reportVideoProgress(currentTime: number, duration: number, completed = false, force = false) {
    if (!selectedVideo || !videoSessionID.current || !Number.isFinite(currentTime) || currentTime < 0) return;
    playbackRef.current = { currentTime, duration };
    const safeDuration = Number.isFinite(duration) && duration > 0 ? duration : 0;
    const delta = Math.max(0, currentTime - reportedProgress.current);
    if (!completed && !force && delta < 10) return;
    reportedProgress.current = Math.max(reportedProgress.current, currentTime);
    void apiRequest(`/videos/${selectedVideo.video_id}/view`, {
      method: "POST",
      body: JSON.stringify({
        session_id: videoSessionID.current,
        watch_seconds: delta,
        progress_seconds: currentTime,
        duration_seconds: safeDuration,
        completed,
      }),
    }).catch(() => {
      // Телеметрия не должна прерывать воспроизведение.
    });
  }

  function closeVideo() {
    reportVideoProgress(playbackRef.current.currentTime, playbackRef.current.duration, false, true);
    setSelectedVideo(null);
  }

  async function toggleLike() {
    if (!selectedVideo) return;
    if (viewerId === null) {
      navigate("/login");
      return;
    }
    setBusyAction("like");
    try {
      const result = await apiRequest<{ likes_count: number; liked: boolean }>(`/videos/${selectedVideo.video_id}/like`, { method: "POST" });
      updateVideo(selectedVideo.video_id, { likes_count: result.likes_count, is_liked_by_current: result.liked });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось поставить лайк");
    } finally {
      setBusyAction(null);
    }
  }

  async function toggleBookmark() {
    if (!selectedVideo) return;
    if (viewerId === null) {
      navigate("/login");
      return;
    }
    setBusyAction("bookmark");
    try {
      const method = selectedVideo.is_bookmarked_by_current ? "DELETE" : "POST";
      const result = await apiRequest<{ bookmarked: boolean }>(`/videos/${selectedVideo.video_id}/bookmark`, { method });
      updateVideo(selectedVideo.video_id, { is_bookmarked_by_current: result.bookmarked });
      if (tab === "bookmarked" && !result.bookmarked) await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось изменить список добавленных видео");
    } finally {
      setBusyAction(null);
    }
  }

  async function toggleFavorite() {
    if (!selectedVideo) return;
    if (viewerId === null) {
      navigate("/login");
      return;
    }
    setBusyAction("favorite");
    try {
      const method = selectedVideo.is_favorited_by_current ? "DELETE" : "POST";
      const result = await apiRequest<{ favorited: boolean }>(`/videos/${selectedVideo.video_id}/favorite`, { method });
      updateVideo(selectedVideo.video_id, { is_favorited_by_current: result.favorited });
      if (tab === "favorite" && !result.favorited) await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось изменить избранное");
    } finally {
      setBusyAction(null);
    }
  }

  const tabDescription: Record<Tab, string> = {
    uploaded: "Соберите здесь моменты, которыми хочется поделиться.",
    favorite: "Видео, к которым хочется быстро вернуться.",
    viewed: "История видео, которые вы уже открывали.",
    bookmarked: "Видео, сохранённые для просмотра позже.",
  };

  return (
    <section className="photos-page videos-page">
      <header className="page-header photos-header videos-hero">
        <div className="videos-hero-copy">
          <span className="eyebrow">Медиатека</span>
          <div className="videos-title-row"><h1>{isOwner || !profileId ? "Мои видео" : `Видео ${ownerName}`}</h1><span className="video-count-badge">{videos.length}</span></div>
          <p>{tabDescription[tab]}</p>
        </div>
        <div className="videos-hero-orb" aria-hidden="true">▶</div>
      </header>

      <div className="video-toolbar panel">
        <div className="video-tabs" role="tablist" aria-label="Раздел видео">
          {videoTabs.map((videoTab) => isOwner || videoTab.value === "uploaded" ? (
            <button key={videoTab.value} type="button" className={tab === videoTab.value ? "" : "secondary"} onClick={() => setTab(videoTab.value)}>{videoTab.label}</button>
          ) : null)}
        </div>
        <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Поиск по названию видео" aria-label="Поиск по названию видео" />
      </div>

      {isOwner && tab === "uploaded" && (
        <div className="photo-upload-create panel video-create-panel">
          <input value={albumTitle} onChange={(event) => setAlbumTitle(event.target.value)} placeholder="Название нового альбома" maxLength={80} />
          <input value={uploadTitle} onChange={(event) => setUploadTitle(event.target.value)} placeholder="Название видео при загрузке" maxLength={200} />
          <button type="button" onClick={() => void createAlbum()}>Создать альбом</button>
        </div>
      )}

      {error && <p className="error">{error}</p>}
      {loading && <p className="muted">Загружаем видео...</p>}
      {!loading && videos.length === 0 && <div className="empty-state"><h2>Видео не найдены</h2><p>{search ? "Попробуйте изменить поисковый запрос." : isOwner && tab === "uploaded" ? "Создайте альбом и добавьте первое видео." : "У пользователя пока нет видео."}</p></div>}

      <div className="photo-album-list">
        {albums.map((album) => (
          <article className="photo-album panel" key={`${tab}-${album.id}`}>
            <header className="photo-album-header">
              <div><h2>{album.title}</h2><span>{album.videos.length} видео</span></div>
              {isOwner && tab === "uploaded" && Boolean(album.id) && <><button type="button" className="secondary" onClick={() => void deleteAlbum(album.id)}>Удалить альбом</button><label className="secondary button-like">{busyAlbum === album.id ? "Загрузка..." : "Добавить видео"}<input type="file" accept="video/*" disabled={busyAlbum !== null} onChange={(event) => { const file = event.target.files?.[0]; event.target.value = ""; if (file) void upload(album.id, file); }} /></label></>}
            </header>
            <div className="photo-grid">
              {album.videos.map((video) => (
                <button type="button" className="photo-tile video-tile" key={video.video_id} onClick={() => void recordView(video)}>
                  <video src={`${API_BASE_URL}/v1/media/${video.media_id}/content`} muted preload="metadata" />
                  <span className="video-meta"><strong>{video.title || video.original_filename || "Видео"}</strong><span>{formatDuration(video.duration)} · {formatDate(video.created_at)}</span><span>◉ {video.views_count} · ♥ {video.likes_count}</span></span>
                </button>
              ))}
            </div>
          </article>
        ))}
      </div>

      {selectedVideo && (
        <div className="video-modal-backdrop" role="presentation" onClick={closeVideo}>
          <div className="video-modal panel" role="dialog" aria-modal="true" aria-label={selectedVideo.title} onClick={(event) => event.stopPropagation()}>
            <button type="button" className="video-modal-close secondary" onClick={closeVideo} aria-label="Закрыть">×</button>
            <div className="video-modal-main">
              <MediaPlayer
                kind="video"
                className="video-player"
                src={`${API_BASE_URL}/v1/media/${selectedVideo.media_id}/content`}
                title={selectedVideo.title}
                autoPlay
                onProgress={reportVideoProgress}
                onPause={(currentTime, duration) => reportVideoProgress(currentTime, duration, false, true)}
                onComplete={(currentTime, duration) => reportVideoProgress(currentTime, duration, true)}
              />
              <div className="video-modal-info">
                <span className="eyebrow">{selectedVideo.owner_name}</span>
                <h2>{selectedVideo.title || selectedVideo.original_filename}</h2>
                <p className="muted">{formatDuration(selectedVideo.duration)} · добавлено {formatDate(selectedVideo.created_at)} · просмотров {selectedVideo.views_count}</p>
                <div className="video-actions">
                  <button type="button" className={selectedVideo.is_liked_by_current ? "" : "secondary"} disabled={busyAction !== null} onClick={() => void toggleLike()}>♥ {selectedVideo.likes_count}</button>
                  <button type="button" className={selectedVideo.is_favorited_by_current ? "" : "secondary"} disabled={busyAction !== null} onClick={() => void toggleFavorite()}>{selectedVideo.is_favorited_by_current ? "★ В избранном" : "☆ В избранное"}</button>
                  {selectedVideo.owner_id !== viewerId && <button type="button" className={selectedVideo.is_bookmarked_by_current ? "" : "secondary"} disabled={busyAction !== null} onClick={() => void toggleBookmark()}>{selectedVideo.is_bookmarked_by_current ? "✓ Смотреть позже" : "＋ Смотреть позже"}</button>}
                  {isOwner && selectedVideo.owner_id === viewerId && <button type="button" className="secondary" onClick={() => void deleteVideo(selectedVideo.video_id)}>Удалить видео</button>}
                </div>
              </div>
            </div>
            <aside className="video-other-list"><h2>Другие видео</h2>{otherVideos.length === 0 && <p className="muted">Других видео пока нет.</p>}{otherVideos.map((video) => <button type="button" className="video-other-item" key={video.video_id} onClick={() => void recordView(video)}><strong>{video.title || video.original_filename}</strong><span>{formatDuration(video.duration)} · ◉ {video.views_count}</span></button>)}</aside>
          </div>
        </div>
      )}
    </section>
  );
}

function createSessionID() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
  return `video-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
