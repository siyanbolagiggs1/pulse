"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";
import { isJwtExpired, attemptRefresh } from "@/lib/refresh";
import { Sidebar, MobileSidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { FloatingMessageButton } from "@/components/layout/floating-message-button";
import { RealtimeProvider } from "@/providers/realtime";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, isLoading, setAuth, clearAuth, setLoading } = useAuthStore();
  const [mobileOpen, setMobileOpen] = useState(false);
  const userRef = useRef(user);
  useEffect(() => { userRef.current = user; }, [user]);

  // Cold-start auth: runs once on mount when there is no in-memory user.
  useEffect(() => {
    if (user) return;

    (async () => {
      try {
        let token = localStorage.getItem("access_token");
        if (!token) { clearAuth(); router.replace("/login"); return; }

        if (isJwtExpired(token)) {
          const newToken = await attemptRefresh();
          if (!newToken) { clearAuth(); router.replace("/login"); return; }
          token = newToken;
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

  // Foreground-resume refresh: when the app comes back from background and the
  // access token has expired, refresh it before any API call needs to.
  useEffect(() => {
    const handleVisibility = async () => {
      if (document.visibilityState !== "visible" || !userRef.current) return;
      const token = localStorage.getItem("access_token");
      if (!token || !isJwtExpired(token)) return;

      const newToken = await attemptRefresh();
      if (newToken) {
        localStorage.setItem("access_token", newToken);
        setAuth(userRef.current, newToken);
      } else {
        clearAuth();
        router.replace("/login");
      }
    };

    document.addEventListener("visibilitychange", handleVisibility);
    return () => document.removeEventListener("visibilitychange", handleVisibility);
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
    <RealtimeProvider>
      <div className="flex h-screen overflow-hidden">
        <Sidebar />
        <MobileSidebar open={mobileOpen} onClose={() => setMobileOpen(false)} />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header onMenuClick={() => setMobileOpen(true)} />
          <main className="flex-1 overflow-y-auto p-4 md:p-6">{children}</main>
        </div>
      </div>
      <FloatingMessageButton />
    </RealtimeProvider>
  );
}
