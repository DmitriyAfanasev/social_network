import type { Profile } from "../../profile/model/profile";

export type User = {
  id: string;
  username: string;
  email?: string;
  is_active?: boolean;
  last_seen_at?: string | null;
  profile: Profile | null;
};

export type PublicProfileResponse = {
  user_id: string;
  handle?: string | null;
  full_name?: string | null;
  first_name?: string | null;
  last_name?: string | null;
  middle_name?: string | null;
  birth_date?: string | null;
  gender?: string | null;
  phone_number?: string | null;
  country?: string | null;
  city?: string | null;
  street?: string | null;
  status?: string | null;
  bio?: string | null;
  avatar_url?: string | null;
};

export function userFromPublicProfile(profile: PublicProfileResponse): User {
  return {
    id: profile.user_id,
    username: profile.handle ?? "",
    profile: {
      first_name: profile.first_name ?? null,
      last_name: profile.last_name ?? null,
      middle_name: profile.middle_name ?? null,
      full_name: profile.full_name?.trim() || [profile.first_name, profile.last_name].filter((part) => Boolean(part?.trim())).join(" ") || null,
      birth_date: profile.birth_date ?? null,
      gender: profile.gender ?? null,
      phone_number: profile.phone_number ?? null,
      country: profile.country ?? null,
      city: profile.city ?? null,
      street: profile.street ?? null,
      avatar: profile.avatar_url || null,
      bio: profile.bio ?? null,
      status: profile.status ?? null,
    },
  };
}

export type AuthUser = {
  id: string;
  email: string;
  status: string;
  last_seen_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type AuthResponse = {
	user: AuthUser;
	access_token: string;
	refresh_token: string;
	token_type: string;
	expires_in: number;
	refresh_expires_in: number;
};

export type FriendsResponse = {
  current_user: User | null;
  friends: User[];
  subscribers: User[];
  subscriptions: User[];
};

export type FriendRecommendation = {
  user: User;
  common_friends: number;
};

export type FriendRecommendationsResponse = {
  recommendations: FriendRecommendation[];
};

export type FriendActionResponse = {
  success: boolean;
  is_friend: boolean;
  is_subscribed: boolean;
  is_subscribed_to_current: boolean;
  message: string;
};

export type FriendRequest = {
  id: string;
  sender_id: string;
  recipient_id: string;
  status: string;
  created_at: string;
};

export type FriendRequestsResponse = {
  incoming: FriendRequest[];
  outgoing: FriendRequest[];
};

export function getUserName(user: User | null | undefined) {
  const fullName = user?.profile?.full_name?.trim();
  if (fullName) {
    return fullName;
  }

  const detailsName = [
    user?.profile?.first_name,
    user?.profile?.last_name,
  ]
    .map((part) => part?.trim())
    .filter(Boolean)
    .join(" ");
  if (detailsName) {
    return detailsName;
  }

  return user?.username || "Без имени";
}

export function getInitials(user: User | null | undefined) {
  return getUserName(user)
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase();
}
