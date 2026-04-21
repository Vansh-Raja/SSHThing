import { auth } from "@clerk/nextjs/server";
import { SignInButton } from "@clerk/nextjs";

import InviteAcceptView from "../../../../components/InviteAcceptView";

type PageProps = {
  params: Promise<{ inviteId: string }>;
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export default async function TeamInvitePage({ params, searchParams }: PageProps) {
  const { inviteId } = await params;
  const resolvedSearchParams = (await searchParams) ?? {};
  const token = Array.isArray(resolvedSearchParams.token)
    ? resolvedSearchParams.token[0]
    : resolvedSearchParams.token;
  const { userId } = await auth();

  return (
    <main className="shell" style={{ padding: "48px 0 64px" }}>
      {userId ? (
        <InviteAcceptView inviteId={inviteId} token={token} />
      ) : (
        <div className="block stack" style={{ maxWidth: 640, margin: "0 auto" }}>
          <span className="eyebrow">Sign in required</span>
          <h1 className="text-xl fw-800">Log in to review this team invite.</h1>
          <p className="muted text-sm">
            Invite acceptance is tied to your Clerk account so SSHThing can bind the team membership to the right user.
          </p>
          <div className="row">
            <SignInButton mode="modal">
              <button className="btn btn--primary" type="button">
                Log in
              </button>
            </SignInButton>
          </div>
        </div>
      )}
    </main>
  );
}
