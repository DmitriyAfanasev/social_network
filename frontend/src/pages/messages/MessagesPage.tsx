import { useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent, KeyboardEvent } from "react";

import type { ProfileResponse } from "../../entities/post/model/post";
import type { CallEvent, Conversation, ConversationView, Message, MessagePage } from "../../entities/message/model/message";
import { getUserName } from "../../entities/user/model/user";
import { UserAvatar } from "../../entities/user/ui/UserAvatar";
import { apiRequest, getAccessToken } from "../../shared/api/http";
import { API_BASE_URL } from "../../shared/config/api";
import { navigate, usePath } from "../../shared/lib/navigation";

function getWebSocketUrl() {
  const url = new URL(API_BASE_URL);
  url.protocol = "ws:";
  const token = getAccessToken();
  return `${url.toString().replace(/\/$/, "")}/v1/messaging/ws?access_token=${encodeURIComponent(token ?? "")}`;
}

/*
 * Calls use a separate WebSocket from messages on purpose. The Go socket
 * is responsible for persisted chat events, while the Go service owns the
 * short-lived call state and signaling fan-out. Keeping two channels makes the
 * failure domains explicit: a slow message consumer cannot block an SDP/ICE
 * exchange, and the call service can be scaled independently.
 */
function getCallWebSocketUrl() {
  const configured = import.meta.env.VITE_CALL_SIGNALING_URL as string | undefined;
  const token = getAccessToken();
  if (configured) return `${configured}${configured.includes("?") ? "&" : "?"}access_token=${encodeURIComponent(token ?? "")}`;
  const url = new URL(API_BASE_URL);
  url.protocol = "ws:";
  return `${url.toString().replace(/\/$/, "")}/v1/calls/ws?access_token=${encodeURIComponent(token ?? "")}`;
}

/*
 * STUN/TURN configuration belongs to the browser because the browser creates
 * the RTCPeerConnection. STUN helps ICE discover a usable public mapping; TURN
 * is a relay fallback for networks where a direct peer-to-peer path is
 * impossible. The Go signaling service only transports these settings and
 * candidate messages; it never sees the media packets.
 */
function getIceServers(): RTCIceServer[] {
  const fallback: RTCIceServer[] = [{ urls: "stun:stun.l.google.com:19302" }];
  const configured = import.meta.env.VITE_WEBRTC_ICE_SERVERS as string | undefined;
  if (!configured) return fallback;
  try {
    const parsed = JSON.parse(configured) as RTCIceServer[];
    return Array.isArray(parsed) && parsed.length > 0 ? parsed : fallback;
  } catch {
    return fallback;
  }
}

function formatMessageTime(value: string) {
  return new Date(value).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

function formatMessageDay(value: string) {
  return new Date(value).toLocaleDateString("ru-RU", { day: "numeric", month: "long", year: "numeric" });
}

function isSameMessageDay(first: string, second: string) {
  const firstDate = new Date(first);
  const secondDate = new Date(second);
  return firstDate.getFullYear() === secondDate.getFullYear()
    && firstDate.getMonth() === secondDate.getMonth()
    && firstDate.getDate() === secondDate.getDate();
}

function formatFileSize(size: number) {
  if (size < 1024 * 1024) {
    return `${Math.max(1, Math.round(size / 1024))} КБ`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} МБ`;
}

function isUserOnline(user: { last_seen_at?: string | null } | null | undefined) {
  if (!user?.last_seen_at) return false;
  return Date.now() - new Date(user.last_seen_at).getTime() < 2 * 60 * 1000;
}

function formatLastSeen(user: { last_seen_at?: string | null } | null | undefined) {
  if (!user?.last_seen_at) return "Был(а) давно";
  const date = new Date(user.last_seen_at);
  return `Был(а) ${date.toLocaleDateString("ru-RU", { day: "numeric", month: "short" })} в ${date.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" })}`;
}

function formatConversationPreview(conversation: ConversationView, viewerId: string | number | null) {
  const message = conversation.last_message;
  if (!message) return conversation.archived ? "Архивный разговор" : "Открыть разговор";
  const attachmentLabel = message.content_type?.startsWith("image/")
    ? "Фото"
    : message.content_type?.startsWith("video/")
      ? "Видео"
      : message.content_type
        ? "Файл"
        : null;
  const preview = attachmentLabel ?? (message.text.trim() || "Сообщение");
  const shortened = preview.length > 56 ? `${preview.slice(0, 56).trimEnd()}…` : preview;
  return { prefix: message.sender_id === viewerId ? "Вы: " : "", text: shortened, attachment: Boolean(attachmentLabel) };
}

export function MessagesPage() {
  const path = usePath();
  const cleanPath = path.split("?")[0];
  const conversationIdFromPath = cleanPath.match(/^\/messages\/([^/]+)$/)?.[1] ?? null;
  const isConversationPage = conversationIdFromPath !== null;
  const requestedUserId = new URLSearchParams(path.split("?")[1] ?? "").get("user");
  const [viewerId, setViewerId] = useState<string | number | null>(null);
  const [conversations, setConversations] = useState<ConversationView[]>([]);
  const [showArchived, setShowArchived] = useState(false);
  const [conversationSearch, setConversationSearch] = useState("");
  const [conversationMenuId, setConversationMenuId] = useState<string | number | null>(null);
  const [selectedId, setSelectedId] = useState<string | number | null>(null);
  const [typingUserId, setTypingUserId] = useState<string | number | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [newMessageStartId, setNewMessageStartId] = useState<string | number | null>(null);
  const [messageText, setMessageText] = useState("");
  const [messageFile, setMessageFile] = useState<File | null>(null);
  const [attachmentOpen, setAttachmentOpen] = useState(false);
  const [editingMessageId, setEditingMessageId] = useState<string | number | null>(null);
  const [editingText, setEditingText] = useState("");
  const [editingSending, setEditingSending] = useState(false);
  const [mediaRemovingId, setMediaRemovingId] = useState<string | number | null>(null);
  const [menuMessageId, setMenuMessageId] = useState<string | number | null>(null);
  const [menuPosition, setMenuPosition] = useState<{ x: number; y: number } | null>(null);
  const [deletingMessageId, setDeletingMessageId] = useState<string | number | null>(null);
  const [loading, setLoading] = useState(true);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const [call, setCall] = useState<CallEvent | null>(null);
  const [callError, setCallError] = useState<string | null>(null);
  const [callSetupOpen, setCallSetupOpen] = useState(false);
  const [pendingCallType, setPendingCallType] = useState<"audio" | "video" | null>(null);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);
  const [microphoneEnabled, setMicrophoneEnabled] = useState(true);
  const [cameraEnabled, setCameraEnabled] = useState(true);
  const [mediaPreparing, setMediaPreparing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const callSocketRef = useRef<WebSocket | null>(null);
  const composerTextareaRef = useRef<HTMLTextAreaElement | null>(null);
  const typingTimeoutRef = useRef<number | null>(null);
  // This object performs ICE/DTLS-SRTP negotiation. It is not a server socket:
  // the actual audio/video path is owned by the two browsers after signaling.
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);
  const pendingIceCandidatesRef = useRef<RTCIceCandidateInit[]>([]);
  const remoteVideoRef = useRef<HTMLVideoElement | null>(null);
  const localVideoRef = useRef<HTMLVideoElement | null>(null);

  function sendSocketEvent(payload: Record<string, unknown>) {
    const socket = socketRef.current;
    if (socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify(payload));
  }

  function sendCallSocketEvent(payload: Record<string, unknown>) {
    const socket = callSocketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) {
      setCallError("Канал звонка ещё не подключён. Обновите страницу и попробуйте снова.");
      return false;
    }
    socket.send(JSON.stringify(payload));
    return true;
  }

  /*
   * getUserMedia is deliberately called before `call.start`/`call.accept`.
   * This gives the user a real pre-call screen: the browser asks for access,
   * shows the local preview, and only then does the signaling state machine
   * contact the other participant. It also avoids the confusing situation in
   * which a call appears to be active while the local browser has no tracks.
   */
  async function ensureLocalStream(callType: "audio" | "video") {
    if (localStreamRef.current) return localStreamRef.current;
    if (!navigator.mediaDevices?.getUserMedia) {
      throw new Error("В этом браузере недоступен API камеры и микрофона. Проверьте разрешения сайта.");
    }

    const stream = await navigator.mediaDevices.getUserMedia({
      audio: true,
      video: callType === "video",
    });
    localStreamRef.current = stream;
    setLocalStream(stream);
    setMicrophoneEnabled(stream.getAudioTracks().some((track) => track.enabled));
    setCameraEnabled(callType === "video" && stream.getVideoTracks().some((track) => track.enabled));
    return stream;
  }

  async function openCallSetup(callType: "audio" | "video") {
    if (!selectedId || !selectedConversation?.other_user_id) return;
    setCallError(null);
    setPendingCallType(callType);
    setCallSetupOpen(true);
    setMediaPreparing(true);
    try {
      await ensureLocalStream(callType);
    } catch (err) {
      setCallError(err instanceof Error ? err.message : "Не удалось получить доступ к камере и микрофону.");
    } finally {
      setMediaPreparing(false);
    }
  }

  function startCallFromSetup() {
    if (!selectedId || !selectedConversation?.other_user_id || !pendingCallType) return;
    const sent = sendCallSocketEvent({
      type: "call.start",
      conversation_id: selectedId,
      target_user_id: selectedConversation.other_user_id,
      call_type: pendingCallType,
    });
    if (sent) {
      setCallSetupOpen(false);
      setCallError(null);
    }
  }

  async function acceptIncomingCall() {
    if (!call || !selectedId) return;
    setCallError(null);
    setMediaPreparing(true);
    try {
      await ensureLocalStream(call.call_type);
      const sent = sendCallSocketEvent({ type: "call.accept", conversation_id: selectedId, call_id: call.call_id });
      if (sent) setCallSetupOpen(false);
    } catch (err) {
      setCallError(err instanceof Error ? err.message : "Не удалось получить доступ к камере и микрофону.");
    } finally {
      setMediaPreparing(false);
    }
  }

  function toggleMicrophone() {
    const nextEnabled = !microphoneEnabled;
    localStreamRef.current?.getAudioTracks().forEach((track) => { track.enabled = nextEnabled; });
    setMicrophoneEnabled(nextEnabled);
  }

  function toggleCamera() {
    const nextEnabled = !cameraEnabled;
    localStreamRef.current?.getVideoTracks().forEach((track) => { track.enabled = nextEnabled; });
    setCameraEnabled(nextEnabled);
  }

  function cancelCallSetup() {
    clearCall();
  }

  /*
   * Closing a call has two independent parts. First close the peer connection
   * and stop local tracks so the camera/microphone indicator is released.
   * Second clear React state so a delayed call.end or a later conversation does
   * not reuse the previous MediaStream. The Go service receives call.end from
   * the button handler; this function only cleans up browser resources.
   */
  function clearCall() {
    peerConnectionRef.current?.close();
    peerConnectionRef.current = null;
    localStreamRef.current?.getTracks().forEach((track) => track.stop());
    localStreamRef.current = null;
    setLocalStream(null);
    setRemoteStream(null);
    pendingIceCandidatesRef.current = [];
    if (remoteVideoRef.current) remoteVideoRef.current.srcObject = null;
    if (localVideoRef.current) localVideoRef.current.srcObject = null;
    setCall(null);
    setCallSetupOpen(false);
    setPendingCallType(null);
    setMediaPreparing(false);
  }

  /*
   * Prepare one browser endpoint for a call. The caller creates the offer only
   * after the callee accepts; the callee creates an endpoint without an offer
   * and waits for the caller's offer. Both sides add local tracks before SDP is
   * created, so the negotiated description contains the requested audio/video
   * capabilities.
   *
   * onicecandidate is intentionally incremental: trickle ICE sends candidates
   * as soon as they are discovered instead of waiting for gathering to finish.
   * This reduces call setup latency and is why signaling remains needed after
   * the initial offer/answer exchange.
   */
  async function preparePeerConnection(callEvent: CallEvent, createOffer: boolean) {
    if (peerConnectionRef.current) return peerConnectionRef.current;
    const stream = await ensureLocalStream(callEvent.call_type);
    const peer = new RTCPeerConnection({
      iceServers: getIceServers(),
    });
    stream.getTracks().forEach((track) => peer.addTrack(track, stream));
    peer.ontrack = (event) => {
      const incomingStream = event.streams[0];
      if (!incomingStream) return;
      setRemoteStream(incomingStream);
      if (remoteVideoRef.current) remoteVideoRef.current.srcObject = incomingStream;
    };
    peer.onconnectionstatechange = () => {
      if (peer.connectionState === "failed") {
        setCallError("WebRTC не смогло соединить устройства. Проверьте Wi-Fi и TURN-сервер.");
      }
    };
    peer.onicecandidate = (event) => {
      if (event.candidate) {
        sendCallSocketEvent({
          type: "call.signal",
          conversation_id: selectedId,
          call_id: callEvent.call_id,
          signal: { kind: "ice", candidate: event.candidate.toJSON() },
        });
      }
    };
    peerConnectionRef.current = peer;
    if (createOffer) {
      const offer = await peer.createOffer();
      await peer.setLocalDescription(offer);
      sendCallSocketEvent({
        type: "call.signal",
        conversation_id: selectedId,
        call_id: callEvent.call_id,
        signal: { kind: "offer", sdp: offer },
      });
    }
    return peer;
  }

  /*
   * This is the browser half of the signaling state machine. `call.invite`,
   * `call.accept`, `call.reject` and `call.end` are application events; offer,
   * answer and ICE are WebRTC negotiation payloads. The Go service validates
   * identity and call state, then echoes a payload to the authorized peer.
   * The sender ignores its own signaling message because it already applied the
   * local description/candidate to its RTCPeerConnection.
   */
  async function handleCallEvent(callEvent: CallEvent) {
    if (callEvent.type === "call.signal" && callEvent.sender_id === viewerId) return;
    if (callEvent.type === "call.reject" || callEvent.type === "call.end") {
      clearCall();
      return;
    }
    setCall(callEvent);
    if (callEvent.type === "call.invite") {
      setCallError(null);
      if (callEvent.callee_id === viewerId) {
        setPendingCallType(callEvent.call_type);
        setCallSetupOpen(true);
        // Do not request media from a background WebSocket callback. The
        // callee explicitly presses "Принять", which is the most reliable
        // browser gesture for opening the camera/microphone permission prompt.
      }
      return;
    }
    if (callEvent.type === "call.accept") {
      setCallSetupOpen(false);
      setPendingCallType(null);
      await preparePeerConnection(callEvent, callEvent.caller_id === viewerId);
      return;
    }
    if (callEvent.type !== "call.signal" || !callEvent.signal) return;
    const peer = await preparePeerConnection(callEvent, false);
    if (callEvent.signal.kind === "ice" && callEvent.signal.candidate) {
      // ICE can arrive a few milliseconds before the offer/answer. Browsers
      // reject addIceCandidate until a remote description exists, so retain
      // the candidate and apply it immediately after setRemoteDescription.
      if (!peer.remoteDescription) {
        pendingIceCandidatesRef.current.push(callEvent.signal.candidate);
      } else {
        await peer.addIceCandidate(callEvent.signal.candidate);
      }
      return;
    }
    if (callEvent.signal.kind === "offer" && callEvent.signal.sdp) {
      await peer.setRemoteDescription(callEvent.signal.sdp);
      for (const candidate of pendingIceCandidatesRef.current.splice(0)) {
        await peer.addIceCandidate(candidate);
      }
      const answer = await peer.createAnswer();
      await peer.setLocalDescription(answer);
      sendCallSocketEvent({
        type: "call.signal",
        conversation_id: selectedId,
        call_id: callEvent.call_id,
        signal: { kind: "answer", sdp: answer },
      });
    } else if (callEvent.signal.kind === "answer" && callEvent.signal.sdp) {
      await peer.setRemoteDescription(callEvent.signal.sdp);
      for (const candidate of pendingIceCandidatesRef.current.splice(0)) {
        await peer.addIceCandidate(candidate);
      }
    }
  }

  useEffect(() => {
    if (localVideoRef.current) localVideoRef.current.srcObject = localStream;
  }, [localStream, callSetupOpen, call]);

  useEffect(() => {
    if (remoteVideoRef.current) remoteVideoRef.current.srcObject = remoteStream;
  }, [remoteStream, call]);

  const selectedConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === selectedId) ?? null,
    [conversations, selectedId],
  );

  async function loadConversations(archived = showArchived) {
    setLoading(true);
    setError(null);

    try {
      const [profile, result] = await Promise.all([
        apiRequest<{ user_id: string }>("/v1/profiles/me"),
        apiRequest<Conversation[]>(`/messages/conversations?archived=${archived}`),
      ]);
      const currentUserID = profile.user_id;
      setViewerId(currentUserID);
      const views = await Promise.all(
        result.map(async (conversation) => {
          const otherUserID = conversation.participant_ids.find((id) => String(id) !== currentUserID) ?? null;
          return { ...conversation, other_user_id: otherUserID, otherUser: otherUserID ? {
            id: String(otherUserID), username: String(otherUserID).slice(0, 8), profile: null,
          } : null };
        }),
      );
      setConversations(views);

      if (requestedUserId) {
        const targetId = requestedUserId;
        if (targetId !== currentUserID) {
          const direct = await apiRequest<Conversation>("/messages/conversations/direct", {
            method: "POST",
            body: JSON.stringify({ other_user_id: targetId }),
          });
          const existing = views.find((conversation) => conversation.id === direct.id);
          if (existing) {
            navigate(`/messages/${existing.id}`);
          } else {
            const directView = { ...direct, other_user_id: targetId, otherUser: {
              id: targetId, username: targetId.slice(0, 8), profile: null,
            } };
            setConversations((current) => [directView, ...current]);
            navigate(`/messages/${direct.id}`);
          }
        }
      }
      if (conversationIdFromPath) {
        setSelectedId(conversationIdFromPath);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить сообщения");
    } finally {
      setLoading(false);
    }
  }

  async function changeConversationState(
    conversationId: string | number,
    action: "archive" | "unarchive" | "delete" | "unread" | "pin" | "unpin" | "mute" | "unmute" | "clear",
  ) {
    if (action === "delete" && !window.confirm("Удалить диалог только у себя? История собеседника не изменится.")) return;
    if (action === "clear" && !window.confirm("Очистить историю только у себя? У собеседника сообщения останутся.")) return;
    const method = action === "delete" || action === "clear" ? "DELETE" : action === "unread" ? "POST" : "PATCH";
    const suffix = action === "archive" || action === "unarchive" || action === "pin" || action === "unpin" || action === "mute" || action === "unmute"
      ? action
      : action === "unread"
        ? "unread"
        : action === "clear"
          ? "history"
          : "";
    await apiRequest(`/messages/conversations/${conversationId}${suffix ? `/${suffix}` : ""}`, { method });
    setConversationMenuId(null);
    if (selectedId === conversationId) setSelectedId(null);
    await loadConversations(showArchived);
  }

  async function loadMessages(conversationId: string | number) {
    setMessagesLoading(true);
    setNewMessageStartId(null);
    setError(null);
    try {
      const result = await apiRequest<MessagePage>(`/messages/conversations/${conversationId}/messages?limit=100`);
      setMessages([...result.items].reverse());
      const lastMessage = result.items[0];
      if (lastMessage) {
        await apiRequest(`/v1/messaging/conversations/${conversationId}/messages/${lastMessage.id}/read`, {
          method: "POST",
          body: JSON.stringify({ message_id: lastMessage.id }),
        });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить переписку");
    } finally {
      setMessagesLoading(false);
    }
  }

  useEffect(() => {
    void loadConversations(showArchived);
    return () => socketRef.current?.close();
  }, [path, showArchived]);

  useEffect(() => {
    if (selectedId === null) {
      setMessages([]);
      setNewMessageStartId(null);
      return;
    }

    void loadMessages(selectedId);
    const socket = new WebSocket(getWebSocketUrl());
    socketRef.current = socket;
    socket.addEventListener("message", (event) => {
      const data = JSON.parse(event.data) as {
        type?: string;
        message?: Message & { body?: string; deleted?: boolean };
        user_id?: string;
        is_typing?: boolean;
        conversation_id?: string;
        call_id?: string;
        sender_id?: string;
        caller_id?: string;
        callee_id?: string;
        call_type?: "audio" | "video";
        status?: "ringing" | "active" | "rejected" | "ended";
        signal?: CallEvent["signal"];
      };
      const message = data.message ? {
        ...data.message,
        text: data.message.text || data.message.body || "",
        deleted_at: data.message.deleted ? data.message.created_at : data.message.deleted_at,
      } : null;
      if (["message.new", "message.updated", "message.deleted"].includes(data.type ?? "") && message?.conversation_id === selectedId) {
        if (message.sender_id !== viewerId) {
          setNewMessageStartId((current) => current ?? message.id);
        }
        setConversations((current) => current.map((conversation) => conversation.id === selectedId
          ? {
            ...conversation,
            last_message: {
              text: message.text,
              sender_id: message.sender_id,
              content_type: message.media_content_type ?? null,
              created_at: message.created_at,
            },
          }
          : conversation));
        setMessages((current) => {
          if (data.type === "message.new") {
            return current.some((item) => item.id === message.id) ? current : [...current, message];
          }
          return current.map((item) => (item.id === message.id ? message : item));
        });
      }
    });
    // Subscribe before sending call.start. The Go service rejects commands for
    // rooms that were not authorized during the WebSocket subscription.
    const callSocket = new WebSocket(getCallWebSocketUrl());
    callSocketRef.current = callSocket;
    callSocket.addEventListener("open", () => {
      callSocket.send(JSON.stringify({ type: "conversation.subscribe", conversation_id: selectedId }));
    });
    callSocket.addEventListener("message", (event) => {
      const data = JSON.parse(event.data) as {
        type?: string;
        call_id?: string;
        sender_id?: string;
        caller_id?: string;
        callee_id?: string;
        call_type?: "audio" | "video";
        status?: "ringing" | "active" | "rejected" | "ended";
        signal?: CallEvent["signal"];
        message?: string;
      };
      if (data.type === "error") {
        setCallError(data.message ?? "Ошибка signaling-сервиса");
        return;
      }
      if (data.type?.startsWith("call.") && data.call_id && data.sender_id && data.caller_id && data.callee_id && data.call_type && data.status) {
        void handleCallEvent({
          type: data.type as CallEvent["type"],
          sender_id: data.sender_id,
          call_id: data.call_id,
          caller_id: data.caller_id,
          callee_id: data.callee_id,
          call_type: data.call_type,
          status: data.status,
          signal: data.signal ?? null,
        }).catch(() => setCallError("Не удалось установить соединение"));
      }
    });
    callSocket.addEventListener("error", () => {
      setCallError("Не удалось подключиться к signaling-сервису звонков через gateway.");
    });
    callSocket.addEventListener("close", () => {
      if (callSocketRef.current === callSocket) {
        setCallError("Соединение с signaling-сервисом звонков прервано.");
      }
    });
    return () => {
      setTypingUserId(null);
      if (typingTimeoutRef.current !== null) window.clearTimeout(typingTimeoutRef.current);
      socket.close();
      callSocket.close();
      clearCall();
      if (socketRef.current === socket) {
        socketRef.current = null;
      }
      if (callSocketRef.current === callSocket) {
        callSocketRef.current = null;
      }
    };
  }, [selectedId]);

  useEffect(() => {
    // В Go messaging websocket нет отдельного клиентского heartbeat-события.
    return undefined;
  }, [selectedId]);

  useEffect(() => {
    if (!call || call.status !== "active" || selectedId === null) return;
    const keepalive = window.setInterval(() => {
      sendCallSocketEvent({
        type: "call.keepalive",
        conversation_id: selectedId,
        call_id: call.call_id,
      });
    }, 30_000);
    return () => window.clearInterval(keepalive);
  }, [call?.call_id, call?.status, selectedId]);

  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    });
    return () => cancelAnimationFrame(frame);
  }, [messages, editingMessageId]);

  useEffect(() => {
    const textarea = composerTextareaRef.current;
    if (!textarea) return;
    textarea.style.height = "auto";
    textarea.style.height = `${Math.min(textarea.scrollHeight, 150)}px`;
  }, [messageText, editingText]);

  useEffect(() => {
    if (conversationMenuId === null) return;
    const closeMenu = (event: MouseEvent) => {
      if (!(event.target as Element).closest(".conversation-actions")) {
        setConversationMenuId(null);
      }
    };
    document.addEventListener("click", closeMenu);
    return () => document.removeEventListener("click", closeMenu);
  }, [conversationMenuId]);

  const otherUserName = getUserName(selectedConversation?.otherUser) || "Собеседник";
  const filteredConversations = conversations.filter((conversation) => {
    const query = conversationSearch.trim().toLowerCase();
    if (!query) return true;
    const name = getUserName(conversation.otherUser).toLowerCase();
    const username = conversation.otherUser?.username.toLowerCase() ?? "";
    return name.includes(query) || username.includes(query);
  });

  async function submitMessage(event: Pick<FormEvent, "preventDefault">) {
    event.preventDefault();
    const text = messageText.trim();
    if ((!text && !messageFile) || selectedId === null) {
      return;
    }

    setSending(true);
    setError(null);
    try {
      // REST returns the persisted message immediately. The backend also
      // broadcasts the same payload through WebSocket to the other participant.
      let mediaId: string | undefined;
      if (messageFile) {
        const mediaBody = new FormData();
        mediaBody.append("file", messageFile);
        const media = await apiRequest<{ id: string }>("/v1/media", { method: "POST", body: mediaBody });
        mediaId = media.id;
      }
      const message = await apiRequest<Message>(`/messages/conversations/${selectedId}/messages`, {
        method: "POST",
        body: JSON.stringify({ body: text, ...(mediaId ? { media_id: mediaId } : {}) }),
      });
      setMessages((current) => (current.some((item) => item.id === message.id) ? current : [...current, message]));
      setMessageText("");
      setMessageFile(null);
      setAttachmentOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отправить сообщение");
    } finally {
      setSending(false);
    }
  }

  function handleMessageTextChange(value: string) {
    setMessageText(value);
  }

  async function removeMessageMedia(message: Message) {
    if (selectedId === null || message.media_id === null) return;
    setMediaRemovingId(message.id);
    setError(null);
    try {
      const updated = await apiRequest<Message>(`/v1/messaging/conversations/${selectedId}/messages/${message.id}/media`, { method: "DELETE" });
      setMessages((current) => current.map((item) => item.id === updated.id ? updated : item));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить вложение");
    } finally {
      setMediaRemovingId(null);
    }
  }

  async function deleteMessage(message: Message) {
    if (selectedId === null) return;
    setDeletingMessageId(message.id);
    setError(null);
    try {
      await apiRequest<void>(`/v1/messaging/messages/${message.id}`, { method: "DELETE" });
      setMessages((current) => current.filter((item) => item.id !== message.id));
      setMenuMessageId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить сообщение");
    } finally {
      setDeletingMessageId(null);
    }
  }

  async function copyMessage(message: Message) {
    try {
      await navigator.clipboard.writeText(message.text);
      setMenuMessageId(null);
    } catch {
      setError("Не удалось скопировать сообщение");
    }
  }

  async function saveEditedMessage(event: Pick<FormEvent, "preventDefault">) {
    event.preventDefault();
    const text = editingText.trim();
    if (!text || editingMessageId === null || selectedId === null) return;
    setEditingSending(true);
    setError(null);
    try {
      const updated = await apiRequest<Message>(`/v1/messaging/messages/${editingMessageId}`, {
        method: "PATCH",
        body: JSON.stringify({ body: text }),
      });
      setMessages((current) => current.map((item) => item.id === updated.id ? updated : item));
      setEditingMessageId(null);
      setEditingText("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось изменить сообщение");
    } finally {
      setEditingSending(false);
    }
  }

  function handleMessageKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      if (editingMessageId !== null) {
        if (!editingSending && editingText.trim()) void saveEditedMessage(event);
        return;
      }
      if (!sending && (messageText.trim() || messageFile)) {
        void submitMessage(event);
      }
    }
  }

  if (loading) {
    return <p className="muted">Загружаем сообщения...</p>;
  }

  return (
    <section className="messages-page">
      <header className="page-header messages-title">
        <div>
          <span className="eyebrow">Личные разговоры</span>
          <h1>Сообщения</h1>
        </div>
        <span className="messages-count">{conversations.length} диалогов</span>
      </header>
      {error && <p className="error">{error}</p>}
      <div className={`messages-layout panel ${isConversationPage ? "conversation-detail-layout" : "conversation-list-only"}`}>
        {!isConversationPage && <aside className="conversation-list">
          <div className="conversation-list-heading">
            <div className="conversation-tabs">
              <button type="button" className={!showArchived ? "active" : ""} onClick={() => setShowArchived(false)}>Диалоги</button>
              <button type="button" className={showArchived ? "active" : ""} onClick={() => setShowArchived(true)}>Архив</button>
            </div>
            <span>{filteredConversations.length}</span>
          </div>
          <label className="conversation-search">
            <span className="sr-only">Поиск по диалогам</span>
            <input value={conversationSearch} onChange={(event) => setConversationSearch(event.target.value)} placeholder="Поиск по диалогам" />
          </label>
          {conversations.length === 0 && <p className="muted conversation-empty">Здесь появятся ваши разговоры.</p>}
          {conversations.length > 0 && filteredConversations.length === 0 && <p className="muted conversation-empty">Ничего не найдено.</p>}
          {filteredConversations.map((conversation) => (
            <div
              key={conversation.id}
              className={conversation.id === selectedId ? "conversation-item-row active" : "conversation-item-row"}
            >
              <button type="button" className="conversation-item" onClick={() => navigate(`/messages/${conversation.id}`)}>
                <UserAvatar user={conversation.otherUser} size="sm" />
                <span>
                  <strong>{getUserName(conversation.otherUser) || `Пользователь #${conversation.other_user_id}`}</strong>
                  {(() => {
                    const preview = formatConversationPreview(conversation, viewerId);
                    return typeof preview === "string" ? <small>{preview}</small> : (
                      <small className={preview.attachment ? "conversation-preview attachment" : "conversation-preview"}>
                        {preview.prefix}<span>{preview.text}</span>
                      </small>
                    );
                  })()}
                </span>
              </button>
              <div className="conversation-actions" onMouseLeave={() => setConversationMenuId(null)}>
                <button
                  type="button"
                  className="conversation-menu-trigger"
                  aria-label="Действия с диалогом"
                  aria-expanded={conversationMenuId === conversation.id}
                  onClick={(event) => { event.stopPropagation(); setConversationMenuId((current) => current === conversation.id ? null : conversation.id); }}
                >•••</button>
                {conversationMenuId === conversation.id && (
                  <div className="conversation-actions-menu" role="menu" onClick={(event) => event.stopPropagation()}>
                    <button type="button" role="menuitem" onClick={() => void changeConversationState(conversation.id, "unread")}>◌ Отметить непрочитанным</button>
                    <button type="button" role="menuitem" onClick={() => void changeConversationState(conversation.id, conversation.pinned ? "unpin" : "pin")}>{conversation.pinned ? "⌁ Открепить чат" : "⌁ Закрепить чат"}</button>
                    <button type="button" role="menuitem" onClick={() => void changeConversationState(conversation.id, showArchived ? "unarchive" : "archive")}>{showArchived ? "▣ Вернуть из архива" : "▣ Архивировать"}</button>
                    <button type="button" role="menuitem" onClick={() => void changeConversationState(conversation.id, conversation.muted ? "unmute" : "mute")}>{conversation.muted ? "♧ Включить уведомления" : "♧ Отключить уведомления"}</button>
                    <button type="button" role="menuitem" className="danger" onClick={() => void changeConversationState(conversation.id, "clear")}>⌫ Очистить историю</button>
                    <button type="button" role="menuitem" className="danger" onClick={() => void changeConversationState(conversation.id, "delete")}>Удалить диалог</button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </aside>}
        {isConversationPage && <section className="conversation-view">
          {selectedConversation ? (
            <>
              <header className="conversation-header">
                <button type="button" className="conversation-back" onClick={() => navigate("/messages")}>← Все диалоги</button>
                <button type="button" className="conversation-person" onClick={() => selectedConversation.other_user_id && navigate(`/profile/${selectedConversation.other_user_id}`)}>
                  <UserAvatar user={selectedConversation.otherUser} size="sm" />
                  <span>
                    <strong>{getUserName(selectedConversation.otherUser)}</strong>
                    <small className={isUserOnline(selectedConversation.otherUser) ? "conversation-status online" : "conversation-status"}>
                      {isUserOnline(selectedConversation.otherUser) ? "В сети" : formatLastSeen(selectedConversation.otherUser)}
                    </small>
                  </span>
                </button>
                <div className="conversation-call-actions">
                  <button
                    type="button"
                    className="secondary"
                    disabled={!selectedConversation.can_send_message || call !== null || callSetupOpen}
                    onClick={() => void openCallSetup("audio")}
                  >
                    📞 Позвонить
                  </button>
                  <button
                    type="button"
                    className="secondary"
                    disabled={!selectedConversation.can_send_message || call !== null || callSetupOpen}
                    onClick={() => void openCallSetup("video")}
                  >
                    🎥 Видео
                  </button>
                </div>
              </header>
              <div className="message-list" aria-live="polite">
                {messagesLoading && <p className="muted">Загружаем переписку...</p>}
                {!messagesLoading && messages.length === 0 && (
                  <div className="empty-state message-empty">
                    <h2>Начните разговор</h2>
                    <p>Напишите первое сообщение — оно появится здесь.</p>
                  </div>
                )}
                {messages.map((message, index) => {
                  const previousMessage = messages[index - 1];
                  const showDateDivider = !previousMessage || !isSameMessageDay(previousMessage.created_at, message.created_at);
                  return (
                  <div key={message.id} className="message-group">
                    {showDateDivider && <div className="message-date-divider">{formatMessageDay(message.created_at)}</div>}
                    {message.id === newMessageStartId && <div className="new-messages-divider">Новые сообщения</div>}
                    <article
                      className={message.sender_id === viewerId ? "message-bubble own" : "message-bubble"}
                      aria-label={message.sender_id === viewerId ? "Ваше сообщение" : `Сообщение от ${otherUserName}`}
                      onContextMenu={(event) => {
                        event.preventDefault();
                        if (!message.deleted_at) {
                          setMenuMessageId(message.id);
                          setMenuPosition({ x: event.clientX, y: event.clientY });
                        }
                      }}
                    >
                      <span className="message-author">{message.sender_id === viewerId ? "Вы" : otherUserName}</span>
                      <p>{message.deleted_at ? "Сообщение удалено" : message.text}</p>
                      {message.media_id && (
                        message.media_content_type?.startsWith("image/") ? (
                          <a className="message-image-preview" href={`${API_BASE_URL}/v1/media/${message.media_id}/content`} target="_blank" rel="noreferrer">
                            <img src={`${API_BASE_URL}/v1/media/${message.media_id}/content`} alt="Вложение к сообщению" />
                            <span>Открыть оригинал</span>
                          </a>
                        ) : (
                          <a href={`${API_BASE_URL}/v1/media/${message.media_id}/content`} target="_blank" rel="noreferrer">
                            Открыть вложение
                          </a>
                        )
                      )}
                      <div className="message-meta">
                        <time>{formatMessageTime(message.created_at)}</time>
                        {message.edited_at && !message.deleted_at && <small className="message-edited-label">изменено</small>}
                      </div>
                      {!message.deleted_at && menuMessageId === message.id && (
                        <div className="message-actions message-context-menu-anchor" onMouseLeave={() => setMenuMessageId(null)} style={menuPosition ? { left: Math.min(menuPosition.x + 8, window.innerWidth - 196), top: Math.min(menuPosition.y + 8, window.innerHeight - 180) } : undefined}>
                          <div className="message-actions-menu" role="menu">
                            <button type="button" role="menuitem" onClick={() => void copyMessage(message)}>Копировать текст</button>
                            {message.sender_id === viewerId && (
                              <>
                                <button type="button" role="menuitem" onClick={() => { setEditingMessageId(message.id); setEditingText(message.text); setMenuMessageId(null); setError(null); }}>Редактировать</button>
                                {message.media_id && <button type="button" role="menuitem" onClick={() => { setMenuMessageId(null); void removeMessageMedia(message); }}>Удалить вложение</button>}
                                <button type="button" role="menuitem" className="danger" onClick={() => void deleteMessage(message)} disabled={deletingMessageId === message.id}>{deletingMessageId === message.id ? "Удаляем..." : "Удалить сообщение"}</button>
                              </>
                            )}
                          </div>
                        </div>
                      )}
                    </article>
                  </div>
                );
                })}
                <div ref={bottomRef} />
              </div>
              {callSetupOpen && (!call || (call.status === "ringing" && call.callee_id === viewerId)) && (
                <div className="call-modal-backdrop">
                  <section className="call-dialog call-setup-dialog" role="dialog" aria-modal="true" aria-labelledby="call-setup-title">
                    <div className="call-dialog-heading">
                      <div>
                        <span className="call-kicker">{call ? "Входящий звонок" : "Подготовка звонка"}</span>
                        <h2 id="call-setup-title">{call ? getUserName(selectedConversation.otherUser) : otherUserName}</h2>
                        <p>{(call?.call_type ?? pendingCallType) === "video" ? "Видео-звонок" : "Аудио-звонок"}</p>
                      </div>
                      <span className="call-status-dot" aria-label="Камера и микрофон настраиваются" />
                    </div>

                    <div className="call-preview-shell">
                      {(call?.call_type ?? pendingCallType) === "video" && localStream ? (
                        <>
                          <video ref={localVideoRef} autoPlay muted playsInline className="call-local-preview" />
                          {!cameraEnabled && <span className="call-video-off-label">Камера выключена</span>}
                        </>
                      ) : (call?.call_type ?? pendingCallType) === "video" ? (
                        <div className="call-audio-preview">
                          <span className="call-audio-preview-icon">📹</span>
                          <strong>Камера пока не включена</strong>
                          <span>Нажмите «Принять», чтобы разрешить доступ и увидеть себя</span>
                        </div>
                      ) : (
                        <div className="call-audio-preview">
                          <span className="call-audio-preview-icon">🎙️</span>
                          <strong>{localStream ? "Микрофон готов" : "Микрофон пока не включен"}</strong>
                          <span>{localStream ? "Перед началом вы увидите и настроите себя" : "Нажмите «Принять», чтобы разрешить доступ"}</span>
                        </div>
                      )}
                      {mediaPreparing && <div className="call-preview-loading">Запрашиваем доступ к устройствам…</div>}
                    </div>

                    <div className="call-controls call-setup-controls">
                      <button type="button" className={microphoneEnabled ? "call-control active" : "call-control muted"} onClick={toggleMicrophone} disabled={!localStream}>
                        {microphoneEnabled ? "🎙️ Микрофон" : "🔇 Микрофон выключен"}
                      </button>
                      {(call?.call_type ?? pendingCallType) === "video" && (
                        <button type="button" className={cameraEnabled ? "call-control active" : "call-control muted"} onClick={toggleCamera} disabled={!localStream}>
                          {cameraEnabled ? "📹 Камера" : "🚫 Камера выключена"}
                        </button>
                      )}
                    </div>

                    {call ? (
                      <div className="call-action-row">
                        <button type="button" className="primary" onClick={() => void acceptIncomingCall()} disabled={mediaPreparing}>Принять</button>
                        <button type="button" className="secondary" onClick={() => { sendCallSocketEvent({ type: "call.reject", conversation_id: selectedId, call_id: call.call_id }); clearCall(); }}>Отклонить</button>
                      </div>
                    ) : (
                      <div className="call-action-row">
                        <button type="button" className="primary" onClick={startCallFromSetup} disabled={mediaPreparing || !localStream}>Позвонить</button>
                        <button type="button" className="secondary" onClick={cancelCallSetup}>Отмена</button>
                      </div>
                    )}
                    {callError && <p className="call-error" role="alert">{callError}</p>}
                  </section>
                </div>
              )}
              {call && !(callSetupOpen && call.status === "ringing" && call.callee_id === viewerId) && (
                <div className="call-modal-backdrop">
                  <section className="call-dialog call-active-dialog" role="dialog" aria-modal="true" aria-labelledby="active-call-title">
                    <div className="call-dialog-heading">
                      <div>
                        <span className="call-kicker">{call.caller_id === viewerId ? "Звонок" : "Собеседник"}</span>
                        <h2 id="active-call-title">{otherUserName}</h2>
                        <p>{call.call_type === "video" ? "Видео-звонок" : "Аудио-звонок"} · {call.status === "active" ? "Подключено" : "Ожидаем ответа…"}</p>
                      </div>
                      <span className={call.status === "active" ? "call-status-dot connected" : "call-status-dot"} />
                    </div>

                    <div className="call-video-stage">
                      <div className="call-remote-tile">
                        <video ref={remoteVideoRef} autoPlay playsInline className="call-remote-video" />
                        {!remoteStream && <div className="call-media-placeholder"><span>◉</span><strong>{call.status === "active" ? "Подключаем видео…" : "Ожидаем ответа…"}</strong></div>}
                        {call.call_type === "audio" && <div className="call-media-placeholder"><span>🎧</span><strong>Аудио-звонок</strong></div>}
                      </div>
                      <div className="call-local-tile">
                        {call.call_type === "video" ? (
                          <>
                            <video ref={localVideoRef} autoPlay muted playsInline className="call-local-video" />
                            {!cameraEnabled && <span className="call-video-off-label">Камера выключена</span>}
                          </>
                        ) : <span className="call-local-audio-icon">🎙️</span>}
                        <span className="call-local-name">Вы</span>
                      </div>
                    </div>

                    <div className="call-controls">
                      <button type="button" className={microphoneEnabled ? "call-control active" : "call-control muted"} onClick={toggleMicrophone}>
                        {microphoneEnabled ? "🎙️ Микрофон" : "🔇 Микрофон выключен"}
                      </button>
                      {call.call_type === "video" && (
                        <button type="button" className={cameraEnabled ? "call-control active" : "call-control muted"} onClick={toggleCamera}>
                          {cameraEnabled ? "📹 Камера" : "🚫 Камера выключена"}
                        </button>
                      )}
                      <button type="button" className="call-control end" onClick={() => { sendCallSocketEvent({ type: "call.end", conversation_id: selectedId, call_id: call.call_id }); clearCall(); }}>☎ Завершить</button>
                    </div>
                    {callError && <p className="call-error" role="alert">{callError}</p>}
                  </section>
                </div>
              )}
              {typingUserId !== null && (
                <div className="conversation-presence typing-presence" role="status">печатает…</div>
              )}
              {!selectedConversation.can_send_message && (
                <div className="message-restriction" role="status">
                  <strong>Сообщения ограничены</strong>
                  <span>Пользователь запретил отправлять ему новые сообщения.</span>
                </div>
              )}
              <form className="message-composer" onSubmit={editingMessageId !== null ? saveEditedMessage : submitMessage}>
                <fieldset disabled={!selectedConversation.can_send_message} className="message-composer-fields">
                {editingMessageId !== null && (
                  <div className="message-editing-bar">
                    <div>
                      <strong>Редактирование сообщения</strong>
                      <small>Измените текст и сохраните его</small>
                    </div>
                    <button type="button" onClick={() => { setEditingMessageId(null); setEditingText(""); }} aria-label="Отменить редактирование">×</button>
                  </div>
                )}
                {editingMessageId === null && messageFile && (
                  <div className="message-file-card">
                    <span className="message-file-card-icon">{messageFile.type.startsWith("image/") ? "◈" : "•"}</span>
                    <span className="message-file-card-info">
                      <strong>{messageFile.name}</strong>
                      <small>{formatFileSize(messageFile.size)} · готово к отправке</small>
                    </span>
                    <button type="button" onClick={() => setMessageFile(null)} aria-label="Убрать вложение">×</button>
                  </div>
                )}
                <div className="message-composer-row">
                  <div className="message-attachment-wrap">
                    <button
                      type="button"
                      className={attachmentOpen ? "message-attachment active" : "message-attachment"}
                      aria-label="Прикрепить файл"
                      aria-expanded={attachmentOpen}
                      onClick={() => setAttachmentOpen((current) => !current)}
                    >
                      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                        <path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48" />
                      </svg>
                    </button>
                    {attachmentOpen && (
                      <div className="attachment-popover" role="dialog" aria-label="Добавить вложение">
                        <div className="attachment-popover-heading">
                          <strong>Добавить вложение</strong>
                          <span>Выберите файл для сообщения</span>
                        </div>
                        <label className="attachment-option">
                          <span className="attachment-option-icon photo-icon" aria-hidden="true">
                            <svg viewBox="0 0 24 24" focusable="false">
                              <rect x="3" y="4" width="18" height="16" rx="3" />
                              <circle cx="8.5" cy="9" r="1.5" />
                              <path d="m4.5 17 4.2-4.2a1.8 1.8 0 0 1 2.5 0l1.4 1.4 1.2-1.2a1.8 1.8 0 0 1 2.5 0l3.2 3.2" />
                            </svg>
                          </span>
                          <span><strong>Фото и видео</strong><small>JPG, PNG, GIF, MP4</small></span>
                          <input type="file" accept="image/*,video/*" onChange={(event) => { setMessageFile(event.target.files?.[0] ?? null); setAttachmentOpen(false); }} />
                        </label>
                        <label className="attachment-option">
                          <span className="attachment-option-icon file-icon" aria-hidden="true">
                            <svg viewBox="0 0 24 24" focusable="false">
                              <path d="M7 3.5h7l3 3V20.5H7a2 2 0 0 1-2-2v-13a2 2 0 0 1 2-2Z" />
                              <path d="M14 3.5v4h4M8.5 12h7M8.5 15h7M8.5 18h4" />
                            </svg>
                          </span>
                          <span><strong>Документ</strong><small>PDF, DOC, TXT и другие</small></span>
                          <input type="file" accept=".pdf,.doc,.docx,.txt,.zip" onChange={(event) => { setMessageFile(event.target.files?.[0] ?? null); setAttachmentOpen(false); }} />
                        </label>
                      </div>
                    )}
                  </div>
                  <textarea
                    ref={composerTextareaRef}
                    value={editingMessageId !== null ? editingText : messageText}
                    onChange={(event) => editingMessageId !== null ? setEditingText(event.target.value) : handleMessageTextChange(event.target.value)}
                    onKeyDown={handleMessageKeyDown}
                    placeholder="Напишите сообщение..."
                    rows={1}
                    maxLength={5000}
                  />
                  <button className="message-send" type="submit" disabled={editingSending || sending || (editingMessageId !== null ? !editingText.trim() : (!messageText.trim() && !messageFile))} aria-label={editingMessageId !== null ? "Сохранить изменения" : "Отправить сообщение"}>
                    <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 4 16 8-16 8 3-8-3-8Zm3 8h13" /></svg>
                  </button>
                </div>
                </fieldset>
              </form>
            </>
          ) : (
            <div className="message-welcome">
              <h2>Диалог не найден</h2>
              <p>Вернитесь к списку диалогов и выберите нужный разговор.</p>
              <button type="button" className="secondary" onClick={() => navigate("/messages")}>К списку диалогов</button>
            </div>
          )}
        </section>}
      </div>
    </section>
  );
}
