export function parseTags(raw: string): string[] {
  return raw
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}

export function formatTime(value: number | null | undefined): string {
  if (!value) {
    return "Never";
  }
  return new Date(value).toLocaleString();
}

export function formatRelative(value: number | null | undefined): string {
  if (!value) return "never";
  const now = Date.now();
  const diffSec = Math.round((now - value) / 1000);
  const absSec = Math.abs(diffSec);
  if (absSec < 60) return diffSec >= 0 ? "just now" : "in a moment";
  const diffMin = Math.round(diffSec / 60);
  const absMin = Math.abs(diffMin);
  if (absMin < 60) return diffSec >= 0 ? `${absMin}m ago` : `in ${absMin}m`;
  const diffH = Math.round(diffMin / 60);
  const absH = Math.abs(diffH);
  if (absH < 24) return diffSec >= 0 ? `${absH}h ago` : `in ${absH}h`;
  const diffD = Math.round(diffH / 24);
  const absD = Math.abs(diffD);
  if (absD < 30) return diffSec >= 0 ? `${absD}d ago` : `in ${absD}d`;
  return new Date(value).toLocaleDateString();
}
