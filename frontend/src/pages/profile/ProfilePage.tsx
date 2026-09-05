import { useEffect, useRef, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";

import type { LikeResponse, ProfileResponse } from "../../entities/post/model/post";
import { PostItem } from "../../entities/post/ui/PostItem";
import type { MusicResponse, MusicTrack } from "../../entities/music/model/music";
import type { AvatarHistoryResponse, AvatarUploadResponse } from "../../entities/profile/model/profile";
import type { FriendActionResponse } from "../../entities/user/model/user";
import { getUserName } from "../../entities/user/model/user";
import { UserAvatar } from "../../entities/user/ui/UserAvatar";
import { apiRequest, getAccessToken } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { formatDate } from "../../shared/lib/date";
import { navigate } from "../../shared/lib/navigation";
import { MediaPlayer } from "../../shared/ui/MediaPlayer";

type ProfilePageProps = {
  id: string;
};

type AvatarManagerTab = "upload" | "photos";

type RelationshipSnapshot = {
  friends: string[];
  subscribers: string[];
  subscriptions: string[];
};

type FriendRequestSnapshot = {
  incoming: Array<{ id: string; sender_id: string; recipient_id: string; status: string }>;
  outgoing: Array<{ id: string; sender_id: string; recipient_id: string; status: string }>;
};

export function ProfilePage({ id }: ProfilePageProps) {
  const [profile, setProfile] = useState<ProfileResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [friendUpdating, setFriendUpdating] = useState(false);
  const [incomingRequestId, setIncomingRequestId] = useState<string | null>(null);
  const [outgoingRequestId, setOutgoingRequestId] = useState<string | null>(null);
  const [avatarModalOpen, setAvatarModalOpen] = useState(false);
  const [avatarHistory, setAvatarHistory] = useState<AvatarHistoryResponse | null>(null);
  const [avatarLoading, setAvatarLoading] = useState(false);
  const [avatarUpdating, setAvatarUpdating] = useState(false);
  const [avatarError, setAvatarError] = useState<string | null>(null);
  const [avatarTab, setAvatarTab] = useState<AvatarManagerTab>("upload");
  const [pendingAvatarFile, setPendingAvatarFile] = useState<File | null>(null);
  const [pendingAvatarPreviewUrl, setPendingAvatarPreviewUrl] = useState<string | null>(null);
  const [bioExpanded, setBioExpanded] = useState(false);
  const [statusDraft, setStatusDraft] = useState("");
  const [statusEditing, setStatusEditing] = useState(false);
  const [statusSaving, setStatusSaving] = useState(false);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [musicTracks, setMusicTracks] = useState<MusicTrack[]>([]);
  const [musicError, setMusicError] = useState<string | null>(null);
  const avatarInputRef = useRef<HTMLInputElement | null>(null);

  async function loadProfile() {
    try {
      const result = await apiRequest<ProfileResponse>(`/profile/${id}`);
      let nextProfile = result;
      setIncomingRequestId(null);
      setOutgoingRequestId(null);
      if (!result.is_own_profile && getAccessToken()) {
        try {
          const [relationships, requests] = await Promise.all([
            apiRequest<RelationshipSnapshot>("/v1/social/relationships"),
            apiRequest<FriendRequestSnapshot>("/v1/social/friend-requests"),
          ]);
          const targetID = result.user.id;
          const incoming = requests.incoming.find((request) => request.sender_id === targetID && request.status === "pending");
          const outgoing = requests.outgoing.find((request) => request.recipient_id === targetID && request.status === "pending");
          setIncomingRequestId(incoming?.id ?? null);
          setOutgoingRequestId(outgoing?.id ?? null);
          nextProfile = {
            ...result,
            is_friend: relationships.friends.includes(targetID),
            is_subscribed: Boolean(outgoing),
            is_subscribed_to_current: Boolean(incoming),
          };
        } catch {
          // Публичный профиль остаётся доступным, даже если social временно недоступен.
        }
      }
      setProfile(nextProfile);
      setStatusDraft(nextProfile.user.profile?.status ?? "");
      void loadMusic(nextProfile.is_own_profile ? undefined : nextProfile.user.id);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить профиль");
    }
  }

  async function loadMusic(ownerID?: string) {
    try {
      const path = ownerID ? `/music?owner_id=${encodeURIComponent(ownerID)}` : "/music";
      const result = await apiRequest<MusicResponse>(path);
      setMusicTracks(result.tracks);
      setMusicError(null);
    } catch (err) {
      setMusicError(err instanceof Error ? err.message : "Не удалось загрузить музыку");
    }
  }

  useEffect(() => {
    void loadProfile();
  }, [id]);

  useEffect(() => {
    return () => {
      if (pendingAvatarPreviewUrl) {
        URL.revokeObjectURL(pendingAvatarPreviewUrl);
      }
    };
  }, [pendingAvatarPreviewUrl]);

  async function toggleFriend() {
    if (!profile || profile.is_own_profile) {
      return;
    }

    if (profile.is_friend && !window.confirm("Вы уверены, что хотите удалить пользователя из друзей?")) {
      return;
    }

    setFriendUpdating(true);
    setError(null);

    try {
      if (profile.is_friend) {
        await apiRequest<FriendActionResponse>(`/friends/${profile.user.id}`, { method: "DELETE" });
      } else if (incomingRequestId) {
        await apiRequest<FriendActionResponse>(`/v1/social/friend-requests/${incomingRequestId}/accept`, { method: "POST" });
      } else if (outgoingRequestId) {
        await apiRequest<FriendActionResponse>(`/v1/social/friend-requests/${outgoingRequestId}`, { method: "DELETE" });
      } else {
        await apiRequest<FriendActionResponse>(`/friends/${profile.user.id}`, { method: "POST" });
      }
      await loadProfile();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось обновить друзей");
    } finally {
      setFriendUpdating(false);
    }
  }

  async function toggleLike(post: ProfileResponse["posts"][number]) {
    await apiRequest<LikeResponse>(`/posts/${post.id}/likes`, { method: post.is_liked_by_current ? "DELETE" : "POST" });
    await loadProfile();
  }

  async function loadAvatarHistory() {
    setAvatarLoading(true);
    setAvatarError(null);

    try {
      const result = await apiRequest<AvatarHistoryResponse>("/profile/avatar/history");
      setAvatarHistory(result);
    } catch (err) {
      setAvatarError(err instanceof Error ? err.message : "Не удалось загрузить аватары");
    } finally {
      setAvatarLoading(false);
    }
  }

  async function openAvatarManager() {
    setAvatarModalOpen(true);
    setAvatarTab("upload");
    await loadAvatarHistory();
  }

  function closeAvatarManager() {
    clearPendingAvatar();
    setAvatarModalOpen(false);
  }

  function selectAvatarFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }

    if (pendingAvatarPreviewUrl) {
      URL.revokeObjectURL(pendingAvatarPreviewUrl);
    }

    setPendingAvatarFile(file);
    setPendingAvatarPreviewUrl(URL.createObjectURL(file));
    setAvatarError(null);
  }

  function clearPendingAvatar() {
    if (pendingAvatarPreviewUrl) {
      URL.revokeObjectURL(pendingAvatarPreviewUrl);
    }

    setPendingAvatarFile(null);
    setPendingAvatarPreviewUrl(null);
    if (avatarInputRef.current) {
      avatarInputRef.current.value = "";
    }
  }

  async function uploadAvatar() {
    if (!pendingAvatarFile) {
      setAvatarError("Сначала выберите фото.");
      return;
    }

    setAvatarUpdating(true);
    setAvatarError(null);

    try {
      const mediaBody = new FormData();
      mediaBody.append("file", pendingAvatarFile);
      const media = await apiRequest<{ id: string }>("/v1/media", { method: "POST", body: mediaBody });
      await apiRequest<AvatarUploadResponse>("/v1/profiles/me/avatar", {
        method: "PUT",
        body: JSON.stringify({ media_id: media.id }),
      });
      clearPendingAvatar();
      await Promise.all([loadProfile(), loadAvatarHistory()]);
      setAvatarTab("photos");
    } catch (err) {
      setAvatarError(err instanceof Error ? err.message : "Не удалось загрузить аватар");
    } finally {
      setAvatarUpdating(false);
    }
  }

  async function removeAvatar() {
    if (!window.confirm("Удалить текущую фотографию профиля?")) return;
    setAvatarUpdating(true);
    setAvatarError(null);
    try {
      clearPendingAvatar();
      await apiRequest("/profile/avatar", { method: "DELETE" });
      await Promise.all([loadProfile(), loadAvatarHistory()]);
    } catch (err) {
      setAvatarError(err instanceof Error ? err.message : "Не удалось удалить фото");
    } finally {
      setAvatarUpdating(false);
    }
  }

  async function selectAvatar(mediaId: string) {
    setAvatarUpdating(true);
    setAvatarError(null);

    try {
      await apiRequest<AvatarUploadResponse>("/v1/profiles/me/avatar/select", {
        method: "POST",
        body: JSON.stringify({ media_id: mediaId }),
      });
      await Promise.all([loadProfile(), loadAvatarHistory()]);
    } catch (err) {
      setAvatarError(err instanceof Error ? err.message : "Не удалось выбрать аватар");
    } finally {
      setAvatarUpdating(false);
    }
  }

  async function deleteMusic(trackId: string) {
    if (!window.confirm("Удалить этот трек из профиля?")) return;
    try {
      await apiRequest(`/music/${trackId}`, { method: "DELETE" });
      await loadMusic();
    } catch (err) {
      setMusicError(err instanceof Error ? err.message : "Не удалось удалить трек");
    }
  }

  async function saveStatus(event: FormEvent) {
    event.preventDefault();
    setStatusSaving(true);
    setStatusError(null);
    try {
      await apiRequest("/v1/profiles/me", {
        method: "PATCH",
        body: JSON.stringify({ status: statusDraft.trim() || null }),
      });
      await loadProfile();
      setStatusEditing(false);
    } catch (err) {
      setStatusError(err instanceof Error ? err.message : "Не удалось сохранить статус");
    } finally {
      setStatusSaving(false);
    }
  }

  function getFriendButtonText() {
    if (!profile) {
      return "";
    }

    if (friendUpdating) {
      return "Обновляем...";
    }

    if (profile.is_friend) {
      return "Убрать из друзей";
    }

    if (profile.is_subscribed_to_current) {
      return "Принять в друзья";
    }

    if (profile.is_subscribed) {
      return "Отменить заявку";
    }

    return "Добавить в друзья";
  }

  if (error) {
    return <p className="error">{error}</p>;
  }

  if (!profile) {
    return <p className="muted">Загружаем профиль...</p>;
  }

  const user = profile.user;
  const bio = user.profile?.bio ?? "";
  const hasLongBio = bio.length > 220;
  const details = [
    ["Имя", user.profile?.first_name],
    ["Фамилия", user.profile?.last_name],
    ["Отчество", user.profile?.middle_name],
    ["Дата рождения", user.profile?.birth_date],
    ["Пол", user.profile?.gender],
    ["Город", user.profile?.city],
    ["Страна", user.profile?.country],
    ["Улица", user.profile?.street],
    ["Телефон", user.profile?.phone_number],
  ].filter(([, value]) => value);
  const onlineFriends = profile.friends.filter((friend) => isRecentlyOnline(friend.last_seen_at));

  return (
    <section className="profile-page">
      <header className="panel profile-card profile-hero">
        <div className="profile-cover" aria-hidden="true"><span>GENERAL</span></div>
        <div className="profile-hero-body">
          <div className="profile-identity">
            {profile.is_own_profile ? (
              <button
                type="button"
                className="avatar-manage-button"
                onClick={() => void openAvatarManager()}
                aria-label="Изменить фотографию профиля"
              >
                <UserAvatar user={user} size="lg" />
                <span>Изменить</span>
              </button>
            ) : (
              <UserAvatar user={user} size="lg" />
            )}
            <div className="profile-identity-copy">
              <span className="eyebrow">@{user.username}</span>
              <h1>{getUserName(user)}</h1>
              {profile.is_own_profile ? (
                statusEditing ? (
                  <form className="profile-status-editor" onSubmit={saveStatus}>
                    <span className="presence-dot" aria-hidden="true" />
                    <input autoFocus value={statusDraft} onChange={(event) => setStatusDraft(event.target.value)} placeholder="Добавьте статус" maxLength={160} aria-label="Статус профиля" />
                    <button type="submit" disabled={statusSaving}>{statusSaving ? "..." : "Сохранить"}</button>
                    <button type="button" className="secondary" disabled={statusSaving} onClick={() => { setStatusDraft(user.profile?.status ?? ""); setStatusEditing(false); }}>Отмена</button>
                  </form>
                ) : (
                  <button type="button" className="profile-status-display" onClick={() => setStatusEditing(true)} aria-label="Редактировать статус">
                    <span className="presence-dot" aria-hidden="true" />
                    <span>{statusDraft || "Добавьте статус"}</span>
                    <span className="profile-status-edit-icon" aria-hidden="true">✎</span>
                  </button>
                )
              ) : user.profile?.status ? (
                <div className="profile-status-visible"><span className="presence-dot" />{user.profile.status}</div>
              ) : (
                <div className="profile-presence"><span className="presence-dot" />Публичный профиль</div>
              )}
              {statusError && profile.is_own_profile && <small className="profile-status-error">{statusError}</small>}
              <p className={hasLongBio && !bioExpanded ? "profile-bio profile-bio-collapsed" : "profile-bio"}>{bio || "Пользователь пока не заполнил описание."}</p>
              {hasLongBio && <button type="button" className="profile-more-link" onClick={() => setBioExpanded((expanded) => !expanded)}>{bioExpanded ? "Скрыть" : "Подробнее"}</button>}
            </div>
          </div>
          <div className="profile-actions">
          {profile.is_own_profile ? (
            <>
              <button onClick={() => navigate(`/profile/${id}/edit`)}>Редактировать</button>
            </>
          ) : (
            <>
              {(profile.can_send_friend_request !== false || profile.is_friend || profile.is_subscribed || profile.is_subscribed_to_current) && (
                <button
                  className={profile.is_friend || profile.is_subscribed ? "secondary" : undefined}
                  disabled={friendUpdating}
                  onClick={() => void toggleFriend()}
                >
                  {getFriendButtonText()}
                </button>
              )}
              {profile.can_send_message !== false && (
                <button type="button" className="secondary" onClick={() => navigate(`/messages?user=${user.id}`)}>
                  Написать
                </button>
              )}
            </>
          )}
          </div>
        </div>
        <nav className="profile-tabs" aria-label="Разделы профиля">
          <button type="button" className="profile-tab active">Посты <span>{profile.posts.length}</span></button>
          <button type="button" className="profile-tab" onClick={() => navigate(`/profile/${id}/photos`)}>Фотографии</button>
          <button type="button" className="profile-tab" onClick={() => navigate(`/profile/${id}/videos`)}>Видео</button>
        </nav>
      </header>
      {avatarModalOpen && profile.is_own_profile && (
        <div className="mock-dialog" role="dialog" aria-modal="true">
          <div className="avatar-dialog-panel">
            <header>
              <div>
                <span className="eyebrow">Аватар</span>
                <h2>Фото профиля</h2>
              </div>
              <button type="button" className="secondary" onClick={closeAvatarManager}>
                Закрыть
              </button>
            </header>
            <div className="avatar-tabs" role="tablist">
              <button
                type="button"
                className={avatarTab === "upload" ? "active" : undefined}
                onClick={() => setAvatarTab("upload")}
              >
                Загрузка
              </button>
              <button
                type="button"
                className={avatarTab === "photos" ? "active" : undefined}
                onClick={() => setAvatarTab("photos")}
              >
                Фотографии
              </button>
            </div>
            {avatarTab === "upload" && (
              <>
                <div className="avatar-current">
                  <UserAvatar user={profile.user} size="lg" />
                  <div>
                    <strong>{getUserName(profile.user)}</strong>
                    <span>Текущая фотография профиля</span>
                  </div>
                  <button type="button" className="avatar-remove-button" disabled={avatarUpdating} onClick={() => void removeAvatar()}>
                    Удалить фото
                  </button>
                </div>
                <div className={pendingAvatarPreviewUrl ? "avatar-upload-preview has-preview" : "avatar-upload-preview"}>
                  {pendingAvatarPreviewUrl ? (
                    <img src={pendingAvatarPreviewUrl} alt="" />
                  ) : (
                    <span>Выберите фото, чтобы увидеть превью перед загрузкой.</span>
                  )}
                  <div>
                    <strong>{pendingAvatarFile?.name ?? "Новое фото"}</strong>
                    <p>
                      Файл сначала показывается только у вас в браузере. На сервер он попадёт после подтверждения.
                    </p>
                  </div>
                </div>
                <div className="avatar-actions">
                  <label className="avatar-file-picker">
                    <span className="avatar-file-picker-icon" aria-hidden="true"><svg viewBox="0 0 24 24" focusable="false"><path d="M4 7.5h3l1.4-2h7.2l1.4 2H20a1.5 1.5 0 0 1 1.5 1.5v9A1.5 1.5 0 0 1 20 19.5H4A1.5 1.5 0 0 1 2.5 18V9A1.5 1.5 0 0 1 4 7.5Z" /><circle cx="12" cy="13.5" r="3.2" /><path d="M12 2.5v5M9.5 5l2.5-2.5L14.5 5" /></svg></span>
                    <span className="avatar-file-picker-copy">
                      <strong>{pendingAvatarFile ? "Выбрать другое фото" : "Выбрать фото"}</strong>
                      <small>{pendingAvatarFile ? pendingAvatarFile.name : "JPG, PNG или GIF · до 10 МБ"}</small>
                    </span>
                    <input
                      ref={avatarInputRef}
                      type="file"
                      accept="image/*"
                      disabled={avatarUpdating}
                      onChange={selectAvatarFile}
                    />
                  </label>
                  <button type="button" disabled={avatarUpdating || !pendingAvatarFile} onClick={() => void uploadAvatar()}>
                    {avatarUpdating ? "Загружаем..." : "Подтвердить"}
                  </button>
                  {pendingAvatarFile && (
                    <button type="button" className="secondary" disabled={avatarUpdating} onClick={clearPendingAvatar}>
                      Отмена
                    </button>
                  )}
                </div>
              </>
            )}
            {avatarError && <p className="error">{avatarError}</p>}
            {avatarTab === "photos" && (
              <section className="avatar-history">
                <div className="section-title compact">
                  <h3>Альбом профиля</h3>
                  <span>{avatarHistory?.avatars.length ?? 0}</span>
                </div>
                <div className="avatar-album-title">
                  <strong>Архив аватаров</strong>
                  <p>Все фото, которые раньше использовались как аватар профиля.</p>
                </div>
                {avatarLoading && <p className="muted">Загружаем историю...</p>}
                {!avatarLoading && avatarHistory?.avatars.length === 0 && (
                  <p className="muted">Альбом появится после первой подтверждённой загрузки фото.</p>
                )}
                <div className="avatar-history-grid">
                  {avatarHistory?.avatars.map((avatar) => (
                    <button
                      key={avatar.avatar_url}
                      type="button"
                      className={avatar.is_current ? "avatar-history-item active" : "avatar-history-item"}
                      disabled={avatarUpdating || avatar.is_current}
                      onClick={() => void selectAvatar(avatar.media_id ?? avatar.avatar_url)}
                    >
                      <img src={`${API_BASE_URL}${avatar.avatar_url}`} alt="" />
                      <span>{avatar.is_current ? "Текущий" : formatDate(avatar.created_at)}</span>
                    </button>
                  ))}
                </div>
              </section>
            )}
          </div>
        </div>
      )}
      <div className="profile-content-grid">
        <div className="profile-main-column">
        {details.length > 0 && (
          <dl className="panel detail-grid">
            {details.map(([label, value]) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
        )}
      <section className="profile-posts-section">
        <div className="section-title">
          <h2>Посты</h2>
          <span>{profile.posts.length}</span>
        </div>
        <div className="post-list">
          {profile.posts.map((post) => (
            <PostItem
              key={post.id}
              post={post}
              canLike={Boolean(profile.current_user)}
              currentUserId={profile.current_user.id}
              onLike={() => void toggleLike(post)}
              onChanged={loadProfile}
            />
          ))}
        </div>
      </section>
        </div>
        <aside className="profile-sidebar-widgets">
          <section className="panel profile-widget friends-widget">
            <header className="widget-header">
              <div><span className="eyebrow">Сейчас</span><h2>Друзья онлайн <span>{onlineFriends.length}</span></h2></div>
              <button type="button" className="widget-link" onClick={() => navigate("/friends")}>Все</button>
            </header>
            {onlineFriends.length > 0 ? (
              <div className="online-friends-grid">
                {onlineFriends.slice(0, 6).map((friend) => (
                  <button type="button" className="friend-widget-person" key={friend.id} onClick={() => navigate(`/profile/${friend.id}`)}>
                    <span className="friend-avatar-wrap"><UserAvatar user={friend} size="md" /><span className="online-dot" /></span>
                    <strong>{getUserName(friend)}</strong>
                  </button>
                ))}
              </div>
            ) : <p className="widget-empty">Сейчас никто не в сети</p>}
          </section>
          <section className="panel profile-widget friends-widget">
            <header className="widget-header">
              <div><span className="eyebrow">Ваш круг</span><h2>Друзья <span>{profile.friends.length}</span></h2></div>
              <button type="button" className="widget-link" onClick={() => navigate("/friends")}>Открыть</button>
            </header>
            {profile.friends.length > 0 ? (
              <div className="friends-mini-grid">
                {profile.friends.slice(0, 8).map((friend) => (
                  <button type="button" className="friend-widget-person" key={friend.id} onClick={() => navigate(`/profile/${friend.id}`)}>
                    <span className="friend-avatar-wrap"><UserAvatar user={friend} size="md" />{isRecentlyOnline(friend.last_seen_at) && <span className="online-dot" />}</span>
                    <strong>{getUserName(friend)}</strong>
                  </button>
                ))}
              </div>
            ) : <p className="widget-empty">Здесь появятся ваши друзья</p>}
          </section>
          <section className="panel profile-widget music-widget">
            <header className="widget-header">
              <div><span className="eyebrow">Аудиотека</span><h2>Музыка <span>{musicTracks.length}</span></h2></div>
              {profile.is_own_profile && <div className="music-widget-actions"><button type="button" className="widget-link" onClick={() => navigate("/music/add")}>Добавить</button><button type="button" className="widget-link" onClick={() => navigate("/music")}>Открыть</button></div>}
            </header>
            {musicTracks.length > 0 ? (
              <div className="music-track-list">
                {musicTracks.map((track) => (
                  <article className="music-track" key={track.id}>
                    <div className="music-track-icon" aria-hidden="true">♫</div>
                    <div className="music-track-copy">
                      <strong>{track.title}</strong>
                      <span>{track.artist || "Исполнитель не указан"}</span>
                      <small>{formatDate(track.created_at)}</small>
                    </div>
                    <MediaPlayer kind="audio" src={`${API_BASE_URL}/v1/media/${track.media_id}/content`} title={`${track.title} — ${track.artist || "трек"}`} preload="none" />
                    {profile.is_own_profile && (
                      <button type="button" className="music-delete" onClick={() => void deleteMusic(track.id)} aria-label={`Удалить ${track.title}`}>
                        ×
                      </button>
                    )}
                  </article>
                ))}
              </div>
            ) : (
              <p className="widget-empty">{musicError ? (profile.is_own_profile ? musicError : "Музыка скрыта настройками приватности.") : profile.is_own_profile ? "Добавьте первый трек в свою аудиотеку." : "В аудиотеке пока пусто."}</p>
            )}
            {musicError && profile.is_own_profile && <p className="error">{musicError}</p>}
          </section>
        </aside>
      </div>
    </section>
  );
}

function isRecentlyOnline(lastSeenAt: string | null | undefined): boolean {
  if (!lastSeenAt) return false;
  const elapsed = Date.now() - new Date(lastSeenAt).getTime();
  return elapsed >= 0 && elapsed <= 5 * 60 * 1000;
}
