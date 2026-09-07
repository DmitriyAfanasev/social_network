import { Children, useEffect, useState } from "react";
import type { ReactNode } from "react";

import type {
  FriendActionResponse,
  FriendRecommendationsResponse,
  FriendRequestsResponse,
  PublicProfileResponse,
  FriendsResponse,
  User,
} from "../../entities/user/model/user";
import { getUserName, userFromPublicProfile } from "../../entities/user/model/user";
import { UserAvatar } from "../../entities/user/ui/UserAvatar";
import { apiRequest } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";

type PresenceBatchResponse = {
  presence: Array<{ user_id: string; online: boolean }>;
};

async function loadFriendUser(id: string): Promise<User> {
  try {
    const profile = await apiRequest<PublicProfileResponse>(`/v1/profiles/${id}`);
    return userFromPublicProfile(profile);
  } catch {
    return requestUser(id);
  }
}

export function FriendsPage() {
  const [friends, setFriends] = useState<FriendsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [updatingId, setUpdatingId] = useState<string | number | null>(null);
  const [recommendations, setRecommendations] = useState<FriendRecommendationsResponse | null>(null);
  const [requests, setRequests] = useState<FriendRequestsResponse | null>(null);
  const [usersByID, setUsersByID] = useState<Record<string, User>>({});
  const [view, setView] = useState<FriendsView>("all");
  const [search, setSearch] = useState("");
  const [presenceByID, setPresenceByID] = useState<Record<string, boolean>>({});

  async function loadFriends() {
    setLoading(true);

    try {
      const [result, recommendationResult, requestResult] = await Promise.all([
        apiRequest<FriendsResponse>("/friends"),
        apiRequest<FriendRecommendationsResponse>("/friends/recommendations"),
        apiRequest<FriendRequestsResponse>("/friends/requests"),
      ]);
      const ids = new Set<string>();
      result.friends.forEach((user) => ids.add(String(user.id)));
      result.subscribers.forEach((user) => ids.add(String(user.id)));
      result.subscriptions.forEach((user) => ids.add(String(user.id)));
      recommendationResult.recommendations.forEach(({ user }) => ids.add(String(user.id)));
      requestResult.incoming.forEach((request) => ids.add(String(request.sender_id)));
      requestResult.outgoing.forEach((request) => ids.add(String(request.recipient_id)));
      const profileEntries = await Promise.all([...ids].map(async (id) => [id, await loadFriendUser(id)] as const));
      const resolvedUsers = Object.fromEntries(profileEntries) as Record<string, User>;
      setUsersByID(resolvedUsers);
      setFriends({
        ...result,
        friends: result.friends.map((user) => resolvedUsers[String(user.id)] ?? user),
        subscribers: result.subscribers.map((user) => resolvedUsers[String(user.id)] ?? user),
        subscriptions: result.subscriptions.map((user) => resolvedUsers[String(user.id)] ?? user),
      });
      setRecommendations({
        recommendations: recommendationResult.recommendations.map((item) => ({
          ...item,
          user: resolvedUsers[String(item.user.id)] ?? item.user,
        })),
      });
      setRequests(requestResult);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить друзей");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadFriends();
  }, []);

  const friendIDsKey = (friends?.friends ?? []).map((user) => String(user.id)).join(",");

  useEffect(() => {
    if (!friendIDsKey) {
      setPresenceByID({});
      return;
    }

    let active = true;
    const checkPresence = () => {
      void apiRequest<PresenceBatchResponse>(`/v1/messaging/presence?user_ids=${friendIDsKey}`)
        .then((result) => {
          if (active) setPresenceByID(Object.fromEntries(result.presence.map((item) => [item.user_id, item.online])));
        })
        .catch(() => {
          if (active) setPresenceByID({});
        });
    };

    checkPresence();
    const timer = window.setInterval(checkPresence, 15_000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [friendIDsKey]);

  async function removeFriend(friendId: string | number) {
    if (!window.confirm("Вы уверены, что хотите удалить пользователя из друзей?")) {
      return;
    }

    setUpdatingId(friendId);
    setError(null);

    try {
      await apiRequest<FriendActionResponse>(`/friends/${friendId}`, { method: "DELETE" });
      await loadFriends();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить друга");
    } finally {
      setUpdatingId(null);
    }
  }

  async function acceptFriend(requestId: string | number) {
    setUpdatingId(requestId);
    setError(null);

    try {
      await apiRequest<FriendActionResponse>(`/v1/social/friend-requests/${requestId}/accept`, { method: "POST" });
      await loadFriends();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось принять заявку");
    } finally {
      setUpdatingId(null);
    }
  }

  async function cancelSubscription(userId: string | number) {
    setUpdatingId(userId);
    setError(null);

    try {
      await apiRequest<FriendActionResponse>(`/friends/subscriptions/${userId}`, { method: "DELETE" });
      await loadFriends();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отменить заявку");
    } finally {
      setUpdatingId(null);
    }
  }

  const allFriends = friends?.friends ?? [];
  const onlineFriends = allFriends.filter((friend) => presenceByID[String(friend.id)] === true);
  const searchNeedle = search.trim().toLocaleLowerCase();
  const visibleFriends = (view === "online" ? onlineFriends : allFriends).filter((friend) => {
    if (!searchNeedle) return true;
    return `${getUserName(friend)} ${friend.username}`.toLocaleLowerCase().includes(searchNeedle);
  });
  const incomingRequests = requests?.incoming ?? [];
  const outgoingRequests = requests?.outgoing ?? [];
  const possibleFriends = recommendations?.recommendations ?? [];

  return (
    <section className="friends-page friends-page-vk">
      <main className="friends-main-column">
        <header className="friends-page-header">
          <div className="friends-tabs">
            <button type="button" className={view === "all" ? "friends-tab active" : "friends-tab"} onClick={() => setView("all")}>Все друзья <span>{allFriends.length}</span></button>
            <button type="button" className={view === "online" ? "friends-tab active" : "friends-tab"} onClick={() => setView("online")}>Друзья онлайн <span>{onlineFriends.length}</span></button>
          </div>
          <button type="button" className="friends-find-button" onClick={() => setView("recommendations")}>Найти друзей</button>
        </header>

        <label className="friends-search">
          <span aria-hidden="true">⌕</span>
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Введите запрос" />
          <span aria-hidden="true">☷</span>
        </label>

        {error && <p className="error">{error}</p>}
        {loading && <p className="muted">Загружаем друзей...</p>}

        {!loading && (view === "all" || view === "online") && (
          <section className="friends-results">
            {visibleFriends.length > 0 ? visibleFriends.map((friend) => (
              <FriendCard
                key={friend.id}
                user={friend}
                updating={updatingId === friend.id}
                onMessage={(user) => navigate(`/messages?user=${user.id}`)}
                actions={<button type="button" className="friend-more" aria-label={`Действия для ${getUserName(friend)}`}>•••</button>}
              />
            )) : <div className="friends-empty"><h2>{view === "online" ? "Сейчас никто не в сети" : "Список друзей пуст"}</h2><p>{view === "online" ? "Когда друзья появятся онлайн, они будут показаны здесь." : "Откройте профиль другого пользователя и добавьте его в друзья."}</p></div>}
          </section>
        )}

        {!loading && view === "incoming" && (
          <FriendsRequestList title="Заявки в друзья" empty="Новых заявок нет">
            {incomingRequests.map((request) => <FriendCard key={request.id} user={usersByID[request.sender_id] ?? requestUser(request.sender_id)} updating={updatingId === request.id} status="Хочет добавить вас в друзья" onMessage={(target) => navigate(`/messages?user=${target.id}`)} actions={<button type="button" disabled={updatingId === request.id} onClick={() => void acceptFriend(request.id)}>Принять</button>} />)}
          </FriendsRequestList>
        )}

        {!loading && view === "outgoing" && (
          <FriendsRequestList title="Исходящие заявки" empty="Исходящих заявок нет">
            {outgoingRequests.map((request) => <FriendCard key={request.id} user={usersByID[request.recipient_id] ?? requestUser(request.recipient_id)} updating={updatingId === request.id} status="Заявка отправлена" onMessage={(target) => navigate(`/messages?user=${target.id}`)} actions={<span className="muted">Ожидает ответа</span>} />)}
            {friends?.subscriptions.map((subscription) => <FriendCard key={subscription.id} user={subscription} updating={updatingId === subscription.id} status="Заявка отправлена" onMessage={(target) => navigate(`/messages?user=${target.id}`)} actions={<button type="button" className="secondary" disabled={updatingId === subscription.id} onClick={() => void cancelSubscription(subscription.id)}>{updatingId === subscription.id ? "Отменяем..." : "Отменить"}</button>} />)}
          </FriendsRequestList>
        )}

        {!loading && view === "subscribers" && (
          <FriendsRequestList title="Подписчики" empty="Подписчиков пока нет">
            {friends?.subscribers.map((subscriber) => <FriendCard key={subscriber.id} user={subscriber} updating={updatingId === subscriber.id} onMessage={(user) => navigate(`/messages?user=${user.id}`)} actions={<button type="button" onClick={() => navigate(`/profile/${subscriber.id}`)}>Открыть профиль</button>} />)}
          </FriendsRequestList>
        )}

        {!loading && view === "recommendations" && (
          <FriendsRequestList title="Возможно, вы знакомы" empty="Пока нет рекомендаций">
            {possibleFriends.map(({ user, common_friends }) => <FriendCard key={user.id} user={user} updating={updatingId === user.id} status={`${common_friends} общих друзей`} onMessage={(target) => navigate(`/messages?user=${target.id}`)} actions={<button type="button" onClick={() => navigate(`/profile/${user.id}`)}>Открыть профиль</button>} />)}
          </FriendsRequestList>
        )}
      </main>

      <aside className="friends-sidebar">
        <section className="friends-menu-card">
          <button type="button" className="friends-menu-title" onClick={() => setView("all")}>Мои друзья <span>⌄</span></button>
          <button type="button" className={view === "incoming" ? "friends-menu-link active" : "friends-menu-link"} onClick={() => setView("incoming")}>Заявки в друзья <b>{incomingRequests.length || ""}</b></button>
          <button type="button" className={view === "outgoing" ? "friends-menu-link active" : "friends-menu-link"} onClick={() => setView("outgoing")}>Исходящие заявки <b>{outgoingRequests.length || ""}</b></button>
          <button type="button" className={view === "recommendations" ? "friends-menu-link active" : "friends-menu-link"} onClick={() => setView("recommendations")}>Поиск друзей</button>
          {friends && friends.subscribers.length > 0 && <button type="button" className={view === "subscribers" ? "friends-menu-link active" : "friends-menu-link"} onClick={() => setView("subscribers")}>Подписчики <b>{friends.subscribers.length}</b></button>}
        </section>

        <section className="friends-suggestions-card">
          <div className="friends-suggestions-heading"><h2>Возможные друзья</h2><button type="button" onClick={() => setView("recommendations")}>Все</button></div>
          {possibleFriends.length > 0 ? possibleFriends.slice(0, 5).map(({ user, common_friends }) => (
            <button type="button" className="friend-suggestion" key={user.id} onClick={() => navigate(`/profile/${user.id}`)}>
              <UserAvatar user={user} size="md" />
              <span><strong>{getUserName(user)}</strong><small>{common_friends} общих друга</small></span>
              <b aria-hidden="true">＋</b>
            </button>
          )) : <p className="friends-sidebar-empty">Здесь появятся рекомендации</p>}
          {possibleFriends.length > 0 && <button type="button" className="friends-show-all" onClick={() => setView("recommendations")}>Показать всех</button>}
        </section>
      </aside>
    </section>
  );
}

type FriendsView = "all" | "online" | "incoming" | "outgoing" | "subscribers" | "recommendations";

type FriendsRequestListProps = {
  title: string;
  empty: string;
  children: ReactNode;
};

function FriendsRequestList({ title, empty, children }: FriendsRequestListProps) {
  const hasChildren = Children.count(children) > 0;
  return <section className="friends-results"><div className="friends-results-heading"><h2>{title}</h2></div>{hasChildren ? children : <div className="friends-empty"><h2>{empty}</h2></div>}</section>;
}

type FriendsSectionProps = {
  title: string;
  count: number;
  children: ReactNode;
};

function FriendsSection({ title, count, children }: FriendsSectionProps) {
  return (
    <section className="friends-section">
      <div className="section-title">
        <h2>{title}</h2>
        <span>{count}</span>
      </div>
      <div className="friends-list">{children}</div>
    </section>
  );
}

type FriendCardProps = {
  user: User;
  updating: boolean;
  status?: string;
  actions: ReactNode;
  onMessage: (user: User) => void;
};

function FriendCard({ user, updating, status, actions, onMessage }: FriendCardProps) {
  return (
    <article className="friend-card">
      <button type="button" className="friend-main" onClick={() => navigate(`/profile/${user.id}`)}>
        <UserAvatar user={user} size="lg" />
        <span>
          <strong>{getUserName(user)}</strong>
          <small>@{user.username}</small>
          {status && <em>{status}</em>}
        </span>
      </button>
      <div className="friend-card-actions">
        <button type="button" className="secondary" disabled={updating} onClick={() => onMessage(user)}>
          Написать
        </button>
        {actions}
      </div>
    </article>
  );
}

function requestUser(id: string): User {
  return { id, username: id.slice(0, 8), profile: null };
}
