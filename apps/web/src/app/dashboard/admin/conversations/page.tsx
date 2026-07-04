"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { adminConversationsApi } from "@/lib/api";
import type { AdminConversation } from "@/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { formatDistanceToNow } from "date-fns";
import { toast } from "@/components/ui/use-toast";

export default function AdminConversationsPage() {
  const router = useRouter();
  const [conversations, setConversations] = useState<AdminConversation[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    adminConversationsApi.list()
      .then((r) => setConversations(r.data.data ?? []))
      .catch(() => toast({ title: "Failed to load conversations", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, []);

  const openThread = (c: AdminConversation) => {
    const params = new URLSearchParams({
      businessId: c.business.id,
      businessName: c.business.name,
      promoterId: c.promoter.id,
      promoterName: c.promoter.name,
    });
    router.push(`/dashboard/admin/conversations/${c.id}?${params.toString()}`);
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Conversations</h2>
        <p className="text-muted-foreground">Read-only oversight of business ↔ promoter messages</p>
      </div>

      <Card>
        <CardHeader><CardTitle>All Conversations ({conversations.length})</CardTitle></CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : conversations.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No conversations yet</p>
          ) : (
            <div className="space-y-2">
              {conversations.map((c) => (
                <button
                  key={c.id}
                  onClick={() => openThread(c)}
                  className="flex w-full items-center gap-3 rounded-lg border border-border p-3 text-left hover:bg-accent"
                >
                  <div className="flex -space-x-2">
                    <Avatar className="h-8 w-8 border-2 border-card">
                      <AvatarFallback>{c.business.name?.[0]?.toUpperCase() ?? "?"}</AvatarFallback>
                    </Avatar>
                    <Avatar className="h-8 w-8 border-2 border-card">
                      <AvatarFallback>{c.promoter.name?.[0]?.toUpperCase() ?? "?"}</AvatarFallback>
                    </Avatar>
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="truncate font-medium">{c.business.name}</span>
                      <Badge variant="secondary">business</Badge>
                      <span className="text-muted-foreground">↔</span>
                      <span className="truncate font-medium">{c.promoter.name}</span>
                      <Badge variant="secondary">promoter</Badge>
                    </div>
                    <p className="truncate text-sm text-muted-foreground">{c.lastMessagePreview || "No messages yet"}</p>
                  </div>
                  {c.lastMessagePreview && (
                    <span className="shrink-0 text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(c.lastMessageAt), { addSuffix: true })}
                    </span>
                  )}
                </button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
