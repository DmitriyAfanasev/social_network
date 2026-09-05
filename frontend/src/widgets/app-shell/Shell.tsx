import { useEffect, useState } from "react";
import type { FormEvent, ReactNode } from "react";

import type { User } from "../../entities/user/model/user";
import { getUserName } from "../../entities/user/model/user";
import { UserAvatar } from "../../entities/user/ui/UserAvatar";
import { apiRequest, clearAuthTokens, getAccessToken, getRefreshToken } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { navigate } from "../../shared/lib/navigation";
import { SidebarLink } from "./SidebarLink";

type ShellProps = {
  children: ReactNode;
  path: string;
};

type MessageNotification = {
  type?: string;
  message?: { conversation_id: string; sender_id: string; body: string; media_id: string | null };
};

type FriendNotification = {
  type: "friend.requested" | "friend.accepted";
  actor_id: string;
  message: string;
};

type Toast = {
  kind: "message" | "friend";
  sender: User | null;
  title: string;
  body: string;
  target: string;
};

export function Shell({ children, path }: ShellProps) {
  const [viewer, setViewer] = useState<User | null>(null);
  const [unreadMessages, setUnreadMessages] = useState(0);
  const [unreadNotifications, setUnreadNotifications] = useState(0);
  const [toast, setToast] = useState<Toast | null>(null);
  const [searchDraft, setSearchDraft] = useState(() => new URLSearchParams(window.location.search).get("q") ?? "");

  useEffect(() => {
    if (path.startsWith("/search")) {
      setSearchDraft(new URLSearchParams(path.split("?")[1] ?? "").get("q") ?? "");
    }
  }, [path]);

  function submitSearch(event: FormEvent) {
    event.preventDefault();
    const value = searchDraft.trim();
    navigate(value ? `/search?q=${encodeURIComponent(value)}` : "/search");
  }

  async function logout() {
    const refreshToken = getRefreshToken();
    try {
      if (refreshToken) {
        await apiRequest<void>("/logout", {
          method: "POST",
          body: JSON.stringify({ refresh_token: refreshToken }),
          skipAuthRefresh: true,
        });
      }
    } finally {
      clearAuthTokens();
      setViewer(null);
      navigate("/login");
    }
  }

  useEffect(() => {
    let ignore = false;

    apiRequest<ProfileDTO>("/v1/profiles/me")
      .then((result) => {
        if (!ignore) {
          setViewer(profileToUser(result));
        }
      })
      .catch(() => {
        if (!ignore) {
          setViewer(null);
        }
      });

    return () => {
      ignore = true;
    };
  }, [path]);

  useEffect(() => {
    if (!viewer) return;
    const token = getAccessToken();
    if (!token) return;
    const stream = new EventSource(`${API_BASE_URL}/v1/notifications/stream?access_token=${encodeURIComponent(token)}`);
    let disposed = false;

    stream.addEventListener("message.new", (event) => {
      const notification = JSON.parse((event as MessageEvent<string>).data) as MessageNotification;
      const message = notification.message;
      if (!message || message.sender_id === viewer.id || disposed) return;
      setUnreadMessages((count) => count + 1);
      setToast({
        kind: "message",
        sender: null,
        title: "Новое сообщение",
      body: message.body || "Новое сообщение с вложением",
        target: `/messages?user=${message.sender_id}`,
      });
      void apiRequest<ProfileDTO>(`/profile/${message.sender_id}`)
        .then((profile) => {
          if (!disposed) {
            setToast((current) => current?.kind === "message" && current.target === `/messages?user=${message.sender_id}` ? { ...current, sender: profileToUser(profile) } : current);
          }
        })
        .catch(() => undefined);
    });

    const handleFriendNotification = (event: Event) => {
      const notification = JSON.parse((event as MessageEvent<string>).data) as FriendNotification;
      if (disposed) return;
      setUnreadNotifications((count) => count + 1);
      setToast({
        kind: "friend",
        sender: null,
        title: notification.type === "friend.accepted" ? "Заявку приняли" : "Новая заявка в друзья",
        body: notification.message,
        target: "/friends",
      });
      void apiRequest<ProfileDTO>(`/profile/${notification.actor_id}`)
        .then((profile) => {
          if (!disposed) setToast((current) => current?.kind === "friend" ? { ...current, sender: profileToUser(profile) } : current);
        })
        .catch(() => undefined);
    };

    stream.addEventListener("friend.requested", handleFriendNotification);
    stream.addEventListener("friend.accepted", handleFriendNotification);

    return () => {
      disposed = true;
      stream.close();
    };
  }, [viewer]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(null), 6500);
    return () => window.clearTimeout(timer);
  }, [toast]);

  useEffect(() => {
    if (path === "/messages" || path.startsWith("/messages/")) setUnreadMessages(0);
    if (path === "/notifications" || path === "/friends") setUnreadNotifications(0);
  }, [path]);

  const profileHref = viewer ? "/profile/me" : "/login";

  return (
    <div className="app-shell">
      <header className="topbar">
        <button className="brand-button topbar-brand" onClick={() => navigate("/")}>
          <span className="brand-mark">G</span>
          <span>General</span>
        </button>
        <form className="search-box" aria-label="Поиск" onSubmit={submitSearch}>
          <span>Поиск</span>
          <input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="Люди, посты, видео" />
        </form>
        <div className="topbar-actions">
          {viewer ? (
            <button className="user-chip" onClick={() => navigate(profileHref)}>
              <UserAvatar user={viewer} size="sm" />
              <span>{getUserName(viewer)}</span>
            </button>
          ) : (
            <>
              <button className="secondary" onClick={() => navigate("/login")}>
                Войти
              </button>
              <button onClick={() => navigate("/register")}>Регистрация</button>
            </>
          )}
        </div>
      </header>

      <div className="app-frame">
        <aside className="sidebar">
          <button className="brand-button" onClick={() => navigate("/")}>
            <span className="brand-mark">G</span>
            <span>General</span>
          </button>
          <nav className="side-nav">
            <SidebarLink href="/" label="Лента" path={path} />
            {viewer && (
              <>
                <SidebarLink href={profileHref} label="Профиль" path={path} />
                <SidebarLink href="/friends" label="Мои друзья" path={path} />
                <SidebarLink href="/photos" label="Фотографии" path={path} />
                <SidebarLink href={viewer ? `/profile/${viewer.id}/videos` : "/login"} label="Видео" path={path} />
                <SidebarLink href="/music" label="Музыка" path={path} />
                <SidebarLink href="/messages" label="Сообщения" path={path} badge={unreadMessages > 0 ? String(unreadMessages) : undefined} />
                <SidebarLink href="/notifications" label="Уведомления" path={path} badge={unreadNotifications > 0 ? String(unreadNotifications) : undefined} />
                <SidebarLink href="/settings" label="Настройки" path={path} badge="soon" />
                <button type="button" className="nav-link" onClick={() => void logout()}><span className="nav-dot" /><span>Выйти</span></button>
              </>
            )}
          </nav>
          <div className="sidebar-card">
            <span className="eyebrow">Небольшое напоминание</span>
            <strong>Здесь можно поделиться тем, что сегодня не хочется оставлять только у себя.</strong>
          </div>
        </aside>

        <main className="content">{children}</main>
      </div>

      <footer className="footer">
        <span>General</span>
        <span>Место для настоящих историй</span>
      </footer>
      {toast && (
        <button
          type="button"
          className="message-notification-toast"
          onClick={() => {
            if (toast.kind === "message") setUnreadMessages(0);
            if (toast.kind === "friend") setUnreadNotifications(0);
            setToast(null);
            navigate(toast.target);
          }}
          aria-label={toast.title}
        >
          <UserAvatar user={toast.sender} size="md" />
          <span>
            <strong>{getUserName(toast.sender) || toast.title}</strong>
            <small>{toast.body}</small>
          </span>
          <span className="message-notification-close" onClick={(event) => { event.stopPropagation(); setToast(null); }} aria-hidden="true">×</span>
        </button>
      )}
    </div>
  );
}

type ProfileDTO = {
  user_id: string;
  handle?: string | null;
  display_name: string;
  bio: string;
  avatar_url: string;
  created_at: string;
  updated_at: string;
};

function profileToUser(profile: ProfileDTO): User {
  return {
    id: profile.user_id,
    username: profile.handle ?? "",
    profile: {
      first_name: null,
      last_name: null,
      middle_name: null,
      full_name: profile.display_name || profile.handle || null,
      birth_date: null,
      gender: null,
      phone_number: null,
      country: null,
      city: null,
      street: null,
      avatar: profile.avatar_url || null,
      bio: profile.bio,
      status: null,
    },
  };
}
