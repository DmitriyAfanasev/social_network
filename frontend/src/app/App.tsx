import { ConfirmRegistrationPage } from "../pages/auth/ConfirmRegistrationPage";
import { AdminPage } from "../pages/admin/AdminPage";
import { LoginPage } from "../pages/auth/LoginPage";
import { MessagesPage } from "../pages/messages/MessagesPage";
import { MusicPage } from "../pages/music/MusicPage";
import { AddMusicPage } from "../pages/music/AddMusicPage";
import { SearchPage } from "../pages/search/SearchPage";
import { RegisterPage } from "../pages/auth/RegisterPage";
import { SettingsPage } from "../pages/settings/SettingsPage";
import { FeedPage } from "../pages/feed/FeedPage";
import { FriendsPage } from "../pages/friends/FriendsPage";
import { PhotosPage } from "../pages/photos/PhotosPage";
import { VideosPage } from "../pages/videos/VideosPage";
import { EditProfilePage } from "../pages/profile/EditProfilePage";
import { ProfilePage } from "../pages/profile/ProfilePage";
import { NotFoundPage } from "../pages/system/NotFoundPage";
import { PlaceholderPage } from "../pages/system/PlaceholderPage";
import { usePath } from "../shared/lib/navigation";
import { Shell } from "../widgets/app-shell/Shell";

export function App() {
  const path = usePath();
  const cleanPath = path.split("?")[0];
  const profileMatch = cleanPath.match(/^\/profile\/(?<id>[^/]+)$/);
  const profileEditMatch = cleanPath.match(/^\/profile\/(?<id>[^/]+)\/edit$/);
  const profilePhotosMatch = cleanPath.match(/^\/profile\/(?<id>[^/]+)\/photos$/);
  const profileVideosMatch = cleanPath.match(/^\/profile\/(?<id>[^/]+)\/videos$/);
  const conversationMatch = cleanPath.match(/^\/messages\/(?<id>[^/]+)$/);
  const matched =
    cleanPath === "/" ||
    cleanPath === "/login" ||
    cleanPath === "/register" ||
    cleanPath === "/confirm-registration" ||
    cleanPath === "/confirm" ||
    cleanPath === "/friends" ||
    cleanPath === "/photos" ||
    cleanPath === "/videos" ||
    cleanPath === "/messages" ||
    cleanPath === "/music" ||
    cleanPath === "/music/add" ||
    cleanPath === "/search" ||
    Boolean(conversationMatch?.groups?.id) ||
    cleanPath === "/notifications" ||
    cleanPath === "/settings" ||
    cleanPath === "/admin" ||
    Boolean(profileMatch?.groups?.id) ||
    Boolean(profileEditMatch?.groups?.id) ||
    Boolean(profilePhotosMatch?.groups?.id) ||
    Boolean(profileVideosMatch?.groups?.id);

  return (
    <Shell path={cleanPath}>
      {cleanPath === "/" && <FeedPage />}
      {cleanPath === "/login" && <LoginPage />}
      {cleanPath === "/register" && <RegisterPage />}
      {(cleanPath === "/confirm-registration" || cleanPath === "/confirm") && <ConfirmRegistrationPage />}
      {cleanPath === "/friends" && <FriendsPage />}
      {cleanPath === "/photos" && <PhotosPage />}
      {cleanPath === "/videos" && <VideosPage />}
      {(cleanPath === "/messages" || conversationMatch?.groups?.id) && <MessagesPage />}
      {cleanPath === "/music" && <MusicPage />}
      {cleanPath === "/music/add" && <AddMusicPage />}
      {cleanPath === "/search" && <SearchPage />}
      {cleanPath === "/notifications" && (
        <PlaceholderPage
          title="Уведомления"
          description="Новые заявки в друзья и другие события приходят сюда в реальном времени."
        />
      )}
      {cleanPath === "/settings" && (
        <SettingsPage />
      )}
      {cleanPath === "/admin" && <AdminPage />}
      {profileEditMatch?.groups?.id && <EditProfilePage id={profileEditMatch.groups.id} />}
      {profilePhotosMatch?.groups?.id && <PhotosPage profileId={profilePhotosMatch.groups.id} />}
      {profileVideosMatch?.groups?.id && <VideosPage profileId={profileVideosMatch.groups.id} />}
      {profileMatch?.groups?.id && <ProfilePage id={profileMatch.groups.id} />}
      {!matched && <NotFoundPage />}
    </Shell>
  );
}
