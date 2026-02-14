import type { Metadata } from "next";
import Navigation from "@/components/Navigation";
import "@/styles/globals.css";

export const metadata: Metadata = {
  title: "Trading Dashboard",
  description: "Real-time trading performance dashboard",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <div className="min-h-screen flex flex-col">
          <header className="bg-white border-b border-gray-200 shadow-sm">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="flex items-center justify-between py-4">
                <div>
                  <h1 className="text-2xl font-bold text-gray-900">
                    📊 Trading Dashboard
                  </h1>
                  <p className="text-sm text-gray-600 mt-1">
                    Real-time algorithmic trading performance monitor
                  </p>
                </div>
                <Navigation />
              </div>
            </div>
          </header>
          <main className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-8">
            {children}
          </main>
          <footer className="bg-gray-50 border-t border-gray-200 mt-12">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
              <p className="text-sm text-gray-600">
                Dashboard backend: http://localhost:8081 | WebSocket:
                ws://localhost:8081/ws
              </p>
            </div>
          </footer>
        </div>
      </body>
    </html>
  );
}
