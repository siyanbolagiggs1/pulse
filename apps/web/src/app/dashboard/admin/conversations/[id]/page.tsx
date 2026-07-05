"use client";
import { useEffect, useState } from "react";
import { useParams, useSearchParams } from "next/navigation";
import Link from "next/link";
import { adminConversationsApi } from "@/lib/api";
import type { ChatMessage } from "@/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";
import { cn } from "@/lib/utils";

export default function AdminConversationThreadPage() {
  const { id } = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const aId = searchParams.get("aId");
  const aName = searchParams.get("aName") ?? "Participant A";
  const aRole = searchParams.get("aRole") ?? "";
  const bName = searchParams.get("bName") ?? "Participant B";
  const bRole = searchParams.get("bRole") ?? "";

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    adminConversationsApi.getMessages(id)
      .then((r) => setMessages((r.data.data ?? []).slice().reverse()))
      .catch(() => toast({ title: "Failed to load messages", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return <Skeleton className="h-96 w-full" />;

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/dashboard/admin/conversations"><ArrowLeft className="h-4 w-4" /></Link>
        </Button>
        <div>
          <h2 className="text-lg font-semibold">{aName} ↔ {bName}</h2>
          <p className="text-xs text-muted-foreground">Read-only oversight</p>
        </div>
      </div>

      <div className="space-y-3">
        {messages.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">No messages yet</p>
        ) : (
          messages.map((m) => {
            const isA = aId ? m.senderId === aId : false;
            return (
              <div key={m.id} className={cn("flex", isA ? "justify-end" : "justify-start")}>
                <div className={cn("max-w-[75%] rounded-lg border border-border p-3", isA && "bg-accent")}>
                  <div className="mb-1 flex items-center gap-2">
                    <Badge variant="secondary" className="text-[10px] capitalize">
                      {aId ? (isA ? aRole : bRole) : "sender"}
                    </Badge>
                  </div>
                  <p className="whitespace-pre-wrap break-words text-sm">{m.body}</p>
                  <p className="mt-1 text-[10px] text-muted-foreground">
                    {format(new Date(m.createdAt), "MMM d, yyyy · HH:mm")}
                  </p>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
