import type { Metadata } from "next";
import { ClerkProvider, Show, SignInButton, SignUpButton, UserButton } from "@clerk/nextjs";

import ConvexClientProvider from "../components/ConvexClientProvider";
import "./globals.css";

export const metadata: Metadata = {
  title: "SSHThing Teams",
  description: "Browser auth handoff and team management for SSHThing Teams.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  const publishableKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;
  const hasConvexURL = Boolean(process.env.NEXT_PUBLIC_CONVEX_URL);

  const content = publishableKey && hasConvexURL
    ? <ConvexClientProvider>{children}</ConvexClientProvider>
    : children;

  if (!publishableKey) {
    return (
      <html lang="en">
        <body>{content}</body>
      </html>
    );
  }

  return (
    <html lang="en">
      <body>
        <ClerkProvider publishableKey={publishableKey}>
          <header className="shell" style={{ paddingTop: 24, paddingBottom: 0 }}>
            <div className="card" style={{ padding: 16 }}>
              <div className="actionRow" style={{ justifyContent: "space-between", alignItems: "center" }}>
                <div className="pillRow">
                  <span className="pill">SSHThing Teams</span>
                  <span className="pill">Clerk</span>
                </div>
                <div className="actionRow">
                  <Show when="signed-out">
                    <SignInButton mode="modal">
                      <button className="buttonLink buttonSecondary" type="button">
                        Sign in
                      </button>
                    </SignInButton>
                    <SignUpButton mode="modal">
                      <button className="buttonLink buttonPrimary" type="button">
                        Sign up
                      </button>
                    </SignUpButton>
                  </Show>
                  <Show when="signed-in">
                    <UserButton />
                  </Show>
                </div>
              </div>
            </div>
          </header>
          {content}
        </ClerkProvider>
      </body>
    </html>
  );
}
