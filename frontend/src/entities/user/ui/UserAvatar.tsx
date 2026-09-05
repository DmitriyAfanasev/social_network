import { API_BASE_URL } from "../../../shared/config/api";
import { getInitials } from "../model/user";
import type { User } from "../model/user";

type UserAvatarProps = {
  user?: User | null;
  size?: "sm" | "md" | "lg";
};

export function UserAvatar({ user, size = "md" }: UserAvatarProps) {
  const avatar = user?.profile?.avatar;

  if (avatar) {
    const avatarUrl = avatar.startsWith("http://") || avatar.startsWith("https://") ? avatar : `${API_BASE_URL}${avatar}`;
    return <img className={`avatar avatar-${size}`} src={avatarUrl} alt="" />;
  }

  return <span className={`avatar avatar-${size}`}>{getInitials(user)}</span>;
}
