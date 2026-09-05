import { API_BASE_URL } from "../config/api";

export function isImageAttachment(value: string) {
  return /\.(avif|gif|jpe?g|png|webp)$/i.test(value.split("?")[0]);
}

export function resolveAttachmentUrl(value: string) {
  if (/^https?:\/\//i.test(value)) {
    return value;
  }

  return `${API_BASE_URL}${value.startsWith("/") ? value : `/${value}`}`;
}

export function getFileName(value: string) {
  return decodeURIComponent(value.split("/").at(-1) ?? "Файл");
}
