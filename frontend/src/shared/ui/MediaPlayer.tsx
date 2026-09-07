import { useRef } from "react";
import type { MouseEvent } from "react";


type MediaPlayerProps = {
  src: string;
  title: string;
  kind: "audio" | "video";
  className?: string;
  autoPlay?: boolean;
  preload?: "none" | "metadata" | "auto";
  onProgress?: (currentTime: number, duration: number) => void;
  onPause?: (currentTime: number, duration: number) => void;
  onComplete?: (currentTime: number, duration: number) => void;
};

export function MediaPlayer({ src, title, kind, className, autoPlay = false, preload = "metadata", onProgress, onPause, onComplete }: MediaPlayerProps) {
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
        <video
          ref={videoRef}
          className={className}
          src={src}
          controls
          autoPlay={autoPlay}
          preload={preload}
          onClick={seekFromVideo}
          onTimeUpdate={(event) => onProgress?.(event.currentTarget.currentTime, event.currentTarget.duration)}
          onPause={(event) => onPause?.(event.currentTarget.currentTime, event.currentTarget.duration)}
          onEnded={(event) => onComplete?.(event.currentTarget.currentTime, event.currentTarget.duration)}
          aria-label={title}
        />
      ) : (
        <audio
          ref={audioRef}
          src={src}
          controls
          preload={preload}
          onTimeUpdate={(event) => onProgress?.(event.currentTarget.currentTime, event.currentTarget.duration)}
          onEnded={(event) => onComplete?.(event.currentTarget.currentTime, event.currentTarget.duration)}
          aria-label={title}
        />
      )}
    </div>
  );
}
