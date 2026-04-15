import type { ReactNode } from "react";

export const metadata = {
  title: "SSHThing Teams",
  description: "Browser handoff, team bootstrap, and invite acceptance for SSHThing Teams."
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body style={{ fontFamily: "ui-sans-serif, system-ui", margin: 0, background: "#0b1020", color: "#f8fafc" }}>
        {children}
      </body>
    </html>
  );
}

