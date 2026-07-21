"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";
import { isJwtExpired, attemptRefresh } from "@/lib/refresh";

export default function Home() {
  const router = useRouter();
  const { setAuth, setLoading } = useAuthStore();

  useEffect(() => {
    (async () => {
      try {
        let token = localStorage.getItem("access_token");
        if (!token) { setLoading(false); router.replace("/login"); return; }

        if (isJwtExpired(token)) {
          const newToken = await attemptRefresh();
          if (!newToken) { setLoading(false); router.replace("/login"); return; }
          token = newToken;
          localStorage.setItem("access_token", token);
        }

        const res = await authApi.me();
        setAuth(res.data.data, token);
        router.replace("/dashboard");
      } catch {
        setLoading(false);
        router.replace("/login");
      }
    })();
  }, []);

  return null;
}
