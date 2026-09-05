const DEFAULT_API_BASE_URL = `${window.location.protocol}//${window.location.hostname}:8000`;

export const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? DEFAULT_API_BASE_URL).replace(/\/$/, "");
