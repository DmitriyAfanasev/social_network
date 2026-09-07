import { useEffect, useMemo, useRef, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";

import { apiRequest } from "../../../../shared/api/http";

type CreatePostPanelProps = {
  onCreated: () => Promise<void>;
};

export function CreatePostPanel({ onCreated }: CreatePostPanelProps) {
  const [content, setContent] = useState("");
  const [attachment, setAttachment] = useState<File | null>(null);
  const [videoUploading, setVideoUploading] = useState(false);
  const [videoStatus, setVideoStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const photoInputRef = useRef<HTMLInputElement | null>(null);
  const attachmentMenuRef = useRef<HTMLDetailsElement | null>(null);

  const attachmentPreviewUrl = useMemo(() => {
    if (!attachment || !attachment.type.startsWith("image/")) {
      return null;
    }

    return URL.createObjectURL(attachment);
  }, [attachment]);

  const videoPreviewUrl = useMemo(() => {
    if (!attachment?.type.startsWith("video/")) {
      return null;
    }
    return URL.createObjectURL(attachment);
  }, [attachment]);

  useEffect(() => {
    return () => {
      if (attachmentPreviewUrl) {
        URL.revokeObjectURL(attachmentPreviewUrl);
      }
      if (videoPreviewUrl) {
        URL.revokeObjectURL(videoPreviewUrl);
      }
    };
  }, [attachmentPreviewUrl, videoPreviewUrl]);

  function selectAttachment(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null;
    setError(null);
    setAttachment(file);
    setExpanded(true);
    setVideoStatus(null);
    if (attachmentMenuRef.current) {
      attachmentMenuRef.current.open = false;
    }
  }

  function clearAttachment() {
    setAttachment(null);
    setVideoStatus(null);

    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }

    if (photoInputRef.current) {
      photoInputRef.current.value = "";
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    const trimmedContent = content.trim();

    if (!trimmedContent && !attachment) {
      setError("Добавьте текст или файл.");
      return;
    }

    setError(null);
    setLoading(true);

    try {
      let mediaIds: string[] = [];
      if (attachment?.type.startsWith("video/")) {
        setVideoUploading(true);
        const videoBody = new FormData();
        videoBody.append("file", attachment);
        const video = await apiRequest<{ media_id: string; status: string }>("/videos", {
          method: "POST",
          body: videoBody,
        });
        mediaIds = [video.media_id];
        setVideoStatus(`Видео прикреплено · обработка ${video.status}`);
      }

      if (attachment && !attachment.type.startsWith("video/")) {
        const mediaBody = new FormData();
        mediaBody.append("file", attachment);
        const media = await apiRequest<{ id: string }>("/v1/media", {
          method: "POST",
          body: mediaBody,
        });
        mediaIds = [media.id];
      }

      await apiRequest("/v1/content/posts", {
        method: "POST",
        body: JSON.stringify({ body: trimmedContent, media_ids: mediaIds }),
      });
      setContent("");
      clearAttachment();
      setExpanded(false);
      setVideoStatus(null);
      await onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось создать пост");
    } finally {
      setVideoUploading(false);
      setLoading(false);
    }
  }

  return (
    <section className={expanded || attachment ? "panel composer composer-expanded" : "panel composer composer-collapsed"}>
      <form onSubmit={submit}>
        {expanded || content || attachment ? (
          <textarea
            autoFocus={expanded && !content}
            value={content}
            onChange={(event) => setContent(event.target.value)}
            placeholder="Что у вас нового?"
            rows={3}
          />
        ) : (
          <button type="button" className="composer-placeholder" onClick={() => setExpanded(true)}>Что у вас нового?</button>
        )}
        {attachment && (
          <div className="attachment-preview compact">
            {attachment?.type.startsWith("video/") ? (
              <video src={videoPreviewUrl ?? undefined} controls muted />
            ) : attachmentPreviewUrl ? (
              <img src={attachmentPreviewUrl} alt="" />
            ) : (
              <span className="file-preview">{attachment.name}</span>
            )}
            <div className="attachment-meta">
              <strong>{attachment.name}</strong>
              <span>{attachment.type || "Файл"}</span>
            </div>
            <button type="button" className="icon-button text-button" onClick={clearAttachment}>
              Убрать
            </button>
          </div>
        )}
        <div className="composer-actions">
          <div className="composer-tools">
            <details ref={attachmentMenuRef} className="attachment-menu">
              <summary className="icon-only-button" aria-label="Добавить вложение" title="Добавить вложение">
                <span aria-hidden="true">📎</span>
              </summary>
              <div className="floating-menu attachment-menu-panel" aria-label="Добавить вложение">
                <label className="menu-file-action">
                  <span aria-hidden="true">▧</span>
                  <span>Фото</span>
                  <input ref={photoInputRef} type="file" accept="image/*" onChange={selectAttachment} />
                </label>
                <label className="menu-file-action">
                  <span aria-hidden="true">▶</span>
                  <span>Видео</span>
                  <input type="file" accept="video/*" onChange={selectAttachment} />
                </label>
                <label className="menu-file-action">
                  <span aria-hidden="true">▤</span>
                  <span>Файл</span>
                  <input ref={fileInputRef} type="file" onChange={selectAttachment} />
                </label>
              </div>
            </details>
            {attachment && (
              <span className="attachment-name" title={attachment.name}>
                {attachment.name}
              </span>
            )}
          </div>
          {expanded && <button disabled={loading || videoUploading || (!content.trim() && !attachment)}>
            {loading ? (attachment?.type.startsWith("video/") ? "Загружаем видео..." : "Публикуем...") : "Опубликовать"}
          </button>}
        </div>
        {videoStatus && <p className="muted">{videoStatus}</p>}
        {error && <p className="error">{error}</p>}
      </form>
    </section>
  );
}
