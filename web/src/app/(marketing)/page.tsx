"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";
import { isJwtExpired, attemptRefresh } from "@/lib/refresh";
import { Hero } from "@/components/marketing/Hero";
import { HowItWorks } from "@/components/marketing/HowItWorks";
import { GettingStarted } from "@/components/marketing/GettingStarted";
import { WhyPulse } from "@/components/marketing/WhyPulse";

export default function Home() {
  const router = useRouter();
  const { setAuth } = useAuthStore();

  // Anonymous visitors see the landing page below. Anyone with a still-valid
  // (or refreshable) session gets sent straight to their dashboard, same as
  // before — this effect only ever redirects, never blocks/replaces the
  // landing page content while it runs.
  useEffect(() => {
    (async () => {
      let token = localStorage.getItem("access_token");
      if (!token) return;

      try {
        if (isJwtExpired(token)) {
          const newToken = await attemptRefresh();
          if (!newToken) return;
          token = newToken;
          localStorage.setItem("access_token", token);
        }
        const res = await authApi.me();
        setAuth(res.data.data, token);
        router.replace("/dashboard");
      } catch {
        // Invalid/expired token that couldn't refresh — stay on the landing page.
      }
    })();
  }, []);

  return (
    <main>
      <Hero />
      <HowItWorks />
      <GettingStarted />
      <WhyPulse />
    </main>
  );
}
