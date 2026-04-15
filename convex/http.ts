import { httpRouter } from "convex/server";
import { httpAction } from "./_generated/server";

const http = httpRouter();

const notImplemented = httpAction(async ({ request }) => {
  const url = new URL(request.url);

  return new Response(
    JSON.stringify({
      ok: false,
      error: "not_implemented",
      path: url.pathname
    }),
    {
      status: 501,
      headers: { "content-type": "application/json" }
    }
  );
});

http.route({ path: "/cli-auth/start", method: "POST", handler: notImplemented });
http.route({ path: "/cli-auth/poll", method: "POST", handler: notImplemented });
http.route({ path: "/cli-auth/refresh", method: "POST", handler: notImplemented });
http.route({ path: "/cli-auth/logout", method: "POST", handler: notImplemented });
http.route({ path: "/teams/me", method: "GET", handler: notImplemented });
http.route({ path: "/teams/current/hosts", method: "GET", handler: notImplemented });
http.route({ path: "/teams/current/hosts", method: "POST", handler: notImplemented });
http.route({ path: "/teams/current/members", method: "GET", handler: notImplemented });
http.route({ path: "/teams/current/invites", method: "POST", handler: notImplemented });

export default http;
