import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Quevia — Smart Queue & Appointment Management",
  description: "B2B SaaS Multi-Tenant Platform for Appointment and Queue Operations",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="id">
      <body className="min-h-screen bg-zinc-950 text-zinc-100 antialiased">
        {children}
      </body>
    </html>
  );
}
