"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";

export default function Home() {
  const router = useRouter();
  const { setAuth, setLoading } = useAuthStore();

  useEffect(() => {
    const token = sessionStorage.getItem("access_token");
    if (token) {
      authApi.me()
        .then((res) => {
          setAuth(res.data.data, token);
          router.replace("/dashboard");
        })
        .catch(() => {
          setLoading(false);
          router.replace("/login");
        });
    } else {
      setLoading(false);
      router.replace("/login");
    }
  }, []);

  return null;
}
