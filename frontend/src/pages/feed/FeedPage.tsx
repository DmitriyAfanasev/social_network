import { useEffect, useState } from "react";

import type { FeedResponse, LikeResponse, Post } from "../../entities/post/model/post";
import { PostItem } from "../../entities/post/ui/PostItem";
import { getUserName, userFromPublicProfile, type PublicProfileResponse, type User } from "../../entities/user/model/user";
import { CreatePostPanel } from "../../features/post/create/ui/CreatePostPanel";
import { apiRequest } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";

export function FeedPage() {
  const [feed, setFeed] = useState<FeedResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  async function loadFeed() {
    setLoading(true);

    try {
      const [result, profile] = await Promise.all([
        apiRequest<FeedResponse>("/posts"),
        apiRequest<PublicProfileResponse>("/v1/profiles/me").catch(() => null),
      ]);
      const currentUser: User | null = profile ? userFromPublicProfile(profile) : null;
      const posts = await attachPostAuthors(result.posts);
      setFeed({ ...result, posts, current_user: currentUser });
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить ленту");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadFeed();
  }, []);

  async function toggleLike(post: Post) {
    await apiRequest<LikeResponse>(`/posts/${post.id}/likes`, { method: post.is_liked_by_current ? "DELETE" : "POST" });
    await loadFeed();
  }

  return (
    <section className="page-grid">
      <div className="feed-column">
        <header className="page-header">
          <div>
            <span className="eyebrow">Лента</span>
            <h1>Последние посты</h1>
          </div>
          {feed?.current_user && <button onClick={() => navigate(`/profile/${feed.current_user?.id}`)}>Профиль</button>}
        </header>
        {feed?.current_user && <CreatePostPanel onCreated={loadFeed} />}
        {error && <p className="error">{error}</p>}
        {loading && <p className="muted">Загружаем посты...</p>}
        {!loading && feed?.posts.length === 0 && (
          <div className="empty-state">
            <h2>Пока нет постов</h2>
            <p>Создайте первый пост после входа в аккаунт.</p>
          </div>
        )}
        <div className="post-list">
          {feed?.posts.map((post) => (
            <PostItem
              key={post.id}
              post={post}
              canLike={Boolean(feed.current_user)}
              currentUserId={feed.current_user?.id}
              onLike={() => void toggleLike(post)}
              onChanged={loadFeed}
            />
          ))}
        </div>
      </div>
      <aside className="side-panel">
        <span className="eyebrow">Твоё пространство</span>
        <h2>Лента, которая начинается с тебя</h2>
        <p className="side-panel-note">Публикуй мысли, сохраняй моменты и находи людей, с которыми хочется быть на связи.</p>
        <dl>
          <div>
            <dt>Ты вошёл как</dt>
            <dd>{feed?.current_user ? getUserName(feed.current_user) : "Гость"}</dd>
          </div>
          <div>
            <dt>В ленте сейчас</dt>
            <dd>{feed?.posts.length ?? 0} {feed?.posts.length === 1 ? "пост" : "постов"}</dd>
          </div>
        </dl>
      </aside>
    </section>
  );
}

async function attachPostAuthors(posts: FeedResponse["posts"]) {
  const authorIDs = [...new Set(posts.map((post) => String(post.author_id)))];
  const profiles = await Promise.all(authorIDs.map(async (authorID) => {
    try {
      return await apiRequest<PublicProfileResponse>(`/v1/profiles/${authorID}`);
    } catch {
      return null;
    }
  }));
  const authors = new Map(
    profiles
      .filter((profile): profile is PublicProfileResponse => profile !== null)
      .map((profile) => [profile.user_id, userFromPublicProfile(profile)]),
  );
  return posts.map((post) => ({ ...post, author: authors.get(String(post.author_id)) ?? post.author }));
}
