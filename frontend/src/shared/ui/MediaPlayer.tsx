import { useEffect, useRef, useState } from "react";
import type { ChangeEvent, MouseEvent, RefObject } from "react";


type MediaPlayerProps = {
  src: string;
  title: string;
  kind: "audio" | "video";
  className?: string;
  autoPlay?: boolean;
  preload?: "none" | "metadata" | "auto";
};

export function MediaPlayer({ src, title, kind, className, autoPlay = false, preload = "metadata" }: MediaPlayerProps) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const mediaRef = kind === "video" ? videoRef : audioRef;

  function seekFromVideo(event: MouseEvent<HTMLVideoElement>) {
    if (event.target !== event.currentTarget) return;
    const media = mediaRef.current;
    if (!media || !Number.isFinite(media.duration) || media.duration <= 0) return;
    const bounds = event.currentTarget.getBoundingClientRect();
    const progress = Math.min(1, Math.max(0, (event.clientX - bounds.left) / bounds.width));
    media.currentTime = progress * media.duration;
  }

  return (
    <div className={`media-player media-player-${kind}`}>
      {kind === "video" ? (
        <video ref={videoRef} className={className} src={src} controls autoPlay={autoPlay} preload={preload} onClick={seekFromVideo} aria-label={title} />
      ) : (
        <audio ref={audioRef} src={src} controls preload={preload} aria-label={title} />
      )}
      <MediaSeekBar mediaRef={mediaRef} mediaKey={src} />
    </div>
  );
}

type MediaSeekBarProps = {
  mediaRef: RefObject<HTMLMediaElement | null>;
  mediaKey: string;
};

function MediaSeekBar({ mediaRef, mediaKey }: MediaSeekBarProps) {
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    const media = mediaRef.current;
    if (!media) return;

    const sync = () => {
      setCurrentTime(Number.isFinite(media.currentTime) ? media.currentTime : 0);
      setDuration(Number.isFinite(media.duration) ? media.duration : 0);
    };
    media.addEventListener("loadedmetadata", sync);
    media.addEventListener("durationchange", sync);
    media.addEventListener("timeupdate", sync);
    media.addEventListener("seeking", sync);
    sync();

    return () => {
      media.removeEventListener("loadedmetadata", sync);
      media.removeEventListener("durationchange", sync);
      media.removeEventListener("timeupdate", sync);
      media.removeEventListener("seeking", sync);
    };
  }, [mediaKey, mediaRef]);

  function seek(event: ChangeEvent<HTMLInputElement>) {
    const value = Number(event.target.value);
    const media = mediaRef.current;
    if (!media || !Number.isFinite(value)) return;
    media.currentTime = value;
    setCurrentTime(value);
  }

  return (
    <div className="media-seekbar">
      <input
        type="range"
        min="0"
        max={duration || 0}
        step="0.1"
        value={Math.min(currentTime, duration || 0)}
        onChange={seek}
        disabled={!duration}
        aria-label={`Позиция в треке ${formatTime(currentTime)} из ${formatTime(duration)}`}
      />
      <div className="media-seekbar-times"><span>{formatTime(currentTime)}</span><span>{formatTime(duration)}</span></div>
    </div>
  );
}

function formatTime(value: number) {
  if (!Number.isFinite(value) || value < 0) return "0:00";
  const totalSeconds = Math.floor(value);
  return `${Math.floor(totalSeconds / 60)}:${String(totalSeconds % 60).padStart(2, "0")}`;
}
