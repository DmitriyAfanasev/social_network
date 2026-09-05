import type { ProfilePrivacy } from "../../profile/model/profile";
import type { User } from "../../user/model/user";

export type Post = {
  id: number;
  content: string | null;
  image: string | null;
  image_content_type?: string | null;
  media_ids?: string[];
  author_id: number;
  likes_count: number;
  comments_count: number;
  is_liked_by_current: boolean;
  liked_user_ids: string[];
  liked_users: User[];
  author?: User | null;
  created_at: string;
  updated_at: string;
  featured_comment?: Comment | null;
};

export type Comment = {
  id: number;
  post_id: number;
  user_id: number;
  parent_id: number | null;
  text: string;
  author?: User | null;
  created_at: string;
  updated_at: string;
  likes_count: number;
  is_liked_by_current: boolean;
};

export type CommentsPageResponse = {
  comments: Comment[];
  has_more: boolean;
  offset: number;
  limit: number;
  post_id: number;
};

export type FeedResponse = {
  current_user: User | null;
  posts: Post[];
  page: number;
  total_pages: number;
};

export type ProfileResponse = {
  user: User;
  is_own_profile: boolean;
  is_friend: boolean;
  is_subscribed: boolean;
  is_subscribed_to_current: boolean;
  can_send_friend_request?: boolean;
  can_send_message?: boolean;
  current_user: User;
  posts: Post[];
  friends: User[];
  profile_visibility?: string | null;
  friend_request_policy?: string | null;
  message_policy?: string | null;
  show_email?: boolean | null;
  show_phone?: boolean | null;
  show_birth_date?: boolean | null;
  show_friends?: boolean | null;
  show_posts?: boolean | null;
  privacy?: ProfilePrivacy;
};

export type LikeResponse = {
  likes_count: number;
  liked: boolean;
};

export type PostResponse = {
  post: Post;
};
