import type { Metadata } from "next";
import Link from "next/link";
import { JetBrains_Mono } from "next/font/google";
import { ClerkProvider, Show, SignInButton, UserButton } from "@clerk/nextjs";

import ConvexClientProvider from "../components/ConvexClientProvider";
import { hasBrowserTeamsEnv } from "../lib/env";
import ThemeScript from "../components/ThemeScript";
import ThemeToggle from "../components/ThemeToggle";
import TerminalBackground from "../components/TerminalBackground";
import Brand from "../components/Brand";
import Toaster from "../components/ui/Toaster";
import DialogHost from "../components/ui/DialogHost";
import "./globals.css";

const jetbrains = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800"],
  variable: "--font-jetbrains",
  display: "swap",
});

export const metadata: Metadata = {
  title: "SSHThing Teams — cloud-backed SSH access",
  description:
    "Browser authentication, team management, and host handoff for the SSHThing terminal app.",
};

// Intentionally minimal. Hardcoded colors in `variables` would render
// black-on-black (or cream-on-cream) in the opposite theme; we rely on
// globals.css `!important` overrides keyed to our CSS variables so Clerk
// elements flip with the theme automatically.
const clerkAppearance = {
  variables: {
    borderRadius: "2px",
    fontFamily: "var(--font-jetbrains), ui-monospace, monospace",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const publishableKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;
  const hasConvexURL = Boolean(process.env.NEXT_PUBLIC_CONVEX_URL);
  const hasBrowserEnv = hasBrowserTeamsEnv();

  const inner = publishableKey && hasConvexURL ? (
    <ConvexClientProvider>{children}</ConvexClientProvider>
  ) : (
    children
  );

  const chrome = (
    <>
      <TerminalBackground />
      <div className="site">
        <header className="site__header">
          <div className="shell site__header-inner">
            <Brand />
            <nav className="site__nav" aria-label="Primary">
              {hasBrowserEnv ? (
                <>
                  <Show when="signed-out">
                    <SignInButton mode="modal">
                      <button className="btn btn--ghost hide-sm" type="button">
                        Log in
                      </button>
                    </SignInButton>
                    <Link className="btn btn--primary" href="/signup">
                      Start
                    </Link>
                  </Show>
                  <Show when="signed-in">
                    <Link className="btn btn--ghost hide-sm" href="/teams">
                      Teams
                    </Link>
                    <UserButton
                      appearance={{
                        elements: {
                          avatarBox: "cl-userButtonAvatarBox",
                        },
                      }}
                    />
                  </Show>
                </>
              ) : null}
              <ThemeToggle />
            </nav>
          </div>
        </header>

        {inner}

        <footer className="site__footer">
          <div className="shell site__footer-inner">
            <span className="mono">
              © {new Date().getFullYear()} SSHTHING · ALL SYSTEMS NOMINAL
            </span>
            <span className="mono muted">v2 · teams</span>
          </div>
        </footer>
      </div>
      <Toaster />
      <DialogHost />
    </>
  );

  return (
    <html lang="en" className={jetbrains.variable} suppressHydrationWarning>
      <head>
        <ThemeScript />
      </head>
      <body>
        {publishableKey ? (
          <ClerkProvider
            publishableKey={publishableKey}
            appearance={clerkAppearance}
          >
            {chrome}
          </ClerkProvider>
        ) : (
          chrome
        )}
      </body>
    </html>
  );
}
