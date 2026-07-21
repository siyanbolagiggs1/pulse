"use client";
import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { conversationsApi, usersApi } from "@/lib/api";
import type { Conversation, ChatMessage } from "@/types";
import { useRealtime } from "@/providers/realtime";
import { useAuthStore } from "@/store/auth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { formatDistanceToNow } from "date-fns";
import { toast } from "@/components/ui/use-toast";
import { Search, LifeBuoy } from "lucide-react";

interface SearchResult {
  id: string;
  name: string;
  avatar?: string;
}

export default function MessagesPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [gettingHelp, setGettingHelp] = useState(false);
  const { subscribe, refreshUnreadMessages } = useRealtime();

  const load = useCallback(() => {
    setLoading(true);
    conversationsApi.list()
      .then((r) => setConversations(r.data.data ?? []))
      .catch(() => toast({ title: "Failed to load conversations", variant: "destructive" }))
      .finally(() => setLoading(false));
    refreshUnreadMessages();
  }, [refreshUnreadMessages]);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    return subscribe("chat_message", ((msg: ChatMessage) => {
      setConversations((prev) => {
        const idx = prev.findIndex((c) => c.id === msg.conversationId);
        if (idx === -1) {
          // A new conversation someone else started — refetch to pick it up.
          load();
          return prev;
        }
        const updated: Conversation = {
          ...prev[idx],
          lastMessageAt: msg.createdAt,
          lastMessagePreview: msg.body,
          unreadCount: prev[idx].unreadCount + 1,
        };
        const next = prev.filter((_, i) => i !== idx);
        return [updated, ...next];
      });
    }) as (d: unknown) => void);
  }, [subscribe, load]);

  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    setSearching(true);
    const t = setTimeout(() => {
      usersApi.search(query.trim())
        .then((r) => setResults(r.data.data ?? []))
        .catch(() => setResults([]))
        .finally(() => setSearching(false));
    }, 300);
    return () => clearTimeout(t);
  }, [query]);

  const startConversation = async (recipientId: string) => {
    try {
      const res = await conversationsApi.start(recipientId);
      setQuery("");
      setResults([]);
      router.push(`/dashboard/messages/${res.data.data.id}`);
    } catch (err: any) {
      toast({ title: "Failed to start conversation", description: err?.response?.data?.message, variant: "destructive" });
    }
  };

  const handleHelp = async () => {
    setGettingHelp(true);
    try {
      const res = await conversationsApi.startSupport();
      router.push(`/dashboard/messages/${res.data.data.id}`);
    } catch (err: any) {
      toast({ title: "Couldn't open support chat", description: err?.response?.data?.message, variant: "destructive" });
    } finally {
      setGettingHelp(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold">Messages</h2>
          <p className="text-muted-foreground">Direct messages with other Pulse users</p>
        </div>
        {user?.role !== "admin" && (
          <Button variant="outline" onClick={handleHelp} disabled={gettingHelp}>
            <LifeBuoy className="mr-2 h-4 w-4" />
            {gettingHelp ? "Opening…" : "Help"}
          </Button>
        )}
      </div>

      <Card>
        <CardHeader><CardTitle>New Message</CardTitle></CardHeader>
        <CardContent className="space-y-2">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder="Search by name…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
          {searching && <p className="text-xs text-muted-foreground">Searching…</p>}
          {results.length > 0 && (
            <div className="space-y-1 rounded-md border border-border p-1">
              {results.map((u) => (
                <button
                  key={u.id}
                  onClick={() => startConversation(u.id)}
                  className="flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent"
                >
                  <Avatar className="h-7 w-7">
                    <AvatarFallback>{u.name?.[0]?.toUpperCase() ?? "?"}</AvatarFallback>
                  </Avatar>
                  {u.name}
                </button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Conversations</CardTitle></CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : conversations.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No conversations yet — search above to start one.</p>
          ) : (
            <div className="space-y-2">
              {conversations.map((c) => (
                <button
                  key={c.id}
                  onClick={() => router.push(`/dashboard/messages/${c.id}`)}
                  className="flex w-full items-center gap-3 rounded-lg border border-border p-3 text-left hover:bg-accent"
                >
                  <Avatar className="h-9 w-9">
                    <AvatarFallback>{c.otherParty.name?.[0]?.toUpperCase() ?? "?"}</AvatarFallback>
                  </Avatar>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="truncate font-medium">{c.otherParty.name}</span>
                      {c.lastMessagePreview && (
                        <span className="shrink-0 text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(c.lastMessageAt), { addSuffix: true })}
                        </span>
                      )}
                    </div>
                    <p className="truncate text-sm text-muted-foreground">{c.lastMessagePreview || "No messages yet"}</p>
                    {c.needsAdminReview && (
                      <Badge variant="warning" className="mt-1 text-[10px]">Awaiting admin</Badge>
                    )}
                  </div>
                  {c.unreadCount > 0 && <Badge>{c.unreadCount > 9 ? "9+" : c.unreadCount}</Badge>}
                </button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
