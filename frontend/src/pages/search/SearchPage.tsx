import { useEffect, useState } from "react";

import { apiRequest } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { formatDate } from "../../shared/lib/date";
import { navigate, usePath } from "../../shared/lib/navigation";


type SearchUser = { id: number; username: string; full_name: string; avatar: string | null };
type SearchPost = { id: number; content: string | null; author_id: number; author_name: string; created_at: string };
type SearchVideo = { video_id: number; media_id: number; title: string; owner_id: number | null; owner_name: string; created_at: string };
type SearchResponse = { users: SearchUser[]; posts: SearchPost[]; videos: SearchVideo[] };


export function SearchPage() {
  const path = usePath();
  const query = new URLSearchParams(path.split("?")[1] ?? "").get("q")?.trim() ?? "";
  const [result, setResult] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!query) {
      setResult({ users: [], posts: [], videos: [] });
      return;
    }
    setLoading(true);
    void apiRequest<SearchResponse>(`/search?q=${encodeURIComponent(query)}`)
      .then((response) => {
        setResult(response);
        setError(null);
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Не удалось выполнить поиск"))
      .finally(() => setLoading(false));
  }, [query]);

  const total = (result?.users.length ?? 0) + (result?.posts.length ?? 0) + (result?.videos.length ?? 0);

  return (
    <section className="search-page">
      <header className="page-header search-hero">
        <div><span className="eyebrow">Поиск по General</span><h1>{query ? `Результаты для «${query}»` : "Найдите нужное"}</h1><p>Пользователи, посты и видео в одном месте.</p></div>
        <div className="search-hero-mark" aria-hidden="true">⌕</div>
      </header>
      {error && <p className="error">{error}</p>}
      {loading && <p className="muted">Ищем...</p>}
      {!loading && !query && <div className="empty-state"><h2>Начните с запроса</h2><p>Введите имя, текст поста или название видео в поле поиска сверху.</p></div>}
      {!loading && query && total === 0 && <div className="empty-state"><h2>Ничего не нашли</h2><p>Попробуйте другое слово или более короткий запрос.</p></div>}
      {!loading && query && total > 0 && (
        <div className="search-results-grid">
          {result?.users.length ? <section className="panel search-result-section"><div className="section-title"><h2>Люди</h2><span>{result.users.length}</span></div><div className="search-user-list">{result.users.map((user) => <button type="button" className="search-user-result" key={user.id} onClick={() => navigate(`/profile/${user.id}`)}><span className="search-avatar">{user.avatar ? <img src={`${API_BASE_URL}${user.avatar}`} alt="" /> : user.full_name.slice(0, 1).toUpperCase()}</span><span><strong>{user.full_name}</strong><small>@{user.username}</small></span></button>)}</div></section> : null}
          {result?.videos.length ? <section className="panel search-result-section"><div className="section-title"><h2>Видео</h2><span>{result.videos.length}</span></div><div className="search-video-list">{result.videos.map((video) => <button type="button" className="search-video-result" key={video.video_id} onClick={() => video.owner_id && navigate(`/profile/${video.owner_id}/videos?q=${encodeURIComponent(video.title)}`)}><span className="search-video-icon">▶</span><span><strong>{video.title}</strong><small>{video.owner_name} · добавлено {formatDate(video.created_at)}</small></span></button>)}</div></section> : null}
          {result?.posts.length ? <section className="panel search-result-section"><div className="section-title"><h2>Посты</h2><span>{result.posts.length}</span></div><div className="search-post-list">{result.posts.map((post) => <article className="search-post-result" key={post.id}><div><strong>{post.author_name}</strong><small>{formatDate(post.created_at)}</small></div><p>{post.content}</p><button type="button" className="widget-link" onClick={() => navigate(`/profile/${post.author_id}`)}>Открыть профиль</button></article>)}</div></section> : null}
        </div>
      )}
    </section>
  );
}
