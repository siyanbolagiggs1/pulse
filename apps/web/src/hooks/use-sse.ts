"use client";
import { useEffect, useRef, useCallback } from "react";
import { useAuthStore } from "@/store/auth";
import type { Notification } from "@/types";

export function useSSE(onNotification: (n: Notification) => void) {
  const user = useAuthStore((s) => s.user);
  const cbRef = useRef(onNotification);
  cbRef.current = onNotification;

  useEffect(() => {
    if (!user) return;

    const token =
      typeof window !== "undefined" ? sessionStorage.getItem("access_token") : null;
    if (!token) return;

    const controller = new AbortController();
    const baseURL =
      process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api";

    let active = true;

    const connect = async () => {
      try {
        const res = await fetch(`${baseURL}/notifications/stream`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });

        if (!res.ok || !res.body) return;

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (active) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";

          for (const line of lines) {
            if (line.startsWith("data: ") && line.length > 6) {
              try {
                const parsed = JSON.parse(line.slice(6)) as Notification;
                if (parsed.id) cbRef.current(parsed);
              } catch {
                // malformed line — ignore
              }
            }
          }
        }
      } catch {
        // aborted or network error — ignore
      }
    };

    connect();

    return () => {
      active = false;
      controller.abort();
    };
  }, [user]);
}
