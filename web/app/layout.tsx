import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Voter Portal | TPT NZ",
  description:
    "Participate in secure, RealMe-verified local body polling. Your vote is anonymous and publicly auditable.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <header className="bg-white border-b border-gray-200">
          <nav className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
            <a href="/" className="text-xl font-bold text-gray-900">
              VoterPortal
            </a>
            <div className="flex items-center gap-4">
              <a
                href="/polls"
                className="text-sm font-medium text-gray-700 hover:text-gray-900"
              >
                Active Polls
              </a>
              <a
                href="/register"
                className="text-sm font-medium text-gray-700 hover:text-gray-900"
              >
                Register
              </a>
              <a
                href="/auth/login"
                className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-700 hover:bg-blue-800"
              >
                Sign In with RealMe
              </a>
            </div>
          </nav>
        </header>
        <main className="min-h-screen bg-gray-50">{children}</main>
        <footer className="bg-white border-t border-gray-200">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <p className="text-sm text-gray-500 text-center">
              TPT NZ Voter Portal — local body polling only. Powered by RealMe
              Verified Identity. Results are publicly auditable.
            </p>
          </div>
        </footer>
      </body>
    </html>
  );
}
