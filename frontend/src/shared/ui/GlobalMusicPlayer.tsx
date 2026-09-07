import { createContext, useContext, useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";

import type { MusicTrack } from "../../entities/music/model/music";
import { API_BASE_URL } from "../config/api";

type MusicPlayerContextValue = {
  currentTrack: MusicTrack | null;
  queue: MusicTrack[];
  isPlaying: boolean;
  currentTime: number;
  duration: number;
  play: (track: MusicTrack, queue?: MusicTrack[]) => void;
  toggle: (track: MusicTrack, queue?: MusicTrack[]) => void;
  pause: () => void;
  seek: (value: number) => void;
  previous: () => void;
  next: () => void;
};

const MusicPlayerContext = createContext<MusicPlayerContextValue | null>(null);
const PLAYER_STORAGE_KEY = "general.music_player";

type PersistedPlayer = {
  currentTrack: MusicTrack;
  queue: MusicTrack[];
};

function readPersistedPlayer(): PersistedPlayer | null {
  try {
    const value = window.localStorage.getItem(PLAYER_STORAGE_KEY);
    if (!value) return null;
    const parsed = JSON.parse(value) as PersistedPlayer;
    if (!parsed.currentTrack?.id || !Array.isArray(parsed.queue)) return null;
    return parsed;
  } catch {
    return null;
  }
}

export function MusicPlayerProvider({ children }: { children: ReactNode }) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const shouldAutoplay = useRef(false);
  const persisted = useRef<PersistedPlayer | null>(readPersistedPlayer());
  const [currentTrack, setCurrentTrack] = useState<MusicTrack | null>(() => persisted.current?.currentTrack ?? null);
  const [queue, setQueue] = useState<MusicTrack[]>(() => persisted.current?.queue ?? []);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio || !currentTrack) return;

    audio.src = trackURL(currentTrack);
    audio.load();
    setCurrentTime(0);
    setDuration(currentTrack.duration ?? 0);

    if (shouldAutoplay.current) {
      shouldAutoplay.current = false;
      void audio.play()
        .then(() => setIsPlaying(true))
        .catch(() => setIsPlaying(false));
    }
  }, [currentTrack]);

  useEffect(() => {
    if (!currentTrack) {
      window.localStorage.removeItem(PLAYER_STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(PLAYER_STORAGE_KEY, JSON.stringify({ currentTrack, queue } satisfies PersistedPlayer));
  }, [currentTrack, queue]);

  function play(track: MusicTrack, nextQueue = [track]) {
    const audio = audioRef.current;
    const normalizedQueue = nextQueue.some((item) => item.id === track.id) ? nextQueue : [track, ...nextQueue];
    setQueue(normalizedQueue);

    if (currentTrack?.id === track.id && audio) {
      void audio.play()
        .then(() => setIsPlaying(true))
        .catch(() => setIsPlaying(false));
      return;
    }

    shouldAutoplay.current = true;
    setCurrentTrack(track);
  }

  function toggle(track: MusicTrack, nextQueue = [track]) {
    if (currentTrack?.id !== track.id) {
      play(track, nextQueue);
      return;
    }

    const audio = audioRef.current;
    if (!audio) return;
    if (audio.paused) {
      void audio.play()
        .then(() => setIsPlaying(true))
        .catch(() => setIsPlaying(false));
    } else {
      audio.pause();
      setIsPlaying(false);
    }
  }

  function pause() {
    audioRef.current?.pause();
    setIsPlaying(false);
  }

  function seek(value: number) {
    const audio = audioRef.current;
    if (!audio || !Number.isFinite(value)) return;
    audio.currentTime = value;
    setCurrentTime(value);
  }

  function next() {
    if (!currentTrack) return;
    const index = queue.findIndex((track) => track.id === currentTrack.id);
    const nextTrack = index >= 0 ? queue[index + 1] : undefined;
    if (nextTrack) {
      shouldAutoplay.current = true;
      setCurrentTrack(nextTrack);
    } else {
      pause();
    }
  }

  function previous() {
    if (!currentTrack) return;
    const index = queue.findIndex((track) => track.id === currentTrack.id);
    const previousTrack = index > 0 ? queue[index - 1] : undefined;
    if (previousTrack) {
      shouldAutoplay.current = true;
      setCurrentTrack(previousTrack);
      return;
    }
    seek(0);
  }

  return (
    <MusicPlayerContext.Provider value={{ currentTrack, queue, isPlaying, currentTime, duration, play, toggle, pause, seek, previous, next }}>
      {children}
      <audio
        ref={audioRef}
        className="global-music-audio"
        preload="metadata"
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onTimeUpdate={(event) => setCurrentTime(event.currentTarget.currentTime)}
        onLoadedMetadata={(event) => setDuration(Number.isFinite(event.currentTarget.duration) ? event.currentTarget.duration : currentTrack?.duration ?? 0)}
        onEnded={next}
        aria-label={currentTrack ? `${currentTrack.title} — ${currentTrack.artist || "трек"}` : "Музыкальный плеер"}
      />
    </MusicPlayerContext.Provider>
  );
}

export function useMusicPlayer() {
  const context = useContext(MusicPlayerContext);
  if (!context) {
    throw new Error("useMusicPlayer должен использоваться внутри MusicPlayerProvider");
  }
  return context;
}

export function MusicTrackControl({ track, queue }: { track: MusicTrack; queue?: MusicTrack[] }) {
  const player = useMusicPlayer();
  const active = player.currentTrack?.id === track.id;

  return (
    <button
      type="button"
      className={active ? "music-track-play active" : "music-track-play"}
      onClick={() => player.toggle(track, queue)}
      aria-label={active && player.isPlaying ? `Поставить на паузу: ${track.title}` : `Воспроизвести: ${track.title}`}
    >
      {active && player.isPlaying ? "Ⅱ" : "▶"}
    </button>
  );
}

export function GlobalMusicPlayer() {
  const player = useMusicPlayer();
  const [expanded, setExpanded] = useState(false);
  const track = player.currentTrack;

  if (!track) return null;

  const playerDuration = player.duration || track.duration || 0;

  return (
    <div className="global-music-player">
      <div className="global-music-bar">
        <button type="button" className="global-music-skip" onClick={player.previous} aria-label="Предыдущий трек">‹</button>
        <button type="button" className="global-music-play" onClick={() => player.toggle(track, player.queue)} aria-label={player.isPlaying ? "Пауза" : "Воспроизвести"}>
          {player.isPlaying ? "Ⅱ" : "▶"}
        </button>
        <button type="button" className="global-music-skip" onClick={player.next} aria-label="Следующий трек">›</button>
        <button type="button" className="global-music-track" onClick={() => setExpanded((value) => !value)} aria-expanded={expanded}>
          <span className="global-music-art" aria-hidden="true">♫</span>
          <span className="global-music-copy"><strong>{track.title}</strong><small>{track.artist || "Исполнитель не указан"}</small></span>
        </button>
        <span className="global-music-time">{formatTime(player.currentTime)}</span>
        <input
          className="global-music-range"
          type="range"
          min="0"
          max={playerDuration || 0}
          step="0.1"
          value={Math.min(player.currentTime, playerDuration || 0)}
          onChange={(event) => player.seek(Number(event.target.value))}
          disabled={!playerDuration}
          aria-label="Позиция в треке"
        />
        <button type="button" className="global-music-expand" onClick={() => setExpanded((value) => !value)} aria-label={expanded ? "Свернуть плеер" : "Открыть очередь"}>
          {expanded ? "⌃" : "⌄"}
        </button>
      </div>

      {expanded && (
        <section className="global-music-dropdown" aria-label="Очередь воспроизведения">
          <header className="global-music-dropdown-header">
            <div><span className="eyebrow">Сейчас играет</span><h2>Очередь</h2></div>
            <button type="button" className="global-music-close" onClick={() => setExpanded(false)} aria-label="Закрыть очередь">×</button>
          </header>
          <div className="global-music-queue">
            {player.queue.map((item) => {
              const active = item.id === track.id;
              return (
                <button type="button" className={active ? "global-music-queue-item active" : "global-music-queue-item"} key={item.id} onClick={() => player.play(item, player.queue)}>
                  <span className="global-music-queue-art" aria-hidden="true">♫</span>
                  <span><strong>{item.title}</strong><small>{item.artist || "Исполнитель не указан"}</small></span>
                  <time>{formatTime(item.duration ?? 0)}</time>
                </button>
              );
            })}
          </div>
        </section>
      )}
    </div>
  );
}

function trackURL(track: MusicTrack) {
  return `${API_BASE_URL}/v1/media/${track.media_id}/content`;
}

function formatTime(value: number) {
  if (!Number.isFinite(value) || value < 0) return "0:00";
  const totalSeconds = Math.floor(value);
  return `${Math.floor(totalSeconds / 60)}:${String(totalSeconds % 60).padStart(2, "0")}`;
}
