"use client";
import { useEffect, useRef, useCallback } from "react";
import { useAuthStore } from "@/store/auth";
import type { Notification } from "@/types";

export function useSSE(onNotification: (n: Notification) => void) {
  const user = useAuthStore((s) => s.user);
  const cbRef = useRef(onNotification);
  cbRef.current = onNotification;

  const connect = useCallback(async (signal: AbortSignal, retryDelay: { ms: number }) => {
    const token =
      typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
    if (!token || signal.aborted) return;

    const baseURL =
      process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api";

    try {
      const res = await fetch(`${baseURL}/notifications/stream`, {
        headers: { Authorization: `Bearer ${token}` },
        signal,
      });

      if (!res.ok || !res.body) return;

      retryDelay.ms = 3000; // reset backoff on successful connection

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (!signal.aborted) {
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
      // aborted or network error — fall through to reconnect
    }

    // Reconnect with exponential backoff (max 30s)
    if (!signal.aborted) {
      await new Promise((r) => setTimeout(r, retryDelay.ms));
      retryDelay.ms = Math.min(retryDelay.ms * 2, 30000);
      connect(signal, retryDelay);
    }
  }, []);

  useEffect(() => {
    if (!user) return;

    const controller = new AbortController();
    connect(controller.signal, { ms: 3000 });

    return () => controller.abort();
  }, [user, connect]);
}
