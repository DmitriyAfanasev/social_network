export type MusicTrack = {
  id: string;
  media_id: string;
  user_id: string;
  title: string;
  artist: string;
  duration: number | null;
  created_at: string;
  is_saved: boolean;
};

export type MusicResponse = {
  tracks: MusicTrack[];
};
