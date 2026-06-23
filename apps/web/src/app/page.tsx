"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";

function isJwtExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")));
    return Date.now() / 1000 >= payload.exp;
  } catch {
    return true;
  }
}

export default function Home() {
  const router = useRouter();
  const { setAuth, setLoading } = useAuthStore();

  useEffect(() => {
    (async () => {
      try {
        let token = localStorage.getItem("access_token");
        if (!token) { setLoading(false); router.replace("/login"); return; }

        if (isJwtExpired(token)) {
          const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api";
          const res = await fetch(`${base}/auth/refresh`, {
            method: "POST",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
          });
          if (!res.ok) { setLoading(false); router.replace("/login"); return; }
          const body = await res.json();
          token = body.data.accessToken as string;
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
