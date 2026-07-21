"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { adminConversationsApi } from "@/lib/api";
import type { AdminConversation } from "@/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { formatDistanceToNow } from "date-fns";
import { toast } from "@/components/ui/use-toast";

export default function AdminConversationsPage() {
  const router = useRouter();
  const [conversations, setConversations] = useState<AdminConversation[]>([]);
  const [loading, setLoading] = useState(true);
  const [broadcastOpen, setBroadcastOpen] = useState(false);
  const [broadcasting, setBroadcasting] = useState(false);

  const handleBroadcastWelcome = async () => {
    setBroadcasting(true);
    try {
      const res = await adminConversationsApi.broadcastWelcome();
      const { sent, skipped } = res.data.data;
      toast({ title: "Welcome messages sent", description: `Sent to ${sent} user(s), skipped ${skipped} (already welcomed).` });
      setBroadcastOpen(false);
    } catch (err: any) {
      toast({ title: "Failed to send", description: err?.response?.data?.message, variant: "destructive" });
    } finally {
      setBroadcasting(false);
    }
  };

  useEffect(() => {
    adminConversationsApi.list()
      .then((r) => setConversations(r.data.data ?? []))
      .catch(() => toast({ title: "Failed to load conversations", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, []);

  const openThread = (c: AdminConversation) => {
    const params = new URLSearchParams({
      aId: c.participantA.id,
      aName: c.participantA.name,
      aRole: c.participantA.role,
      bId: c.participantB.id,
      bName: c.participantB.name,
      bRole: c.participantB.role,
    });
    router.push(`/dashboard/admin/conversations/${c.id}?${params.toString()}`);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold">Conversations</h2>
          <p className="text-muted-foreground">Read-only oversight of all conversations</p>
        </div>
        <Button variant="outline" onClick={() => setBroadcastOpen(true)}>
          Send welcome message to existing users
        </Button>
      </div>

      <Dialog open={broadcastOpen} onOpenChange={setBroadcastOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Send welcome message to existing users?</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Sends the one-time welcome message from support to every business and promoter who
            doesn&apos;t already have one. Safe to run more than once — anyone already welcomed is skipped.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBroadcastOpen(false)}>Cancel</Button>
            <Button onClick={handleBroadcastWelcome} disabled={broadcasting}>
              {broadcasting ? "Sending…" : "Send"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
                      <AvatarFallback>{c.participantA.name?.[0]?.toUpperCase() ?? "?"}</AvatarFallback>
                    </Avatar>
                    <Avatar className="h-8 w-8 border-2 border-card">
                      <AvatarFallback>{c.participantB.name?.[0]?.toUpperCase() ?? "?"}</AvatarFallback>
                    </Avatar>
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="truncate font-medium">{c.participantA.name}</span>
                      <Badge variant="secondary" className="capitalize">{c.participantA.role}</Badge>
                      <span className="text-muted-foreground">↔</span>
                      <span className="truncate font-medium">{c.participantB.name}</span>
                      <Badge variant="secondary" className="capitalize">{c.participantB.role}</Badge>
                      {c.needsAdminReview && <Badge variant="warning">Awaiting admin</Badge>}
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
