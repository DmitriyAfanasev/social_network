import { useEffect, useState } from "react";
import type { ReactNode } from "react";

import type {
  FriendActionResponse,
  FriendRecommendationsResponse,
  FriendRequestsResponse,
  FriendsResponse,
  User,
} from "../../entities/user/model/user";
import { getUserName } from "../../entities/user/model/user";
import { UserAvatar } from "../../entities/user/ui/UserAvatar";
import { apiRequest } from "../../shared/api/http";
import { navigate } from "../../shared/lib/navigation";

export function FriendsPage() {
  const [friends, setFriends] = useState<FriendsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [updatingId, setUpdatingId] = useState<string | number | null>(null);
  const [recommendations, setRecommendations] = useState<FriendRecommendationsResponse | null>(null);
  const [requests, setRequests] = useState<FriendRequestsResponse | null>(null);

  async function loadFriends() {
    setLoading(true);

    try {
      const [result, recommendationResult, requestResult] = await Promise.all([
        apiRequest<FriendsResponse>("/friends"),
        apiRequest<FriendRecommendationsResponse>("/friends/recommendations"),
        apiRequest<FriendRequestsResponse>("/friends/requests"),
      ]);
      setFriends(result);
      setRecommendations(recommendationResult);
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

  return (
    <section className="friends-page">
      <header className="page-header">
        <div>
          <span className="eyebrow">Мои друзья</span>
          <h1>Друзья</h1>
        </div>
        {friends?.current_user && <button onClick={() => navigate(`/profile/${friends.current_user?.id}`)}>Профиль</button>}
      </header>
      {error && <p className="error">{error}</p>}
      {loading && <p className="muted">Загружаем друзей...</p>}
      {!loading && recommendations && recommendations.recommendations.length > 0 && (
        <FriendsSection title="Возможно, вы знакомы" count={recommendations.recommendations.length}>
          {recommendations.recommendations.map(({ user, common_friends }) => (
            <FriendCard
              key={user.id}
              user={user}
              updating={updatingId === user.id}
              onMessage={(user) => navigate(`/messages?user=${user.id}`)}
              status={`${common_friends} общих друзей`}
              actions={<button type="button" onClick={() => navigate(`/profile/${user.id}`)}>Открыть профиль</button>}
            />
          ))}
        </FriendsSection>
      )}
      {!loading &&
        friends &&
        friends.friends.length === 0 &&
        friends.subscribers.length === 0 &&
        friends.subscriptions.length === 0 &&
        !requests?.incoming.length &&
        !requests?.outgoing.length && (
        <div className="empty-state">
          <h2>Список друзей пуст</h2>
          <p>Откройте профиль другого пользователя и добавьте его в друзья.</p>
        </div>
      )}
      {requests && requests.incoming.length > 0 && (
        <FriendsSection title="Входящие заявки" count={requests.incoming.length}>
          {requests.incoming.map((request) => {
            const user = requestUser(request.sender_id);
            return <FriendCard key={request.id} user={user} updating={updatingId === request.id} status="Хочет добавить вас в друзья" onMessage={(target) => navigate(`/messages?user=${target.id}`)} actions={<button type="button" disabled={updatingId === request.id} onClick={() => void acceptFriend(request.id)}>Принять</button>} />;
          })}
        </FriendsSection>
      )}
      {requests && requests.outgoing.length > 0 && (
        <FriendsSection title="Исходящие заявки" count={requests.outgoing.length}>
          {requests.outgoing.map((request) => <FriendCard key={request.id} user={requestUser(request.recipient_id)} updating={updatingId === request.id} status="Заявка отправлена" onMessage={(target) => navigate(`/messages?user=${target.id}`)} actions={<span className="muted">Ожидает ответа</span>} />)}
        </FriendsSection>
      )}
      {friends && friends.subscribers.length > 0 && (
        <FriendsSection title="Подписчики" count={friends.subscribers.length}>
          {friends.subscribers.map((subscriber) => (
            <FriendCard
              key={subscriber.id}
              user={subscriber}
              updating={updatingId === subscriber.id}
              onMessage={(user) => navigate(`/messages?user=${user.id}`)}
              actions={<button type="button" onClick={() => navigate(`/profile/${subscriber.id}`)}>Открыть профиль</button>}
            />
          ))}
        </FriendsSection>
      )}
      {friends && friends.friends.length > 0 && (
        <FriendsSection title="Друзья" count={friends.friends.length}>
          {friends.friends.map((friend) => (
            <FriendCard
              key={friend.id}
              user={friend}
              updating={updatingId === friend.id}
              onMessage={(user) => navigate(`/messages?user=${user.id}`)}
              actions={
                <button
                  type="button"
                  className="secondary"
                  disabled={updatingId === friend.id}
                  onClick={() => void removeFriend(friend.id)}
                >
                  {updatingId === friend.id ? "Удаляем..." : "Удалить"}
                </button>
              }
            />
          ))}
        </FriendsSection>
      )}
      {friends && friends.subscriptions.length > 0 && (
        <FriendsSection title="Исходящие заявки" count={friends.subscriptions.length}>
          {friends.subscriptions.map((subscription) => (
            <FriendCard
              key={subscription.id}
              user={subscription}
              updating={updatingId === subscription.id}
              status="Заявка отправлена"
              onMessage={(user) => navigate(`/messages?user=${user.id}`)}
              actions={
                <button
                  type="button"
                  className="secondary"
                  disabled={updatingId === subscription.id}
                  onClick={() => void cancelSubscription(subscription.id)}
                >
                  {updatingId === subscription.id ? "Отменяем..." : "Отменить"}
                </button>
              }
            />
          ))}
        </FriendsSection>
      )}
    </section>
  );
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
