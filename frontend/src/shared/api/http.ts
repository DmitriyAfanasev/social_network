import { API_BASE_URL } from "../config/api";

type ApiError = {
	error?: {
		message?: string;
	};
	detail?: string;
};

export type AuthTokens = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  refresh_expires_in: number;
};

const ACCESS_TOKEN_KEY = "general.access_token";
const REFRESH_TOKEN_KEY = "general.refresh_token";

type ApiRequestInit = RequestInit & {
  skipAuthRefresh?: boolean;
};

let refreshRequest: Promise<void> | null = null;
const inFlightGetRequests = new Map<string, Promise<unknown>>();
const authRefreshExcludedPaths = new Set([
	"/v1/auth/login",
	"/v1/auth/refresh",
	"/v1/auth/logout",
	"/v1/auth/register",
	"/v1/auth/registration-confirmation-requests",
	"/v1/auth/confirm-registration",
	"/v1/auth/password-reset-requests",
	"/v1/auth/password-reset",
]);

/** Возвращает access-токен текущей сессии фронтенда. */
export function getAccessToken() {
	return window.localStorage.getItem(ACCESS_TOKEN_KEY);
}

/** Возвращает refresh-токен текущей сессии для завершения работы. */
export function getRefreshToken() {
	return window.localStorage.getItem(REFRESH_TOKEN_KEY);
}

/** Сохраняет пару токенов, полученную от identity-сервиса. */
export function saveAuthTokens(tokens: AuthTokens) {
	window.localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
	window.localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
}

/** Удаляет локальные данные сессии пользователя. */
export function clearAuthTokens() {
	window.localStorage.removeItem(ACCESS_TOKEN_KEY);
	window.localStorage.removeItem(REFRESH_TOKEN_KEY);
}

export async function apiRequest<T>(path: string, init?: ApiRequestInit): Promise<T> {
  const method = (init?.method ?? "GET").toUpperCase();
  if (method === "GET") {
    const existing = inFlightGetRequests.get(path);
    if (existing) {
      return existing as Promise<T>;
    }

    const request = requestApi<T>(path, init);
    inFlightGetRequests.set(path, request);
    try {
      return await request;
    } finally {
      if (inFlightGetRequests.get(path) === request) {
        inFlightGetRequests.delete(path);
      }
    }
  }

  return requestApi<T>(path, init);
}

/** Загружает приватное медиа с токеном в заголовке и обновлением истёкшей сессии. */
export async function apiBlob(path: string, signal?: AbortSignal): Promise<Blob> {
	let response = await sendRequest(path, { signal, cache: "no-store" });
	if (response.status === 401) {
		await refreshAccessToken();
		response = await sendRequest(path, { signal, cache: "no-store" });
	}
	if (!response.ok) { await parseResponse(response); throw new Error("Не удалось загрузить фотографию"); }
	return response.blob();
}

async function requestApi<T>(path: string, init?: ApiRequestInit): Promise<T> {
  const gatewayPath = toGatewayPath(path, init?.method);
  const gatewayInit = adaptRequestInit(gatewayPath, init);
  const response = await sendRequest(gatewayPath, gatewayInit);

	if (response.status === 401 && shouldTryRefresh(gatewayPath, init)) {
		await refreshAccessToken();
		return adaptResponse<T>(path, await parseResponse<unknown>(await sendRequest(gatewayPath, gatewayInit)));
	}

	return adaptResponse<T>(path, await parseResponse<unknown>(response));
}

function adaptRequestInit(path: string, init?: ApiRequestInit): ApiRequestInit | undefined {
	if (!init?.method) return init;
	if (path.startsWith("/v1/social/friendships/") && init.method.toUpperCase() === "POST") {
		return { ...init, method: "PUT" };
	}
	return init;
}

function shouldTryRefresh(path: string, init?: ApiRequestInit) {
	return !init?.skipAuthRefresh && !authRefreshExcludedPaths.has(path);
}

async function refreshAccessToken() {
	const refreshToken = window.localStorage.getItem(REFRESH_TOKEN_KEY);
	if (!refreshToken) {
		clearAuthTokens();
		throw new Error("Сессия истекла. Войдите снова.");
	}

	refreshRequest ??= sendRequest("/v1/auth/refresh", {
		method: "POST",
		body: JSON.stringify({ refresh_token: refreshToken }),
		skipAuthRefresh: true,
	})
		.then(async (response) => {
			const tokens = await parseResponse<AuthTokens>(response);
			saveAuthTokens(tokens);
		})
		.catch((error) => {
			clearAuthTokens();
			throw error;
		})
    .finally(() => {
      refreshRequest = null;
    });

  return refreshRequest;
}

async function sendRequest(path: string, init?: ApiRequestInit): Promise<Response> {
  const { skipAuthRefresh: _skipAuthRefresh, ...requestInit } = init ?? {};
	const headers = new Headers(init?.headers);
	const accessToken = getAccessToken();
	if (accessToken && !headers.has("Authorization")) {
		headers.set("Authorization", `Bearer ${accessToken}`);
	}

	if (!(init?.body instanceof FormData) && !headers.has("Content-Type")) {
		headers.set("Content-Type", "application/json");
	}

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      credentials: "include",
      ...requestInit,
      headers,
    });
  } catch (error) {
    // Safari/Chrome often reduce a refused connection, wrong bind address or
    // CORS/network failure to the opaque message "Load failed". Include the
    // resolved API URL so LAN debugging does not require guessing whether the
    // frontend still points to localhost or to the computer's Wi-Fi address.
    const reason = error instanceof Error && error.message ? `: ${error.message}` : "";
    throw new Error(`Не удалось подключиться к API ${API_BASE_URL}${reason}`);
  }
}

function toGatewayPath(path: string, method = "GET") {
	if (path.startsWith("/v1/")) return path;
	const [pathname, query = ""] = path.split("?", 2);
	const suffix = query ? `?${query}` : "";

	const exact: Record<string, string> = {
		"/login": "/v1/auth/login",
		"/register": "/v1/auth/register",
		"/refresh": "/v1/auth/refresh",
		"/logout": "/v1/auth/logout",
		"/registration-confirmations/confirm": "/v1/auth/confirm-registration",
		"/password-reset-requests": "/v1/auth/password-reset-requests",
		"/password-resets": "/v1/auth/password-reset",
		"/friends": "/v1/social/relationships",
		"/friends/recommendations": "/v1/social/recommendations",
		"/search": "/v1/profiles/search",
		"/music": "/v1/media/music",
		"/posts": method === "GET" ? "/v1/content/feed" : "/v1/content/posts",
		"/notifications/stream": "/v1/notifications/stream",
	};
	if (exact[pathname]) return `${exact[pathname]}${suffix}`;

	let match = pathname.match(/^\/posts\/([^/]+)\/likes$/);
	if (match) return `/v1/content/posts/${match[1]}/like${suffix}`;
	match = pathname.match(/^\/posts\/([^/]+)\/comments$/);
	if (match) return `/v1/content/posts/${match[1]}/comments${suffix}`;
	match = pathname.match(/^\/posts\/([^/]+)$/);
	if (match) return `/v1/content/posts/${match[1]}${suffix}`;
	match = pathname.match(/^\/comments\/([^/]+)$/);
	if (match) return `/v1/content/comments/${match[1]}${suffix}`;
	match = pathname.match(/^\/profile\/([^/]+)\/photos$/);
	if (match) return `/v1/profiles/${match[1]}/photos${suffix}`;
	match = pathname.match(/^\/profile\/([^/]+)\/videos$/);
	if (match) return `/v1/media/videos?owner_id=${encodeURIComponent(match[1])}${suffix ? `&${suffix.slice(1)}` : ""}`;
	match = pathname.match(/^\/profile\/([^/]+)$/);
	if (match) return `/v1/profiles/${match[1]}${suffix}`;
	if (pathname === "/profile/avatar/history") return `/v1/profiles/me/avatar/history${suffix}`;
	if (pathname === "/profile/avatar") return `/v1/profiles/me/avatar${suffix}`;
	if (pathname === "/profile/avatar/select") return `/v1/profiles/me/avatar/select${suffix}`;
	match = pathname.match(/^\/profile\/photos\/albums\/([^/]+)$/);
	if (match) return `/v1/profiles/me/photo-albums/${match[1]}/photos${suffix}`;
	if (pathname === "/profile/photos/albums") return `/v1/profiles/me/photo-albums${suffix}`;
	match = pathname.match(/^\/profile\/photos\/([^/]+)$/);
	if (match) return `/v1/profiles/me/photos/${match[1]}${suffix}`;
	match = pathname.match(/^\/music\/([^/]+)\/save$/);
	if (match) return `/v1/media/music/${match[1]}/save${suffix}`;
	match = pathname.match(/^\/music\/([^/]+)$/);
	if (match) return `/v1/media/music/${match[1]}${suffix}`;
	if (pathname.endsWith("/music")) return `/v1/media/music${suffix}`;
	match = pathname.match(/^\/messages\/conversations\/([^/]+)\/messages(?:\/([^/]+))?$/);
	if (match) return `/v1/messaging/conversations/${match[1]}/messages${match[2] ? `/${match[2]}` : ""}${suffix}`;
	match = pathname.match(/^\/messages\/conversations\/([^/]+)(?:\/([^/]+))?$/);
	if (match) return `/v1/messaging/conversations/${match[1]}${match[2] ? `/${match[2]}` : ""}${suffix}`;
	if (pathname === "/messages/conversations/direct") return `/v1/messaging/conversations/direct${suffix}`;
	if (pathname === "/messages/conversations") return `/v1/messaging/conversations${suffix}`;
	if (pathname === "/messages") return `/v1/messaging/conversations${suffix}`;
	match = pathname.match(/^\/friends\/recommendations$/);
	if (match) return `/v1/social/recommendations${suffix}`;
	match = pathname.match(/^\/friends\/requests$/);
	if (match) return `/v1/social/friend-requests${suffix}`;
	match = pathname.match(/^\/friends\/([^/]+)$/);
	if (match) return `/v1/social/friendships/${match[1]}${suffix}`;
	match = pathname.match(/^\/friends\/subscriptions\/([^/]+)$/);
	if (match) return `/v1/social/subscriptions/${match[1]}${suffix}`;
	match = pathname.match(/^\/videos\/([^/]+)\/(view|like|bookmark|favorite)$/);
	if (match) return `/v1/media/videos/${match[1]}/${match[2]}${suffix}`;
	match = pathname.match(/^\/videos\/albums(?:\/([^/]+))?$/);
	if (match) return `/v1/media/videos/albums${match[1] ? `/${match[1]}` : ""}${suffix}`;
	match = pathname.match(/^\/videos(?:\/([^/]+))?$/);
	return match ? `/v1/media/videos${match[1] ? `/${match[1]}` : ""}${suffix}` : `${pathname}${suffix}`;
}

async function parseResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = response.statusText;

    try {
      const data = (await response.json()) as ApiError;
		message = data.error?.message ?? data.detail ?? message;
    } catch {
      // Error responses can be empty.
    }

    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();

  if (!text) {
    return undefined as T;
  }

	return JSON.parse(text) as T;
}

function adaptResponse<T>(path: string, data: unknown): T {
	if (!data || typeof data !== "object") return data as T;
	const value = data as Record<string, unknown>;
	const requestPath = path.split("?", 1)[0];

	if (path === "/posts" && Array.isArray(value.posts)) {
		return {
			...value,
			current_user: null,
			page: 1,
			total_pages: 1,
			posts: value.posts.map((post) => adaptPost(post)),
		} as T;
	}
	if (path === "/friends" && Array.isArray(value.friends)) {
		return {
			current_user: null,
			friends: idsToUsers(value.friends),
			subscribers: idsToUsers(value.subscribers),
			subscriptions: idsToUsers(value.subscriptions),
		} as T;
	}
	if (path === "/friends/recommendations" && Array.isArray(value.recommendations)) {
		return {
			recommendations: value.recommendations.map((item) => {
				const recommendation = item as Record<string, unknown>;
				return { user: userFromID(recommendation.user_id), common_friends: recommendation.common_friends ?? 0 };
			}),
		} as T;
	}
	if (path.startsWith("/search") && Array.isArray(value.profiles)) {
		return {
			users: value.profiles.map((profile) => {
				const item = profile as Record<string, unknown>;
				return {
					id: item.user_id,
					username: item.handle ?? "",
					full_name: [item.first_name, item.last_name].filter((part) => typeof part === "string" && part.trim()).join(" ") || item.handle || "Пользователь",
					avatar: item.avatar_url ?? null,
				};
			}),
			posts: [],
			videos: [],
		} as T;
	}
	if (path.includes("/videos") && Array.isArray(value.albums)) {
		return {
			...value,
			albums: value.albums.map((album) => {
				const item = album as Record<string, unknown>;
				return {
					...item,
					videos: Array.isArray(item.videos) ? item.videos.map(adaptVideo) : [],
				};
			}),
		} as T;
	}
	if (requestPath.startsWith("/messages/conversations") && requestPath.endsWith("/messages") && Array.isArray(value.messages)) {
		return {
			items: value.messages.map((message) => adaptMessage(message)),
			next_cursor: null,
			has_more: value.has_more ?? false,
		} as T;
	}
	if (path.includes("/comments") && Array.isArray(value.comments)) {
		return { ...value, comments: value.comments.map(adaptComment) } as T;
	}
	if (path.includes("/photos") && Array.isArray(value.albums)) {
		return { ...value, user: userFromID(value.owner_id) } as T;
	}
	if (requestPath === "/messages/conversations" && Array.isArray(value.conversations)) {
		return value.conversations.map(adaptConversation) as T;
	}
	if (requestPath.startsWith("/messages/conversations/") && "id" in value && "participant_ids" in value) {
		return adaptConversation(value) as T;
	}
	if ("body" in value && "conversation_id" in value) {
		return adaptMessage(value) as T;
	}
	if (path.startsWith("/profile/") && "user_id" in value) {
		const profile = value as Record<string, string | null>;
		return {
			user: userFromProfile(profile),
			current_user: userFromProfile(profile),
			privacy: value.privacy,
			is_own_profile: path === "/profile/me",
			is_friend: false,
			is_subscribed: false,
			is_subscribed_to_current: false,
			posts: [],
			friends: [],
		} as T;
	}
	return data as T;
}

function adaptPost(post: unknown) {
	const value = post as Record<string, unknown>;
	return {
		...value,
		content: value.content ?? value.body ?? null,
		image: value.image ?? null,
		image_content_type: value.image_content_type ?? null,
		media_ids: Array.isArray(value.media_ids) ? value.media_ids.map(String) : [],
		likes_count: value.likes_count ?? 0,
		comments_count: value.comments_count ?? 0,
		is_liked_by_current: value.is_liked_by_current ?? false,
		liked_user_ids: Array.isArray(value.liked_user_ids) ? value.liked_user_ids.map(String) : [],
		liked_users: value.liked_users ?? [],
	};
}

function adaptMessage(message: unknown) {
	const value = message as Record<string, unknown>;
	return {
		...value,
		text: value.text ?? value.body ?? "",
		media_content_type: value.media_content_type ?? null,
		deleted_at: value.deleted ? value.created_at : null,
	};
}

function adaptComment(comment: unknown) {
	const value = comment as Record<string, unknown>;
	return {
		...value,
		user_id: String(value.user_id ?? value.author_id ?? ""),
		text: value.text ?? value.body ?? "",
		parent_id: value.parent_id == null ? null : String(value.parent_id),
		likes_count: value.likes_count ?? 0,
		is_liked_by_current: value.is_liked_by_current ?? false,
	};
}

function adaptVideo(video: unknown) {
	const value = video as Record<string, unknown>;
	return {
		...value,
		video_id: value.video_id ?? value.id,
		owner_id: value.owner_id ?? null,
		owner_name: value.owner_name ?? "Пользователь",
		error: value.error ?? null,
		created_at: value.created_at ?? null,
		is_liked_by_current: value.is_liked_by_current ?? false,
		is_bookmarked_by_current: value.is_bookmarked_by_current ?? false,
		is_favorited_by_current: value.is_favorited_by_current ?? false,
	};
}

function adaptConversation(conversation: unknown) {
	const value = conversation as Record<string, unknown>;
	const participants = Array.isArray(value.participant_ids) ? value.participant_ids : [];
	return {
		...value,
		other_user_id: participants[0] ?? null,
		can_send_message: true,
		last_message: value.last_message ? {
			...(adaptMessage(value.last_message) as Record<string, unknown>),
			content_type: null,
		} : null,
	};
}

function userFromID(id: unknown) {
	return { id: String(id ?? ""), username: String(id ?? "").slice(0, 8), profile: null };
}

function userFromProfile(profile: Record<string, string | null>) {
	return {
		id: profile.user_id ?? "",
		username: profile.handle ?? "",
		email: undefined,
		profile: {
			first_name: profile.first_name ?? null, last_name: profile.last_name ?? null, middle_name: profile.middle_name ?? null,
			full_name: [profile.first_name, profile.last_name].filter((part) => typeof part === "string" && part.trim()).join(" ") || null, birth_date: profile.birth_date ?? null, gender: profile.gender ?? null,
			phone_number: profile.phone_number ?? null, country: profile.country ?? null, city: profile.city ?? null, street: profile.street ?? null,
			avatar: profile.avatar_url ?? null, bio: profile.bio ?? null, status: profile.status ?? null,
		},
	};
}

function idsToUsers(ids: unknown) {
	return Array.isArray(ids) ? ids.map(userFromID) : [];
}
