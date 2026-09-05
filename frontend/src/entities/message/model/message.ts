import type { User } from "../../user/model/user";

export type Conversation = {
  id: string | number;
  participant_ids: Array<string | number>;
  other_user_id: string | number | null;
  created_at: string;
  archived: boolean;
  pinned: boolean;
  muted: boolean;
  can_send_message: boolean;
  last_message: {
    text: string;
    sender_id: string | number;
    content_type: string | null;
    created_at: string;
  } | null;
};

export type Message = {
  id: string | number;
  conversation_id: string | number;
  sender_id: string | number;
  text: string;
  created_at: string;
  media_id: string | number | null;
  media_content_type?: string | null;
  edited_at: string | null;
  deleted_at: string | null;
};

export type MessagePage = {
  items: Message[];
  next_cursor: string | null;
  has_more: boolean;
};

export type ConversationView = Conversation & {
  otherUser: User | null;
};

/**
 * Application-level call event transported by the Go signaling service.
 *
 * `signal` is deliberately typed as browser WebRTC input (`RTCSessionDescriptionInit`
 * and `RTCIceCandidateInit`) instead of as a server-specific format. The
 * signaling service does not interpret codecs or network candidates; it checks
 * authorization and forwards the browser-generated JSON to the other peer.
 */
export type CallEvent = {
  type: "call.invite" | "call.accept" | "call.reject" | "call.end" | "call.signal";
  sender_id: string | number;
  call_id: string;
  caller_id: string | number;
  callee_id: string | number;
  call_type: "audio" | "video";
  status: "ringing" | "active" | "rejected" | "ended";
  signal: {
    kind?: "offer" | "answer" | "ice";
    sdp?: RTCSessionDescriptionInit;
    candidate?: RTCIceCandidateInit;
  } | null;
};
