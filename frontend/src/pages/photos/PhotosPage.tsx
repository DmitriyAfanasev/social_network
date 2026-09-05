import { useEffect, useState } from "react";
import type { ChangeEvent } from "react";

import type { FeedResponse } from "../../entities/post/model/post";
import type {
  PhotoAlbumEnvelopeResponse,
  ProfilePhotoAlbum,
  ProfilePhotosResponse,
} from "../../entities/profile/model/profile";
import { getUserName } from "../../entities/user/model/user";
import { apiRequest } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { navigate } from "../../shared/lib/navigation";

type PhotosPageProps = {
  profileId?: string;
};

type PendingAlbumUpload = {
  albumId: string | number;
  file: File;
  previewUrl: string;
};

export function PhotosPage({ profileId }: PhotosPageProps) {
  const [resolvedProfileId, setResolvedProfileId] = useState<string | null>(profileId ?? null);
  const [photos, setPhotos] = useState<ProfilePhotosResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [newAlbumTitle, setNewAlbumTitle] = useState("");
  const [creatingAlbum, setCreatingAlbum] = useState(false);
  const [albumUpdatingId, setAlbumUpdatingId] = useState<string | number | null>(null);
  const [deletingPhotoId, setDeletingPhotoId] = useState<string | number | null>(null);
  const [pendingUpload, setPendingUpload] = useState<PendingAlbumUpload | null>(null);

  useEffect(() => {
    return () => {
      if (pendingUpload) {
        URL.revokeObjectURL(pendingUpload.previewUrl);
      }
    };
  }, [pendingUpload]);

  async function resolveCurrentProfileId() {
    if (profileId && profileId !== "me") {
      setResolvedProfileId(profileId);
      return profileId;
    }

    const profile = await apiRequest<{ user_id: string }>("/v1/profiles/me");
    if (!profile.user_id) {
      setError("Не удалось определить пользователя профиля.");
      return null;
    }

    setResolvedProfileId(profile.user_id);
    return profile.user_id;
  }

  async function loadPhotos() {
    setLoading(true);
    setError(null);

    try {
      const targetId = await resolveCurrentProfileId();
      if (!targetId) {
        return;
      }

      const result = await apiRequest<ProfilePhotosResponse>(`/profile/${targetId}/photos`);
      setPhotos(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить фотографии");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadPhotos();
  }, [profileId]);

  async function createAlbum() {
    const title = newAlbumTitle.trim();
    if (!title) {
      setError("Введите название альбома.");
      return;
    }

    setCreatingAlbum(true);
    setError(null);

    try {
      await apiRequest<PhotoAlbumEnvelopeResponse>("/profile/photos/albums", {
        method: "POST",
        body: JSON.stringify({ title }),
      });
      setNewAlbumTitle("");
      await loadPhotos();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось создать альбом");
    } finally {
      setCreatingAlbum(false);
    }
  }

  function selectAlbumPhoto(albumId: string | number, event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }

    if (pendingUpload) {
      URL.revokeObjectURL(pendingUpload.previewUrl);
    }

    setPendingUpload({
      albumId,
      file,
      previewUrl: URL.createObjectURL(file),
    });
    setError(null);
    event.target.value = "";
  }

  function clearPendingUpload() {
    if (pendingUpload) {
      URL.revokeObjectURL(pendingUpload.previewUrl);
    }
    setPendingUpload(null);
  }

  async function confirmAlbumUpload() {
    if (!pendingUpload) {
      return;
    }

    const upload = pendingUpload;
    setAlbumUpdatingId(upload.albumId);
    setError(null);

    try {
      const mediaBody = new FormData();
      mediaBody.append("file", upload.file);
      const media = await apiRequest<{ id: string }>("/v1/media", { method: "POST", body: mediaBody });
      await apiRequest<PhotoAlbumEnvelopeResponse>(`/profile/photos/albums/${upload.albumId}`, {
        method: "POST",
        body: JSON.stringify({ media_id: media.id, caption: "" }),
      });
      clearPendingUpload();
      await loadPhotos();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить фото");
    } finally {
      setAlbumUpdatingId(null);
    }
  }

  async function deletePhoto(photoId: string | number) {
    if (!window.confirm("Удалить фотографию из альбома?")) {
      return;
    }

    setDeletingPhotoId(photoId);
    setError(null);

    try {
      await apiRequest<{ message: string }>(`/profile/photos/${photoId}`, { method: "DELETE" });
      await loadPhotos();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить фото");
    } finally {
      setDeletingPhotoId(null);
    }
  }

  const albums = photos?.albums ?? [];
  const avatarAlbum = albums.find((album) => album.kind === "avatars");
  const customAlbums = albums.filter((album) => album.kind === "custom");
  const photoCount = albums.reduce((count, album) => count + album.photos.length, 0);
  const ownerName = photos?.user ? getUserName(photos.user) : "Фотографии";

  function renderPhotoAlbum(album: ProfilePhotoAlbum) {
    const canManageAlbum = Boolean(photos?.is_own_profile && (album.kind === "custom" || album.kind === "avatars"));
    const canUploadAlbum = Boolean(photos?.is_own_profile && album.kind === "custom" && album.id);
    const isCurrentUploadTarget = pendingUpload?.albumId === album.id;

    return (
      <article key={`${album.kind}-${album.id ?? "avatars"}`} className="photo-album">
        <header>
          <div>
            <h3>{album.title}</h3>
            <span>{album.kind === "avatars" ? "Автоматический альбом" : `${album.photos.length} фото`}</span>
          </div>

        </header>
        {isCurrentUploadTarget && pendingUpload && (
          <div className="photo-upload-preview">
            <img src={pendingUpload.previewUrl} alt="" />
            <div>
              <strong>{pendingUpload.file.name}</strong>
              <span>{Math.max(1, Math.round(pendingUpload.file.size / 1024))} КБ</span>
            </div>
            <div className="photo-upload-actions">
              <button
                type="button"
                disabled={albumUpdatingId === album.id}
                onClick={() => void confirmAlbumUpload()}
              >
                {albumUpdatingId === album.id ? "Загружаем..." : "Загрузить"}
              </button>
              <button type="button" className="secondary" disabled={albumUpdatingId === album.id} onClick={clearPendingUpload}>
                Отмена
              </button>
            </div>
          </div>
        )}
        {album.photos.length === 0 ? (
          <p className="muted">В альбоме пока нет фотографий.</p>
        ) : (
          <div className="photo-grid">
            {album.photos.map((photo) => {
              const photoId = photo.id;

              return (
                <div key={`${photo.id ?? photo.photo_url}-${photo.created_at}`} className="photo-tile">
                  <a href={`${API_BASE_URL}${photo.photo_url}`} target="_blank" rel="noreferrer">
                    <img src={`${API_BASE_URL}${photo.photo_url}`} alt="" />
                  </a>
                  {canManageAlbum && photoId !== null && (
                    <button
                      type="button"
                      className="photo-delete-button"
                      aria-label="Удалить фотографию"
                      title="Удалить фотографию"
                      disabled={deletingPhotoId === photoId}
                      onClick={() => void deletePhoto(photoId)}
                    >
                      {deletingPhotoId === photoId ? "…" : "×"}
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </article>
    );
  }

  return (
    <section className="photos-page">
      <header className="page-header photos-header">
        <div>
          <span className="eyebrow">Фотографии</span>
          <h1>{ownerName}</h1>
          <p className="muted">
            {photos?.is_own_profile
              ? "Альбомы профиля, архив аватаров и загруженные фотографии."
              : "Фотографии и публичные альбомы пользователя."}
          </p>
        </div>
        {resolvedProfileId && (
          <button type="button" className="secondary" onClick={() => navigate(`/profile/${profileId ?? "me"}`)}>
            Профиль
          </button>
        )}
      </header>
      {photos?.is_own_profile && (
        <div className="photo-uploader-panel">
          <div>
            <strong>Новый альбом</strong>
            <span>Создайте понятную папку для фото, загрузка появится внутри альбома.</span>
          </div>
          <input
            value={newAlbumTitle}
            onChange={(event) => setNewAlbumTitle(event.target.value)}
            placeholder="Например: Лето, Работа, Поездки"
          />
          <button type="button" disabled={creatingAlbum} onClick={() => void createAlbum()}>
            {creatingAlbum ? "Создаём..." : "Создать"}
          </button>
        </div>
      )}
      {error && <p className="error">{error}</p>}
      {loading && <p className="muted">Загружаем фотографии...</p>}
      {!loading && albums.length === 0 && (
        <div className="empty-state">
          <h2>Фотографий пока нет</h2>
          <p>Создайте первый альбом и загрузите туда фото.</p>
        </div>
      )}
      <div className="section-title">
        <h2>Альбомы</h2>
        <span>{photoCount}</span>
      </div>
      <div className="photo-album-list">
        {customAlbums.map(renderPhotoAlbum)}
        {avatarAlbum && renderPhotoAlbum(avatarAlbum)}
      </div>
    </section>
  );
}
