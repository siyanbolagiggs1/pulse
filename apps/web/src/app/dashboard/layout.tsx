"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";
import { Sidebar, MobileSidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";

function isJwtExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return Date.now() / 1000 >= payload.exp;
  } catch {
    return true;
  }
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, isLoading, setAuth, clearAuth, setLoading } = useAuthStore();
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    if (user) return;

    (async () => {
      try {
        let token = localStorage.getItem("access_token");
        if (!token) { clearAuth(); router.replace("/login"); return; }

        // Proactively refresh before calling /auth/me if the access token is already
        // expired — avoids a 401 round-trip and prevents setAuth from overwriting the
        // refreshed token with the stale one (race condition on PWA cold-start).
        if (isJwtExpired(token)) {
          const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api";
          const res = await fetch(`${base}/auth/refresh`, {
            method: "POST",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
          });
          if (!res.ok) { clearAuth(); router.replace("/login"); return; }
          const body = await res.json();
          token = body.data.accessToken as string;
          localStorage.setItem("access_token", token);
        }

        const res = await authApi.me();
        setAuth(res.data.data, token);
      } catch {
        clearAuth();
        router.replace("/login");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  if (!user) return null;

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <MobileSidebar open={mobileOpen} onClose={() => setMobileOpen(false)} />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header onMenuClick={() => setMobileOpen(true)} />
        <main className="flex-1 overflow-y-auto p-4 md:p-6">{children}</main>
      </div>
    </div>
  );
}
