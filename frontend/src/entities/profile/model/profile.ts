import type { User } from "../../user/model/user";

export type Profile = {
  first_name: string | null;
  last_name: string | null;
  middle_name: string | null;
  full_name: string | null;
  birth_date: string | null;
  gender: string | null;
  phone_number: string | null;
  country: string | null;
  city: string | null;
  street: string | null;
  avatar: string | null;
  bio: string | null;
  status: string | null;
};

export type ProfilePrivacy = {
  profile_visibility: string;
  friend_request_policy: string;
  message_policy: string;
  phone_visibility: string;
  birth_date_visibility: string;
  gender_visibility: string;
  location_visibility: string;
  status_visibility: string;
  friends_visibility: string;
  posts_visibility: string;
  music_visibility: string;
};

export type ProfileForm = {
  first_name: string;
  last_name: string;
  middle_name: string;
  birth_date: string;
  gender: string;
  phone_number: string;
  country: string;
  city: string;
  street: string;
  bio: string;
  status: string;
};

export type AvatarHistoryItem = {
  id: string | number;
  media_id?: string;
  avatar_url: string;
  created_at: string;
  is_current: boolean;
};

export type AvatarHistoryResponse = {
  current_avatar: string | null;
  avatars: AvatarHistoryItem[];
};

export type AvatarUploadResponse = {
  message: string;
  avatar_url: string;
};

export type ProfilePhoto = {
	archived?: boolean;
	latitude?: number | null;
	longitude?: number | null;
  id: string | number | null;
  media_id?: string;
  album_id: string | number | null;
  photo_url: string;
  caption: string | null;
  created_at: string;
};

export type ProfilePhotoAlbum = {
	description?: string;
	visibility?: "public" | "private";
	comment_policy?: "public" | "private" | "nobody";
  id: string | number | null;
  title: string;
  kind: "avatars" | "custom";
  created_at: string | null;
  photos: ProfilePhoto[];
};

export type ProfilePhotosResponse = {
  user: User | null;
  is_own_profile: boolean;
  albums: ProfilePhotoAlbum[];
};

export type PhotoAlbumEnvelopeResponse = {
  album: ProfilePhotoAlbum;
};

export function profileToForm(profile: Profile | null): ProfileForm {
  return {
    first_name: profile?.first_name ?? "",
    last_name: profile?.last_name ?? "",
    middle_name: profile?.middle_name ?? "",
    birth_date: profile?.birth_date ?? "",
    gender: profile?.gender ?? "",
    phone_number: profile?.phone_number ?? "",
    country: profile?.country ?? "",
    city: profile?.city ?? "",
    street: profile?.street ?? "",
    bio: profile?.bio ?? "",
    status: profile?.status ?? "",
  };
}

export function emptyStringsToNull(form: ProfileForm) {
  return Object.fromEntries(
    Object.entries(form).map(([key, value]) => [key, value.trim() === "" ? null : value.trim()]),
  );
}
