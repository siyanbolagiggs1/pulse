"use client";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useAuthStore } from "@/store/auth";

type Listener = (data: unknown) => void;

interface RealtimeContextValue {
  subscribe: (eventType: string, callback: Listener) => () => void;
  connected: boolean;
}

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

function wsURLFromApiURL(apiURL: string): string {
  return apiURL.replace(/^http/, "ws");
}

// Single shared WebSocket connection per session, carrying both notifications
// and chat events. Replaces the old SSE-based useSSE hook — subscribe() lets
// multiple consumers (the notification bell, chat UI) listen for different
// envelope types over the one connection instead of each opening their own.
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((s) => s.user);
  const listenersRef = useRef<Map<string, Set<Listener>>>(new Map());
  const wsRef = useRef<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);

  const subscribe = useCallback((eventType: string, callback: Listener) => {
    const listeners = listenersRef.current;
    if (!listeners.has(eventType)) listeners.set(eventType, new Set());
    listeners.get(eventType)!.add(callback);
    return () => {
      listeners.get(eventType)?.delete(callback);
    };
  }, []);

  useEffect(() => {
    if (!user) return;

    let cancelled = false;
    const retryDelay = { ms: 3000 };

    const connect = () => {
      if (cancelled) return;
      const token =
        typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
      if (!token) return;

      const baseURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:5000/api";
      const ws = new WebSocket(`${wsURLFromApiURL(baseURL)}/ws?token=${encodeURIComponent(token)}`);
      wsRef.current = ws;

      ws.onopen = () => {
        retryDelay.ms = 3000; // reset backoff on successful connection
        setConnected(true);
      };

      ws.onmessage = (event) => {
        try {
          const envelope = JSON.parse(event.data) as { type: string; data: unknown };
          listenersRef.current.get(envelope.type)?.forEach((cb) => cb(envelope.data));
        } catch {
          // malformed frame — ignore
        }
      };

      ws.onclose = () => {
        setConnected(false);
        if (cancelled) return;
        setTimeout(() => {
          retryDelay.ms = Math.min(retryDelay.ms * 2, 30000);
          connect();
        }, retryDelay.ms);
      };

      ws.onerror = () => {
        ws.close();
      };
    };

    connect();

    return () => {
      cancelled = true;
      wsRef.current?.close();
    };
  }, [user]);

  const value = useMemo(() => ({ subscribe, connected }), [subscribe, connected]);

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime() {
  const ctx = useContext(RealtimeContext);
  if (!ctx) throw new Error("useRealtime must be used within a RealtimeProvider");
  return ctx;
}
