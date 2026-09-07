import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import type { ProfilePhoto, ProfilePhotoAlbum, ProfilePhotosResponse, PhotoAlbumEnvelopeResponse } from "../../entities/profile/model/profile";
import { apiBlob, apiRequest } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import "./photos.css";

type ID = string | number;
type Comment = { id: string; author_name: string; body: string; created_at: string };
type AlbumForm = { title: string; description: string; visibility: string; comment_policy: string };
const emptyAlbum: AlbumForm = { title: "", description: "", visibility: "public", comment_policy: "public" };
const photoURL = (photo: ProfilePhoto) => photo.photo_url.startsWith("http") ? photo.photo_url : API_BASE_URL + photo.photo_url;
const mapURL = (photo: ProfilePhoto) => "https://www.openstreetmap.org/?mlat=" + photo.latitude + "&mlon=" + photo.longitude + "#map=15/" + photo.latitude + "/" + photo.longitude;

function GalleryImage({ photo, alt, loading }: { photo: ProfilePhoto; alt: string; loading?: "lazy" }) {
  const [src, setSrc] = useState("");
  const [failed, setFailed] = useState(false);
  const ref = useRef<HTMLImageElement>(null);
  useEffect(() => {
    const controller = new AbortController();
    let objectURL = "";
    setSrc(""); setFailed(false);
    async function load() {
      try {
        const url = new URL(photoURL(photo));
        if (url.origin !== new URL(API_BASE_URL).origin) { setSrc(url.href); return; }
        const blob = await apiBlob(url.pathname, controller.signal);
        if (controller.signal.aborted) return;
        objectURL = URL.createObjectURL(blob); setSrc(objectURL);
      } catch { if (!controller.signal.aborted) setFailed(true); }
    }
    const observer = new IntersectionObserver((entries) => { if (entries.some((entry) => entry.isIntersecting)) { observer.disconnect(); void load(); } }, { rootMargin: "200px" });
    if (loading && ref.current) observer.observe(ref.current); else void load();
    return () => { observer.disconnect(); controller.abort(); if (objectURL) URL.revokeObjectURL(objectURL); };
  }, [photo.photo_url, loading]);
  return <img ref={ref} src={src || undefined} alt={failed ? "Фотография недоступна" : alt} loading={loading}/>;
}

function Icon({ name }: { name: "photo" | "lock" | "archive" | "pin" | "comment" | "check" | "settings" }) {
  const paths = { photo: <><rect x="3" y="3" width="18" height="18" rx="4"/><circle cx="8" cy="8" r="1"/><path d="m3 16 5-5 4 4 4-3 5 5"/></>, lock: <><rect x="6" y="10" width="12" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></>, archive: <><rect x="3" y="3" width="18" height="5" rx="1"/><path d="M5 8v12h14V8M9 12h6"/></>, pin: <><path d="M19 9c0 5-7 12-7 12S5 14 5 9a7 7 0 0 1 14 0Z"/><circle cx="12" cy="9" r="2"/></>, comment: <path d="M5 3h14a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-6l-5 4v-4H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z"/>, check: <><circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/></>, settings: <><circle cx="12" cy="12" r="4"/><path d="m10 2 4 0 1 3 3 1 3 3-1 3 1 3-3 3-3 1-1 3h-4l-1-3-3-1-3-3 1-3-1-3 3-3 3-1Z"/></> };
  return <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>;
}

function Modal({ title, close, children, wide = false }: { title: string; close: () => void; children: ReactNode; wide?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { const dialog = ref.current!; dialog.showModal(); return () => dialog.close(); }, []);
  return <dialog ref={ref} className={"gallery-modal " + (wide ? "gallery-modal-wide" : "")} aria-label={title} onCancel={(event) => { event.preventDefault(); close(); }} onClick={(event) => { if (event.target === event.currentTarget) close(); }}>
    <header><span>{title}</span><button type="button" className="gallery-icon-button" aria-label="Закрыть" onClick={close}>×</button></header>{children}
  </dialog>;
}

export function PhotosPage({ profileId }: { profileId?: string }) {
  const [data, setData] = useState<ProfilePhotosResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState<"photos" | "albums">("photos");
  const [view, setView] = useState<"all" | "archive" | "map" | "comments">("all");
  const [albumKey, setAlbumKey] = useState<string | null>(null);
  const [selecting, setSelecting] = useState(false);
  const [selection, setSelection] = useState<Set<ID>>(new Set());
  const [albumForm, setAlbumForm] = useState<AlbumForm | null>(null);
  const [editingAlbum, setEditingAlbum] = useState<ID | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadAlbum, setUploadAlbum] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [previewURLs, setPreviewURLs] = useState<string[]>([]);
  const [opened, setOpened] = useState<ProfilePhoto | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [comment, setComment] = useState("");
  const [caption, setCaption] = useState("");
  const [latitude, setLatitude] = useState("");
  const [longitude, setLongitude] = useState("");
  const [mapPhotoID, setMapPhotoID] = useState<ID | null>(null);
  const generation = useRef(0);

  async function fetchPhotos() {
    const target = !profileId || profileId === "me" ? (await apiRequest<{ user_id: string }>("/v1/profiles/me")).user_id : profileId;
    return apiRequest<ProfilePhotosResponse>("/v1/profiles/" + target + "/photos");
  }
  async function refresh() { setData(await fetchPhotos()); }
  useEffect(() => {
    const current = ++generation.current;
    setLoading(true); setData(null); setError(""); setAlbumKey(null); setSelection(new Set()); setOpened(null);
    void fetchPhotos().then((result) => { if (current === generation.current) setData(result); }).catch((err) => { if (current === generation.current) setError(String(err.message ?? err)); }).finally(() => { if (current === generation.current) setLoading(false); });
    return () => { generation.current++; };
  }, [profileId]);
  useEffect(() => {
    const urls = files.map((file) => URL.createObjectURL(file)); setPreviewURLs(urls);
    return () => urls.forEach((url) => URL.revokeObjectURL(url));
  }, [files]);
  useEffect(() => {
    let active = true; setComments([]); setComment("");
    if (!opened?.album_id || !opened.id) return;
    setCommentsLoading(true);
    void apiRequest<{ comments: Comment[] }>("/v1/profiles/photos/" + opened.id + "/comments").then((result) => { if (active) setComments(result.comments); }).catch((err) => { if (active) setError(err.message); }).finally(() => { if (active) setCommentsLoading(false); });
    return () => { active = false; };
  }, [opened?.id]);

  const own = Boolean(data?.is_own_profile);
  const albums = data?.albums ?? [];
  const key = (album: ProfilePhotoAlbum) => String(album.id ?? album.kind);
  const activeAlbum = albums.find((album) => key(album) === albumKey);
  const customAlbums = albums.filter((album) => album.kind === "custom");
  const allPhotos = (activeAlbum ? [activeAlbum] : albums).flatMap((album) => album.photos).filter((photo) => view === "archive" ? photo.archived : !photo.archived).sort((a, b) => b.created_at.localeCompare(a.created_at));
  const shownPhotos = view === "map" ? allPhotos.filter((photo) => photo.latitude != null && photo.longitude != null) : allPhotos;
  const years = [...new Set(shownPhotos.map((photo) => photo.created_at.slice(0, 4)))];
  const mapPhoto = shownPhotos.find((photo) => photo.id === mapPhotoID) ?? shownPhotos[0];
  const mapEmbed = mapPhoto?.latitude != null && mapPhoto.longitude != null ? "https://www.openstreetmap.org/export/embed.html?bbox=" + [Math.max(-180, mapPhoto.longitude - .03), Math.max(-90, mapPhoto.latitude - .02), Math.min(180, mapPhoto.longitude + .03), Math.min(90, mapPhoto.latitude + .02)].join(",") + "&layer=mapnik&marker=" + mapPhoto.latitude + "," + mapPhoto.longitude : "";
  const openedAlbum = albums.find((album) => album.id === opened?.album_id);
  const canComment = opened?.album_id && openedAlbum?.comment_policy !== "nobody" && (own || openedAlbum?.comment_policy !== "private");

  async function run(action: () => Promise<void>) {
    setBusy(true); setError("");
    try { await action(); } catch (err) { setError(err instanceof Error ? err.message : "Не удалось выполнить действие"); } finally { setBusy(false); }
  }
  function openPhoto(photo: ProfilePhoto) { setOpened(photo); setCaption(photo.caption ?? ""); setLatitude(photo.latitude?.toString() ?? ""); setLongitude(photo.longitude?.toString() ?? ""); setError(""); }
  function changeView(next: typeof view) { setView(next); setTab("photos"); setAlbumKey(null); setSelection(new Set()); setSelecting(false); }
  function editAlbum(album?: ProfilePhotoAlbum) {
    setEditingAlbum(album?.id ?? null); setAlbumForm(album ? { title: album.title, description: album.description ?? "", visibility: album.visibility ?? "public", comment_policy: album.comment_policy ?? "public" } : { ...emptyAlbum }); setError("");
  }
  async function saveAlbum() {
    if (!albumForm) return;
    await run(async () => {
      const result = await apiRequest<PhotoAlbumEnvelopeResponse>("/v1/profiles/me/photo-albums" + (editingAlbum ? "/" + editingAlbum : ""), { method: editingAlbum ? "PUT" : "POST", body: JSON.stringify(albumForm) });
      await refresh(); setAlbumForm(null); setTab("albums");
      if (uploading) setUploadAlbum(String(result.album.id));
    });
  }
  function startUpload() { setUploadAlbum(String(activeAlbum?.id ?? customAlbums[0]?.id ?? "")); setFiles([]); setUploading(true); setError(""); }
  async function upload() {
    await run(async () => {
      if (!uploadAlbum || !files.length) return;
      for (const file of files) {
        const body = new FormData(); body.append("file", file);
        const media = await apiRequest<{ id: string }>("/v1/media", { method: "POST", body });
        await apiRequest("/v1/profiles/me/photo-albums/" + uploadAlbum + "/photos", { method: "POST", body: JSON.stringify({ media_id: media.id, caption: "" }) });
        setFiles((current) => current.filter((item) => item !== file));
      }
      await refresh(); setUploading(false); setAlbumKey(uploadAlbum); setTab("photos"); setView("all");
    });
  }
  async function writePhoto(photo: ProfilePhoto, patch: Partial<ProfilePhoto>) {
    const next = { ...photo, ...patch };
    await apiRequest("/v1/profiles/me/photos/" + photo.id, { method: "PUT", body: JSON.stringify({ caption: next.caption ?? "", archived: Boolean(next.archived), latitude: next.latitude ?? null, longitude: next.longitude ?? null }) });
  }
  async function batch(action: "archive" | "delete") {
    if (action === "delete" && !window.confirm("Удалить выбранные фотографии (" + selection.size + ")?")) return;
    await run(async () => {
      try {
        for (const photo of allPhotos.filter((item) => item.id != null && selection.has(item.id) && item.album_id)) {
          if (action === "delete") await apiRequest("/v1/profiles/me/photos/" + photo.id, { method: "DELETE" });
          else await writePhoto(photo, { archived: view !== "archive" });
          setSelection((current) => { const next = new Set(current); next.delete(photo.id!); return next; });
        }
      } finally { await refresh(); }
    });
  }
  function albumCard(album: ProfilePhotoAlbum) {
    const photos = album.photos.filter((photo) => !photo.archived);
    const cover = photos[photos.length - 1];
    return <button className="gallery-album-card" key={key(album)} onClick={() => { setAlbumKey(key(album)); setTab("photos"); setView("all"); }}>
      <span className="gallery-album-cover">{cover ? <GalleryImage photo={cover} alt="" loading="lazy"/> : <Icon name="photo"/>}{album.visibility === "private" && <span className="gallery-lock" title="Только я"><Icon name="lock"/></span>}</span>
      <span>{album.kind === "avatars" ? "Фото профиля" : album.title}</span><small>{photos.length ? photos.length + " фото" : "Нет фото"}</small>
    </button>;
  }

  return <section className="gallery-page">
    <header className="gallery-header"><h1>{own ? "Мои фотографии" : "Фотографии"}</h1>
      <div className="gallery-toolbar"><nav aria-label="Раздел фотографий"><button aria-pressed={tab === "photos"} onClick={() => { setTab("photos"); setAlbumKey(null); }}>Фото</button><button aria-pressed={tab === "albums"} onClick={() => { setTab("albums"); setAlbumKey(null); setView("all"); }}>Альбомы</button></nav>
        {own && <div className="gallery-actions"><button className="gallery-primary" onClick={() => tab === "albums" ? editAlbum() : startUpload()}><Icon name="photo"/>{tab === "albums" ? "Создать альбом" : "Загрузить фото"}</button>
          <details className="gallery-menu"><summary aria-label="Действия с фотографиями">•••</summary><div>
            <button onClick={(event) => { changeView("archive"); event.currentTarget.closest("details")?.removeAttribute("open"); }}><Icon name="archive"/>Архив</button>
            <button onClick={(event) => { changeView("map"); event.currentTarget.closest("details")?.removeAttribute("open"); }}><Icon name="pin"/>Показать на карте</button>
            <button onClick={(event) => { changeView("comments"); event.currentTarget.closest("details")?.removeAttribute("open"); }}><Icon name="comment"/>Комментарии к фото</button>
            <button onClick={(event) => { setSelecting(true); setTab("photos"); event.currentTarget.closest("details")?.removeAttribute("open"); }}><Icon name="check"/>Выбрать несколько</button>
          </div></details>
          {activeAlbum?.kind === "custom" && <button className="gallery-icon-button" aria-label="Настройки альбома" onClick={() => editAlbum(activeAlbum)}><Icon name="settings"/></button>}
        </div>}
      </div>
    </header>
    {error && !albumForm && !uploading && !opened && <p className="gallery-error" role="alert">{error}</p>}
    <div className="gallery-content" aria-busy={loading || busy}>
      {loading ? <p className="gallery-empty">Загружаем фотографии…</p> : tab === "albums" ? <>
        <div className="gallery-system-albums">{albums.filter((album) => album.kind !== "custom").map(albumCard)}</div>
        <h2>Мои альбомы</h2><div className="gallery-albums">{customAlbums.map(albumCard)}</div>
        {!customAlbums.length && <p className="gallery-empty">{own ? "Создайте альбом для своих фотографий." : "Нет доступных альбомов."}</p>}
      </> : <>
        {(activeAlbum || view !== "all") && <div className="gallery-section-heading"><button onClick={() => { setAlbumKey(null); changeView("all"); }}>← Все фото</button><h2>{activeAlbum?.title ?? ({ archive: "Архив", map: "Фотографии на карте", comments: "Комментарии к фото", all: "" }[view])}</h2></div>}
        {activeAlbum?.description && <p className="gallery-description">{activeAlbum.description}</p>}
        {view === "archive" && <p className="gallery-description">Архивные фотографии скрыты из альбомов и ленты. Их можно восстановить.</p>}
        {view === "comments" && <p className="gallery-description">Откройте фотографию, чтобы прочитать или оставить комментарий.</p>}
        {view === "map" && mapEmbed && <div className="gallery-map"><iframe title="Карта фотографий" src={mapEmbed} loading="lazy" referrerPolicy="no-referrer"/><div>{shownPhotos.map((photo, index) => <button key={photo.id} aria-pressed={photo.id === mapPhoto?.id} onClick={() => setMapPhotoID(photo.id)}>Фото {index + 1}</button>)}</div></div>}
        {selecting && <div className="gallery-selection"><span>Выбрано: {selection.size}</span><button disabled={busy || !selection.size} onClick={() => void batch("archive")}>{view === "archive" ? "Восстановить" : "В архив"}</button><button disabled={busy || !selection.size} onClick={() => void batch("delete")}>Удалить</button><button onClick={() => { setSelecting(false); setSelection(new Set()); }}>Отмена</button></div>}
        {!shownPhotos.length && <div className="gallery-empty"><Icon name="photo"/><h2>{view === "map" ? "Нет фотографий с местоположением" : view === "archive" ? "Архив пуст" : "Фотографий пока нет"}</h2><p>{view === "map" ? "Укажите координаты в настройках фотографии, чтобы увидеть её на карте." : own ? "Загрузите фотографии в альбом — они появятся здесь." : "Здесь появятся доступные фотографии пользователя."}</p></div>}
        {years.map((year) => <section key={year} className="gallery-year"><h2>{year}</h2><div className="gallery-mosaic">{shownPhotos.filter((photo) => photo.created_at.startsWith(year)).map((photo) => <article key={String(photo.album_id ?? "avatars") + "-" + photo.id} className={"gallery-tile " + (selection.has(photo.id!) ? "is-selected" : "")}>
          <button className="gallery-image-button" aria-label={selecting ? "Выбрать фотографию" : "Открыть фотографию"} aria-pressed={selecting ? selection.has(photo.id!) : undefined} disabled={selecting && !photo.album_id} onClick={() => selecting ? setSelection((current) => { const next = new Set(current); if (next.has(photo.id!)) next.delete(photo.id!); else next.add(photo.id!); return next; }) : openPhoto(photo)}><GalleryImage photo={photo} alt={photo.caption ?? "Фотография"} loading="lazy"/>{selecting && photo.album_id && <span className="gallery-photo-badge"><Icon name="check"/></span>}</button>
          {!selecting && own && photo.album_id && <button className="gallery-photo-more" aria-label="Действия с фотографией" onClick={() => openPhoto(photo)}>•••</button>}
          {view === "map" && <a className="gallery-map-link" href={mapURL(photo)} target="_blank" rel="noreferrer"><Icon name="pin"/>Открыть на карте</a>}
        </article>)}</div></section>)}
      </>}
    </div>
    {albumForm && <Modal title={editingAlbum ? "Настройки альбома" : "Создать альбом"} close={() => { if (!busy) setAlbumForm(null); }}><form onSubmit={(event) => { event.preventDefault(); void saveAlbum(); }}><div className="gallery-form-body">
      <label><span>Название <b>*</b><small>{Array.from(albumForm.title).length} / 128</small></span><input required maxLength={128} value={albumForm.title} placeholder="Введите название" onChange={(event) => setAlbumForm({ ...albumForm, title: event.target.value })}/></label>
      <label><span>Описание<small>{Array.from(albumForm.description).length} / 512</small></span><textarea maxLength={512} value={albumForm.description} placeholder="Введите описание" onChange={(event) => setAlbumForm({ ...albumForm, description: event.target.value })}/></label>
      <h3>Настройки приватности</h3><label className="gallery-privacy">Кто может просматривать этот альбом<select value={albumForm.visibility} onChange={(event) => setAlbumForm({ ...albumForm, visibility: event.target.value })}><option value="public">Все пользователи</option><option value="private">Только я</option></select></label>
      <label className="gallery-privacy">Кто может комментировать фото в альбоме<select value={albumForm.comment_policy} onChange={(event) => setAlbumForm({ ...albumForm, comment_policy: event.target.value })}><option value="public">Все пользователи</option><option value="private">Только я</option><option value="nobody">Никто</option></select></label>
      <p className="gallery-description">Изменить настройки приватности можно в любой момент</p>{error && <p className="gallery-error" role="alert">{error}</p>}
    </div><footer><button type="button" disabled={busy} onClick={() => setAlbumForm(null)}>Отмена</button><button className="gallery-primary" disabled={busy || !albumForm.title.trim()}>{busy ? "Сохраняем…" : editingAlbum ? "Сохранить" : "Создать альбом"}</button></footer></form></Modal>}
    {uploading && !albumForm && <Modal title="Загрузить фотографии" close={() => { if (!busy) setUploading(false); }}><div className="gallery-form-body">
      <label>Альбом<select value={uploadAlbum} onChange={(event) => setUploadAlbum(event.target.value)}><option value="">Выберите альбом</option>{customAlbums.map((album) => <option key={album.id} value={String(album.id)}>{album.title}</option>)}</select></label>
      <button disabled={busy} onClick={() => editAlbum()}>Создать альбом</button><label className="gallery-dropzone"><Icon name="photo"/><span>Выберите фотографии</span><small>До 10 файлов, каждый до 25 МБ</small><input type="file" accept="image/*" multiple disabled={busy} onChange={(event) => { const chosen = Array.from(event.target.files ?? []); if (chosen.length > 10 || chosen.some((file) => !file.type.startsWith("image/") || file.size > 25 * 1024 * 1024)) { setError("Выберите до 10 изображений размером до 25 МБ каждое."); event.target.value = ""; return; } setFiles(chosen); setError(""); }}/></label>
      <div className="gallery-upload-previews">{previewURLs.map((url, index) => <img src={url} alt={files[index]?.name ?? ""} key={url}/>)}</div>{error && <p className="gallery-error" role="alert">{error}</p>}
    </div><footer><button disabled={busy} onClick={() => setUploading(false)}>Отмена</button><button className="gallery-primary" disabled={busy || !uploadAlbum || !files.length} onClick={() => void upload()}>{busy ? "Загружаем…" : "Загрузить" + (files.length ? " (" + files.length + ")" : "")}</button></footer></Modal>}
    {opened && <Modal title="Фотография" wide close={() => { if (!busy) setOpened(null); }}><div className="gallery-viewer"><div className="gallery-viewer-image"><GalleryImage photo={opened} alt={opened.caption ?? "Фотография"}/></div><aside>
      <p className="gallery-description">{new Date(opened.created_at).toLocaleDateString("ru-RU", { day: "numeric", month: "long", year: "numeric" })}</p>
      {own && opened.album_id ? <form onSubmit={(event) => { event.preventDefault(); void run(async () => { const patch = { caption, latitude: latitude.trim() ? Number(latitude) : null, longitude: longitude.trim() ? Number(longitude) : null }; await writePhoto(opened, patch); await refresh(); setOpened({ ...opened, ...patch }); }); }}>
        <label>Подпись<textarea maxLength={2000} value={caption} onChange={(event) => setCaption(event.target.value)}/></label><details className="gallery-location"><summary>Местоположение</summary><label>Широта<input type="number" min="-90" max="90" step="any" value={latitude} onChange={(event) => setLatitude(event.target.value)}/></label><label>Долгота<input type="number" min="-180" max="180" step="any" value={longitude} onChange={(event) => setLongitude(event.target.value)}/></label></details>
        <button disabled={busy}>Сохранить</button><div className="gallery-viewer-actions"><button type="button" disabled={busy} onClick={() => void run(async () => { await writePhoto(opened, { archived: !opened.archived }); await refresh(); setOpened(null); })}><Icon name="archive"/>{opened.archived ? "Восстановить" : "В архив"}</button><button type="button" disabled={busy} onClick={() => { if (window.confirm("Удалить фотографию из альбома?")) void run(async () => { await apiRequest("/v1/profiles/me/photos/" + opened.id, { method: "DELETE" }); await refresh(); setOpened(null); }); }}>Удалить</button></div>
      </form> : <p>{opened.caption}</p>}
      {opened.latitude != null && opened.longitude != null && <a href={mapURL(opened)} target="_blank" rel="noreferrer">Показать на карте ↗</a>}
      {opened.album_id && <div className="gallery-comments"><h3>Комментарии</h3>{commentsLoading ? <p>Загрузка…</p> : comments.length ? comments.map((item) => <article key={item.id}><strong>{item.author_name}</strong><p>{item.body}</p><small>{new Date(item.created_at).toLocaleString("ru-RU")}</small></article>) : <p className="gallery-description">Комментариев пока нет</p>}
        {canComment ? <form onSubmit={(event) => { event.preventDefault(); void run(async () => { await apiRequest("/v1/profiles/photos/" + opened.id + "/comments", { method: "POST", body: JSON.stringify({ body: comment }) }); setComment(""); const result = await apiRequest<{ comments: Comment[] }>("/v1/profiles/photos/" + opened.id + "/comments"); setComments(result.comments); }); }}><label>Ваш комментарий<textarea required maxLength={2000} value={comment} onChange={(event) => setComment(event.target.value)}/></label><button disabled={busy || !comment.trim()}>Отправить</button></form> : <p className="gallery-description">Комментирование ограничено владельцем альбома</p>}
      </div>}{error && <p className="gallery-error" role="alert">{error}</p>}
    </aside></div></Modal>}
  </section>;
}
