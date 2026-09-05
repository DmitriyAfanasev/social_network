import { useEffect, useMemo, useRef, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";

import { UserAvatar } from "../../user/ui/UserAvatar";
import type { Comment, CommentsPageResponse, Post, PostResponse } from "../model/post";
import { apiRequest } from "../../../shared/api/http";
import { getFileName, isImageAttachment, resolveAttachmentUrl } from "../../../shared/lib/attachments";
import { API_BASE_URL } from "../../../shared/config/api";
import { formatDate } from "../../../shared/lib/date";
import { navigate } from "../../../shared/lib/navigation";
import { getUserName, userFromPublicProfile, type PublicProfileResponse, type User } from "../../user/model/user";

type PostItemProps = {
  post: Post;
  canLike?: boolean;
  currentUserId?: string | number;
  onLike?: () => void;
  onChanged?: () => Promise<void>;
};

type CommentNode = Comment & {
  replies: CommentNode[];
  depth: number;
  replyToName?: string;
};

type MediaAttachment = {
  id: string;
  original_filename?: string;
  content_type?: string;
  media_type?: string;
};

export function PostItem({ post, canLike, currentUserId, onLike, onChanged }: PostItemProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState(post.content ?? "");
  const [editAttachment, setEditAttachment] = useState<File | null>(null);
  const [editError, setEditError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [removingAttachment, setRemovingAttachment] = useState(false);
  const [likesOpen, setLikesOpen] = useState(false);
  const [commentsOpen, setCommentsOpen] = useState(false);
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentsOffset, setCommentsOffset] = useState(0);
  const [commentsHasMore, setCommentsHasMore] = useState(false);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [showAllComments, setShowAllComments] = useState(false);
  const [commentText, setCommentText] = useState("");
  const [replyToCommentId, setReplyToCommentId] = useState<string | number | null>(null);
  const [replyText, setReplyText] = useState("");
  const [editingCommentId, setEditingCommentId] = useState<string | number | null>(null);
  const [editingCommentText, setEditingCommentText] = useState("");
  const [deletingCommentId, setDeletingCommentId] = useState<string | number | null>(null);
  const [commentSaving, setCommentSaving] = useState(false);
  const [commentError, setCommentError] = useState<string | null>(null);
  const [mediaAttachments, setMediaAttachments] = useState<MediaAttachment[]>([]);
  const [resolvedAuthor, setResolvedAuthor] = useState<User | null>(post.author ?? null);
  const [likedUsers, setLikedUsers] = useState<User[]>(post.liked_users ?? []);
  const [likesLoading, setLikesLoading] = useState(false);
  const editPhotoInputRef = useRef<HTMLInputElement | null>(null);
  const editFileInputRef = useRef<HTMLInputElement | null>(null);
  const isOwner = currentUserId === post.author_id;
  const canComment = Boolean(currentUserId);
  const author = resolvedAuthor ?? post.author;
  const authorName = author ? getUserName(author) : `user #${post.author_id}`;
  const likePreviewUsers = likedUsers.slice(0, 3);
  const hasLikes = post.likes_count > 0 || (post.liked_user_ids ?? []).length > 0;

  const commentTree = useMemo(() => buildCommentTree(comments), [comments]);
  const visibleCommentTree = useMemo(
    () => showAllComments ? commentTree : commentTree.slice(0, 3),
    [commentTree, showAllComments],
  );

  const editAttachmentPreviewUrl = useMemo(() => {
    if (!editAttachment || !editAttachment.type.startsWith("image/")) {
      return null;
    }

    return URL.createObjectURL(editAttachment);
  }, [editAttachment]);

  useEffect(() => {
    setEditContent(post.content ?? "");
    setEditAttachment(null);
    setEditError(null);
  }, [post.id, post.content, post.image, post.media_ids?.join(",")]);

  useEffect(() => {
    const mediaIDs = post.media_ids ?? [];
    let cancelled = false;
    if (mediaIDs.length === 0) {
      setMediaAttachments([]);
      return () => {
        cancelled = true;
      };
    }

    void Promise.all(mediaIDs.map(async (mediaID) => {
      try {
        return await apiRequest<MediaAttachment>(`/v1/media/${mediaID}`);
      } catch {
        return { id: mediaID };
      }
    })).then((attachments) => {
      if (!cancelled) setMediaAttachments(attachments);
    });

    return () => {
      cancelled = true;
    };
  }, [post.id, post.media_ids?.join(",")]);

  useEffect(() => {
    let cancelled = false;
    if (post.author) {
      setResolvedAuthor(post.author);
      return () => {
        cancelled = true;
      };
    }

    void apiRequest<PublicProfileResponse>(`/v1/profiles/${post.author_id}`)
      .then((profile) => {
        if (!cancelled) {
          setResolvedAuthor(userFromPublicProfile(profile));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setResolvedAuthor(null);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [post.id, post.author_id, post.author?.id]);

  useEffect(() => {
    const likedUserIDs = post.liked_user_ids ?? [];
    const knownUsers = post.liked_users ?? [];
    let cancelled = false;

    if (likedUserIDs.length === 0) {
      setLikedUsers(knownUsers);
      setLikesLoading(false);
      return () => {
        cancelled = true;
      };
    }

    setLikesLoading(true);
    const userIDsToLoad = likesOpen ? likedUserIDs : likedUserIDs.slice(0, 3);
    void Promise.all(userIDsToLoad.map(async (userID) => {
      try {
        const profile = await apiRequest<PublicProfileResponse>(`/v1/profiles/${userID}`);
        return userFromPublicProfile(profile);
      } catch {
        return knownUsers.find((user) => user.id === userID) ?? null;
      }
    })).then((users) => {
      if (!cancelled) {
        setLikedUsers(users.filter((user): user is User => user !== null));
        setLikesLoading(false);
      }
    });

    return () => {
      cancelled = true;
    };
  }, [post.id, post.liked_user_ids?.join(","), likesOpen]);

  useEffect(() => {
    setComments([]);
    setCommentsOffset(0);
    setCommentsHasMore(false);
    setCommentText("");
    setReplyToCommentId(null);
    setReplyText("");
    setEditingCommentId(null);
    setEditingCommentText("");
    setDeletingCommentId(null);
    setCommentError(null);
    setLikesOpen(false);
    setShowAllComments(false);
    setCommentsOpen(post.comments_count > 0);
    if (post.comments_count > 0) void loadComments();
  }, [post.id, post.comments_count]);

  useEffect(() => {
    return () => {
      if (editAttachmentPreviewUrl) {
        URL.revokeObjectURL(editAttachmentPreviewUrl);
      }
    };
  }, [editAttachmentPreviewUrl]);

  function startEditing() {
    setEditContent(post.content ?? "");
    setEditAttachment(null);
    setEditError(null);
    setIsEditing(true);
  }

  function cancelEditing() {
    setIsEditing(false);
    setEditContent(post.content ?? "");
    setEditAttachment(null);
    setEditError(null);
  }

  function selectEditAttachment(event: ChangeEvent<HTMLInputElement>) {
    setEditError(null);
    setEditAttachment(event.target.files?.[0] ?? null);
  }

  function clearEditAttachment() {
    setEditAttachment(null);

    if (editPhotoInputRef.current) {
      editPhotoInputRef.current.value = "";
    }

    if (editFileInputRef.current) {
      editFileInputRef.current.value = "";
    }
  }

  async function updateComment(event: FormEvent, commentId: string | number) {
    event.preventDefault();
    const content = editingCommentText.trim();

    if (!content) {
      setCommentError("Напишите комментарий.");
      return;
    }

    setCommentSaving(true);
    setCommentError(null);

    try {
      await apiRequest<Comment>(`/comments/${commentId}`, {
        method: "PATCH",
        body: JSON.stringify({ content }),
      });
      setEditingCommentId(null);
      setEditingCommentText("");
      await loadComments();
      await onChanged?.();
    } catch (err) {
      setCommentError(err instanceof Error ? err.message : "Не удалось сохранить комментарий");
    } finally {
      setCommentSaving(false);
    }
  }

  async function deleteComment(commentId: string | number) {
    if (!window.confirm("Удалить комментарий?")) {
      return;
    }

    setDeletingCommentId(commentId);
    setCommentError(null);

    try {
      await apiRequest<void>(`/comments/${commentId}`, { method: "DELETE" });
      if (replyToCommentId === commentId) {
        setReplyToCommentId(null);
        setReplyText("");
      }
      if (editingCommentId === commentId) {
        setEditingCommentId(null);
        setEditingCommentText("");
      }
      await loadComments();
      await onChanged?.();
    } catch (err) {
      setCommentError(err instanceof Error ? err.message : "Не удалось удалить комментарий");
    } finally {
      setDeletingCommentId(null);
    }
  }

  async function likeComment(commentId: string | number) {
    if (!currentUserId) return;
    try {
      const result = await apiRequest<{ likes_count: number; liked: boolean }>(`/comments/${commentId}/likes`, {
        method: "POST",
      });
      setComments((current) => current.map((comment) => (
        comment.id === commentId
          ? { ...comment, likes_count: result.likes_count, is_liked_by_current: result.liked }
          : comment
      )));
      await onChanged?.();
    } catch (err) {
      setCommentError(err instanceof Error ? err.message : "Не удалось поставить лайк");
    }
  }

  async function saveEdit(event: FormEvent) {
    event.preventDefault();
    const trimmedContent = editContent.trim();

    if (!trimmedContent && !editAttachment && !post.image && !(post.media_ids?.length ?? 0)) {
      setEditError("Добавьте текст или файл.");
      return;
    }

    setSaving(true);
    setEditError(null);

    try {
      await apiRequest<PostResponse>(`/posts/${post.id}`, {
        method: "PATCH",
        body: JSON.stringify({ body: trimmedContent, media_ids: post.media_ids ?? [] }),
      });
      setIsEditing(false);
      await onChanged?.();
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Не удалось сохранить пост");
    } finally {
      setSaving(false);
    }
  }

  async function deletePost() {
    if (!window.confirm("Удалить пост?")) {
      return;
    }

    setDeleting(true);

    try {
      await apiRequest<{ message: string }>(`/posts/${post.id}`, { method: "DELETE" });
      await onChanged?.();
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Не удалось удалить пост");
    } finally {
      setDeleting(false);
    }
  }

  async function removeExistingAttachment() {
    setRemovingAttachment(true);
    setEditError(null);

    try {
      await apiRequest<{ message: string }>(`/posts/${post.id}/image`, { method: "DELETE" });
      await onChanged?.();
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Не удалось удалить вложение");
    } finally {
      setRemovingAttachment(false);
    }
  }

  async function loadComments(offset = 0) {
    setCommentsLoading(true);
    setCommentError(null);

    try {
      const result = await apiRequest<CommentsPageResponse>(`/posts/${post.id}/comments?offset=${offset}&limit=3`);
      setComments((current) => (offset === 0 ? result.comments : [...current, ...result.comments]));
      setCommentsOffset(offset + result.comments.length);
      setCommentsHasMore(result.has_more);
    } catch (err) {
      setCommentError(err instanceof Error ? err.message : "Не удалось загрузить комментарии");
    } finally {
      setCommentsLoading(false);
    }
  }

  function toggleComments() {
    const nextOpen = !commentsOpen;
    setCommentsOpen(nextOpen);

    if (nextOpen && comments.length === 0) {
      void loadComments();
    }
  }

  async function submitComment(event: FormEvent, parentId?: number) {
    event.preventDefault();
    const isReply = parentId !== undefined;
    const content = (isReply ? replyText : commentText).trim();

    if (!content) {
      setCommentError(isReply ? "Напишите ответ." : "Напишите комментарий.");
      return;
    }

    setCommentSaving(true);
    setCommentError(null);

    try {
      await apiRequest<Comment>(`/posts/${post.id}/comments`, {
        method: "POST",
        body: JSON.stringify({ body: content }),
      });
      if (isReply) {
        setReplyText("");
        setReplyToCommentId(null);
      } else {
        setCommentText("");
      }
      setCommentsOpen(true);
      setShowAllComments(true);
      await loadComments();
      await onChanged?.();
    } catch (err) {
      setCommentError(err instanceof Error ? err.message : "Не удалось отправить комментарий");
    } finally {
      setCommentSaving(false);
    }
  }

  return (
    <article className="post">
      <div className="post-header">
        <button type="button" className="avatar-link" onClick={() => navigate(`/profile/${post.author_id}`)}>
          <UserAvatar user={author} size="sm" />
        </button>
        <div>
          <button type="button" className="user-name-link" onClick={() => navigate(`/profile/${post.author_id}`)}>
            {authorName}
          </button>
          <span>{formatDate(post.created_at)}</span>
        </div>
        {isOwner && !isEditing && (
          <details className="post-actions-menu">
            <summary className="icon-only-button" aria-label="Действия с постом" title="Действия с постом">
              <span aria-hidden="true">⋯</span>
            </summary>
            <div className="floating-menu post-actions-panel">
              <button type="button" className="menu-action" onClick={startEditing}>
                Изменить
              </button>
              <button type="button" className="menu-action danger" disabled={deleting} onClick={deletePost}>
                {deleting ? "Удаляем..." : "Удалить"}
              </button>
            </div>
          </details>
        )}
      </div>
      {isEditing ? (
        <form className="post-edit-form" onSubmit={saveEdit}>
          <textarea value={editContent} onChange={(event) => setEditContent(event.target.value)} rows={4} />
          {editAttachment && (
            <div className="attachment-preview compact">
              {editAttachmentPreviewUrl ? (
                <img src={editAttachmentPreviewUrl} alt="" />
              ) : (
                <span className="file-preview">{editAttachment.name}</span>
              )}
              <div className="attachment-meta">
                <strong>{editAttachment.name}</strong>
                <span>{editAttachment.type || "Файл"}</span>
              </div>
              <button type="button" className="icon-button text-button" onClick={clearEditAttachment}>
                Убрать
              </button>
            </div>
          )}
          {post.image && !editAttachment && (
            <div className="current-attachment">
              {(post.image_content_type?.startsWith("image/") || isImageAttachment(post.image)) && (
                <img className="post-media compact" src={resolveAttachmentUrl(post.image)} alt="" />
              )}
              <span>Текущее вложение: {getFileName(post.image)}</span>
              <button
                type="button"
                className="icon-button text-button danger"
                disabled={removingAttachment}
                onClick={removeExistingAttachment}
              >
                {removingAttachment ? "Удаляем..." : "Удалить файл"}
              </button>
            </div>
          )}
          <div className="post-edit-actions">
            <div className="composer-tools">
              <label className="icon-button file-action" title="Заменить фото">
                <span>Фото</span>
                <input ref={editPhotoInputRef} type="file" accept="image/*" onChange={selectEditAttachment} />
              </label>
              <label className="icon-button file-action" title="Заменить файл">
                <span>Файл</span>
                <input ref={editFileInputRef} type="file" onChange={selectEditAttachment} />
              </label>
            </div>
            <div className="post-edit-actions">
              <button type="submit" disabled={saving}>
                {saving ? "Сохраняем..." : "Сохранить"}
              </button>
              <button type="button" className="secondary" onClick={cancelEditing}>
                Отмена
              </button>
            </div>
          </div>
          {editError && <p className="error">{editError}</p>}
        </form>
      ) : (
        <>
          {post.content && <p>{post.content}</p>}
          {mediaAttachments.length > 0 ? (
            <div className="post-media-list">
              {mediaAttachments.map((attachment) => {
                const contentType = attachment.content_type ?? "";
                const mediaURL = resolveAttachmentUrl(`/v1/media/${attachment.id}/content`);
                const filename = attachment.original_filename || `Медиафайл ${attachment.id.slice(0, 8)}`;
                if (contentType.startsWith("image/")) {
                  return <img className="post-media" key={attachment.id} src={mediaURL} alt={filename} />;
                }
                if (contentType.startsWith("video/")) {
                  return <video className="post-video" key={attachment.id} src={mediaURL} controls preload="metadata" />;
                }
                if (contentType.startsWith("audio/")) {
                  return <div className="post-audio" key={attachment.id}><span aria-hidden="true">♫</span><strong>{filename}</strong><audio src={mediaURL} controls preload="metadata" /></div>;
                }
                return <a className="file-attachment" key={attachment.id} href={mediaURL} target="_blank" rel="noreferrer">▤ {filename}</a>;
              })}
            </div>
          ) : post.image &&
            (post.image_content_type?.startsWith("image/") || isImageAttachment(post.image) ? (
              <img className="post-media" src={resolveAttachmentUrl(post.image)} alt="" />
            ) : (
              <a className="file-attachment" href={resolveAttachmentUrl(post.image)} target="_blank" rel="noreferrer">
                {getFileName(post.image)}
              </a>
            ))}
        </>
      )}
      {editError && !isEditing && <p className="error">{editError}</p>}
      {post.featured_comment && !commentsOpen && (
        <div className="featured-comment">
          <div className="featured-comment-heading">
            <span className="eyebrow">Популярный комментарий</span>
            <span className="muted">♥ {post.featured_comment.likes_count}</span>
          </div>
          <div className="featured-comment-content">
            <UserAvatar user={post.featured_comment.author} size="sm" />
            <div>
              <strong>{getUserName(post.featured_comment.author)}</strong>
              <p>{post.featured_comment.text}</p>
            </div>
          </div>
        </div>
      )}
      <div className="post-stats">
        <div className="like-group">
          <button
            type="button"
            className={post.is_liked_by_current ? "like-button active" : "like-button"}
            disabled={!canLike}
            onClick={onLike}
            aria-label={post.is_liked_by_current ? "Убрать лайк" : "Поставить лайк"}
            title={post.is_liked_by_current ? "Убрать лайк" : "Поставить лайк"}
          >
            <svg className="like-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 20.3 4.8 13.6A4.8 4.8 0 0 1 12 7.2a4.8 4.8 0 0 1 7.2 6.4L12 20.3Z" /></svg>
            <span className="like-count">{post.likes_count}</span>
          </button>
          {hasLikes && (
            <div className="like-preview">
              {likePreviewUsers.length > 0 && (
                <div className="like-preview-users" aria-label="Кто поставил лайк">
                  {likePreviewUsers.map((user) => (
                    <button
                      key={user.id}
                      type="button"
                      aria-label={getUserName(user)}
                      title={getUserName(user)}
                      onClick={() => navigate(`/profile/${user.id}`)}
                    >
                      <UserAvatar user={user} size="sm" />
                    </button>
                  ))}
                </div>
              )}
              <button type="button" className="text-button" onClick={() => setLikesOpen(true)}>
                Посмотреть всех{post.likes_count > 0 ? ` (${post.likes_count})` : ""}
              </button>
            </div>
          )}
        </div>
        <button type="button" className="ghost-button" onClick={toggleComments}>
          {post.comments_count} комментариев
        </button>
      </div>
      {likesOpen && (
        <div className="liked-users">
          <div className="liked-users-heading">
            <strong>Кто поставил лайк</strong>
            <button type="button" className="text-button" onClick={() => setLikesOpen(false)}>Закрыть</button>
          </div>
          {likedUsers.length > 0 ? (
            likedUsers.map((user) => (
              <button
                key={user.id}
                type="button"
                className="liked-user"
                onClick={() => navigate(`/profile/${user.id}`)}
              >
                <UserAvatar user={user} size="sm" />
                <span>{getUserName(user)}</span>
              </button>
            ))
          ) : likesLoading ? (
            <p className="muted">Загружаем список...</p>
          ) : (
            <p className="muted">Не удалось загрузить список пользователей.</p>
          )}
        </div>
      )}
      {commentsOpen && (
        <section className="comments-block">
          {commentError && <p className="error">{commentError}</p>}
          <div className="comment-list">
            {visibleCommentTree.map((comment) => (
              <CommentTreeItem
                key={comment.id}
                comment={comment}
                currentUserId={currentUserId}
                canReply={canComment}
                replyToCommentId={replyToCommentId}
                replyText={replyText}
                editingCommentId={editingCommentId}
                editingCommentText={editingCommentText}
                deletingCommentId={deletingCommentId}
                commentSaving={commentSaving}
                onStartReply={(commentId) => {
                  setReplyToCommentId(commentId);
                  setReplyText("");
                  setEditingCommentId(null);
                  setEditingCommentText("");
                  setCommentError(null);
                }}
                onCancelReply={() => {
                  setReplyToCommentId(null);
                  setReplyText("");
                }}
                onReplyTextChange={setReplyText}
                onSubmitReply={submitComment}
                onStartEdit={(comment) => {
                  setEditingCommentId(comment.id);
                  setEditingCommentText(comment.text);
                  setReplyToCommentId(null);
                  setReplyText("");
                  setCommentError(null);
                }}
                onCancelEdit={() => {
                  setEditingCommentId(null);
                  setEditingCommentText("");
                }}
                onEditTextChange={setEditingCommentText}
                onSubmitEdit={updateComment}
                onDelete={(commentId) => void deleteComment(commentId)}
                canModerate={isOwner}
                onLike={(commentId) => void likeComment(commentId)}
              />
            ))}
          </div>
          {commentsLoading && <p className="muted">Загружаем комментарии...</p>}
          {!commentsLoading && comments.length === 0 && !canComment && <p className="muted">Комментариев пока нет.</p>}
          {!showAllComments && (comments.length > 3 || post.comments_count > 3) && (
            <button type="button" className="secondary comments-more" onClick={() => setShowAllComments(true)}>
              Показать все комментарии
            </button>
          )}
          {showAllComments && commentsHasMore && (
            <button type="button" className="secondary comments-more" onClick={() => void loadComments(commentsOffset)}>
              Показать ещё
            </button>
          )}
          {canComment ? (
            <form className="comment-form" onSubmit={(event) => void submitComment(event)}>
              <textarea
                value={commentText}
                onChange={(event) => setCommentText(event.target.value)}
                placeholder="Написать комментарий"
                rows={3}
              />
              <div className="comment-form-actions">
                <button type="submit" disabled={commentSaving}>
                  {commentSaving ? "Отправляем..." : "Комментировать"}
                </button>
              </div>
            </form>
          ) : (
            <p className="muted">Войдите, чтобы написать комментарий.</p>
          )}
        </section>
      )}
    </article>
  );
}

function buildCommentTree(comments: Comment[]): CommentNode[] {
  const nodes = new Map<string | number, CommentNode>();
  const roots: CommentNode[] = [];

  for (const comment of comments) {
    nodes.set(comment.id, { ...comment, replies: [], depth: 0 });
  }

  function resolveDepth(commentId: string | number, visited = new Set<string | number>()): number {
    const node = nodes.get(commentId);
    if (!node?.parent_id || !nodes.has(node.parent_id) || visited.has(commentId)) {
      return 0;
    }
    visited.add(commentId);
    return Math.min(2, resolveDepth(node.parent_id, visited) + 1);
  }

  for (const node of nodes.values()) {
    node.depth = resolveDepth(node.id);
  }

  for (const node of nodes.values()) {
    if (node.parent_id && nodes.has(node.parent_id)) {
      node.replyToName = getUserName(nodes.get(node.parent_id)?.author);
      nodes.get(node.parent_id)?.replies.push(node);
    } else {
      roots.push(node);
    }
  }

  return roots;
}

function isCommentEdited(comment: Comment): boolean {
  return new Date(comment.updated_at).getTime() > new Date(comment.created_at).getTime();
}

type CommentTreeItemProps = {
  comment: CommentNode;
  currentUserId?: string | number;
  canReply: boolean;
  replyToCommentId: string | number | null;
  replyText: string;
  editingCommentId: string | number | null;
  editingCommentText: string;
  deletingCommentId: string | number | null;
  commentSaving: boolean;
  onStartReply: (commentId: number) => void;
  onCancelReply: () => void;
  onReplyTextChange: (value: string) => void;
  onSubmitReply: (event: FormEvent, parentId: number) => Promise<void>;
  onStartEdit: (comment: CommentNode) => void;
  onCancelEdit: () => void;
  onEditTextChange: (value: string) => void;
  onSubmitEdit: (event: FormEvent, commentId: number) => Promise<void>;
  onDelete: (commentId: number) => void;
  canModerate: boolean;
  onLike: (commentId: number) => void;
};

function CommentTreeItem({
  comment,
  currentUserId,
  canReply,
  replyToCommentId,
  replyText,
  editingCommentId,
  editingCommentText,
  deletingCommentId,
  commentSaving,
  onStartReply,
  onCancelReply,
  onReplyTextChange,
  onSubmitReply,
  onStartEdit,
  onCancelEdit,
  onEditTextChange,
  onSubmitEdit,
  onDelete,
  canModerate,
  onLike,
}: CommentTreeItemProps) {
  const actionsMenuRef = useRef<HTMLDetailsElement | null>(null);
  const isReplying = replyToCommentId === comment.id;
  const isEditing = editingCommentId === comment.id;
  const canManage = canModerate || currentUserId === comment.user_id;

  useEffect(() => {
    function closeMenu(event: MouseEvent) {
      if (actionsMenuRef.current && !actionsMenuRef.current.contains(event.target as Node)) {
        actionsMenuRef.current.open = false;
      }
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        actionsMenuRef.current?.removeAttribute("open");
      }
    }

    document.addEventListener("click", closeMenu);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("click", closeMenu);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, []);

  return (
    <article className="comment">
      <button type="button" className="avatar-link" onClick={() => navigate(`/profile/${comment.user_id}`)}>
        <UserAvatar user={comment.author} size="sm" />
      </button>
      <div className="comment-body">
        <header>
          <button type="button" className="user-name-link" onClick={() => navigate(`/profile/${comment.user_id}`)}>
            {getUserName(comment.author)}
          </button>
          <span>{formatDate(comment.created_at)}</span>
          {isCommentEdited(comment) && <span>изменён</span>}
          {canManage && !isEditing && (
            <details ref={actionsMenuRef} className="comment-actions-menu">
              <summary className="comment-actions-trigger" aria-label="Действия с комментарием" title="Действия">
                <span aria-hidden="true">⋯</span>
              </summary>
              <div className="floating-menu comment-actions-panel">
                <button
                  type="button"
                  className="menu-action danger"
                  disabled={deletingCommentId === comment.id}
                  onClick={() => { actionsMenuRef.current?.removeAttribute("open"); onDelete(comment.id); }}
                >
                  {deletingCommentId === comment.id ? "Удаляем..." : "Удалить"}
                </button>
              </div>
            </details>
          )}
        </header>
        {isEditing ? (
          <form className="comment-form reply" onSubmit={(event) => void onSubmitEdit(event, comment.id)}>
            <textarea
              value={editingCommentText}
              onChange={(event) => onEditTextChange(event.target.value)}
              rows={2}
            />
            <div className="comment-form-actions">
              <button type="submit" disabled={commentSaving}>
                {commentSaving ? "Сохраняем..." : "Сохранить"}
              </button>
              <button type="button" className="secondary" onClick={onCancelEdit}>
                Отмена
              </button>
            </div>
          </form>
        ) : (
          <>
            {comment.replyToName && <div className="comment-reply-context"><span>ответ</span> <strong>@{comment.replyToName}</strong></div>}
            <p>{comment.text}</p>
          </>
        )}
        {canReply && !isEditing && (
          <button type="button" className="comment-reply-button" onClick={() => onStartReply(comment.id)}>
            Ответить
          </button>
        )}
        {isReplying && (
          <form className="comment-form reply" onSubmit={(event) => void onSubmitReply(event, comment.id)}>
            <textarea
              value={replyText}
              onChange={(event) => onReplyTextChange(event.target.value)}
              placeholder={`Ответить ${getUserName(comment.author)}`}
              rows={2}
            />
            <div className="comment-form-actions">
              <button type="submit" disabled={commentSaving}>
                {commentSaving ? "Отправляем..." : "Ответить"}
              </button>
              <button type="button" className="secondary" onClick={onCancelReply}>
                Отмена
              </button>
            </div>
          </form>
        )}
        {comment.replies.length > 0 && (
          <div className="comment-replies">
            {comment.replies.map((reply) => (
              <CommentTreeItem
                key={reply.id}
                comment={reply}
                currentUserId={currentUserId}
                canReply={canReply && comment.depth < 2}
                replyToCommentId={replyToCommentId}
                replyText={replyText}
                editingCommentId={editingCommentId}
                editingCommentText={editingCommentText}
                deletingCommentId={deletingCommentId}
                commentSaving={commentSaving}
                onStartReply={onStartReply}
                onCancelReply={onCancelReply}
                onReplyTextChange={onReplyTextChange}
                onSubmitReply={onSubmitReply}
                onStartEdit={onStartEdit}
                onCancelEdit={onCancelEdit}
                onEditTextChange={onEditTextChange}
                onSubmitEdit={onSubmitEdit}
                onDelete={onDelete}
                canModerate={canModerate}
                onLike={onLike}
              />
            ))}
          </div>
        )}
      </div>
    </article>
  );
}
