export function authErrorStatus(error: unknown): number {
  const message = error instanceof Error ? error.message : "";
  switch (message) {
    case "not_authenticated":
    case "missing_user":
    case "missing_bearer_token":
    case "invalid_access_token":
      return 401;
    default:
      return 400;
  }
}
