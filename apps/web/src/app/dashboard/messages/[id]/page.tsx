"use client";
import { useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { conversationsApi } from "@/lib/api";
import type { ChatMessage, Conversation } from "@/types";
import { useAuthStore } from "@/store/auth";
import { useRealtime } from "@/providers/realtime";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, Send } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";
import { cn } from "@/lib/utils";

export default function ConversationThreadPage() {
  const { id } = useParams<{ id: string }>();
  const user = useAuthStore((s) => s.user);
  const { subscribe } = useRealtime();

  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [body, setBody] = useState("");
  const [sending, setSending] = useState(false);
  const [otherTyping, setOtherTyping] = useState(false);
  const [otherReadAt, setOtherReadAt] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastTypingSentRef = useRef(0);

  useEffect(() => {
    setLoading(true);
    Promise.all([conversationsApi.getMessages(id), conversationsApi.get(id)])
      .then(([msgRes, convRes]) => {
        setMessages((msgRes.data.data ?? []).slice().reverse());
        setConversation(convRes.data.data);
      })
      .catch(() => toast({ title: "Failed to load conversation", variant: "destructive" }))
      .finally(() => setLoading(false));
    conversationsApi.markRead(id).catch(() => {});
  }, [id]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    return subscribe("chat_message", ((msg: ChatMessage) => {
      if (msg.conversationId !== id) return;
      setMessages((prev) => [...prev, msg]);
      if (msg.senderId !== user?.id) {
        conversationsApi.markRead(id).catch(() => {});
      }
    }) as (d: unknown) => void);
  }, [subscribe, id, user?.id]);

  useEffect(() => {
    return subscribe("typing", ((data: { conversationId: string; userId: string }) => {
      if (data.conversationId !== id || data.userId === user?.id) return;
      setOtherTyping(true);
      if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
      typingTimeoutRef.current = setTimeout(() => setOtherTyping(false), 3000);
    }) as (d: unknown) => void);
  }, [subscribe, id, user?.id]);

  useEffect(() => {
    return subscribe("read_receipt", ((data: { conversationId: string; userId: string; readAt: string }) => {
      if (data.conversationId !== id || data.userId === user?.id) return;
      setOtherReadAt(data.readAt);
    }) as (d: unknown) => void);
  }, [subscribe, id, user?.id]);

  const handleBodyChange = (value: string) => {
    setBody(value);
    const now = Date.now();
    if (now - lastTypingSentRef.current > 1500) {
      lastTypingSentRef.current = now;
      conversationsApi.typing(id).catch(() => {});
    }
  };

  const handleSend = async () => {
    const trimmed = body.trim();
    if (!trimmed) return;
    setSending(true);
    try {
      const res = await conversationsApi.sendMessage(id, trimmed);
      setMessages((prev) => [...prev, res.data.data]);
      setBody("");
    } catch (err: any) {
      toast({ title: "Failed to send", description: err?.response?.data?.message, variant: "destructive" });
    } finally {
      setSending(false);
    }
  };

  if (loading) return <Skeleton className="h-96 w-full" />;

  const otherName = conversation?.otherParty.name ?? "Conversation";
  const lastOwnMessage = [...messages].reverse().find((m) => m.senderId === user?.id);
  const seen = !!(lastOwnMessage && otherReadAt && lastOwnMessage.createdAt <= otherReadAt);

  return (
    <div className="mx-auto flex h-full max-w-2xl flex-col">
      <div className="flex items-center gap-3 border-b border-border pb-3">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/dashboard/messages"><ArrowLeft className="h-4 w-4" /></Link>
        </Button>
        <Avatar className="h-8 w-8">
          <AvatarFallback>{otherName[0]?.toUpperCase() ?? "?"}</AvatarFallback>
        </Avatar>
        <div>
          <p className="font-medium">{otherName}</p>
          {otherTyping && <p className="text-xs text-primary">typing…</p>}
        </div>
      </div>

      <div className="flex-1 space-y-3 overflow-y-auto py-4">
        {messages.map((m) => {
          const mine = m.senderId === user?.id;
          return (
            <div key={m.id} className={cn("flex", mine ? "justify-end" : "justify-start")}>
              <div
                className={cn(
                  "max-w-[75%] rounded-lg px-3 py-2 text-sm",
                  mine ? "bg-primary text-primary-foreground" : "bg-accent"
                )}
              >
                <p className="whitespace-pre-wrap break-words">{m.body}</p>
                <p className={cn("mt-1 text-[10px]", mine ? "text-primary-foreground/70" : "text-muted-foreground")}>
                  {format(new Date(m.createdAt), "HH:mm")}
                </p>
              </div>
            </div>
          );
        })}
        {seen && <p className="text-right text-[10px] text-muted-foreground">Seen</p>}
        <div ref={bottomRef} />
      </div>

      <div className="flex items-center gap-2 border-t border-border pt-3">
        <Input
          placeholder="Type a message…"
          value={body}
          onChange={(e) => handleBodyChange(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleSend(); } }}
        />
        <Button size="icon" onClick={handleSend} disabled={sending || !body.trim()}>
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
