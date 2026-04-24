/**
 * Thin fetch wrapper used by every Teams API call (browser-side).
 *
 * Sets JSON content type, parses the response, and throws a plain Error with
 * the server-provided error code (or a generic fallback) when the status is
 * not ok. Centralising this keeps each page / drawer handler short and makes
 * the "request_failed" fallback consistent everywhere.
 */
export async function apiRequest<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });
  const data = (await response.json().catch(() => ({}))) as T & {
    error?: string;
  };
  if (!response.ok) {
    throw new Error(data.error || "request_failed");
  }
  return data;
}

export function errorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}
