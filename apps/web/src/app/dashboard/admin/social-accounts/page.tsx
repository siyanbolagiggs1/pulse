"use client";
import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { ExternalLink, CheckCircle, XCircle } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

interface PendingAccount {
  id: string;
  userId: string;
  platform: string;
  username: string;
  profileUrl: string;
  createdAt: string;
}

export default function AdminSocialAccountsPage() {
  const [accounts, setAccounts] = useState<PendingAccount[]>([]);
  const [loading, setLoading] = useState(true);

  const [approveOpen, setApproveOpen] = useState(false);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [selected, setSelected] = useState<PendingAccount | null>(null);
  const [processing, setProcessing] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const [stats, setStats] = useState({ followerCount: "", followingCount: "", engagementRate: "", accountAgeDays: "" });

  const load = () => {
    setLoading(true);
    adminApi.listPendingSocialAccounts()
      .then((r) => setAccounts(r.data.data ?? []))
      .catch(() => toast({ title: "Failed to load", variant: "destructive" }))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const openApprove = (a: PendingAccount) => {
    setSelected(a);
    setStats({ followerCount: "", followingCount: "", engagementRate: "", accountAgeDays: "" });
    setApproveOpen(true);
  };

  const openReject = (a: PendingAccount) => {
    setSelected(a);
    setRejectReason("");
    setRejectOpen(true);
  };

  const handleApprove = async () => {
    if (!selected) return;
    if (!stats.followerCount || !stats.accountAgeDays) {
      return toast({ title: "Follower count and account age are required", variant: "destructive" });
    }
    setProcessing(true);
    try {
      await adminApi.approveSocialAccount(selected.id, {
        followerCount: Number(stats.followerCount),
        followingCount: Number(stats.followingCount) || 0,
        engagementRate: Number(stats.engagementRate) || 0,
        accountAgeDays: Number(stats.accountAgeDays),
      });
      toast({ title: "Account approved" });
      setApproveOpen(false);
      load();
    } catch (err: any) {
      toast({ title: "Failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setProcessing(false); }
  };

  const handleReject = async () => {
    if (!selected || !rejectReason.trim()) return;
    setProcessing(true);
    try {
      await adminApi.rejectSocialAccount(selected.id, rejectReason.trim());
      toast({ title: "Account rejected" });
      setRejectOpen(false);
      load();
    } catch (err: any) {
      toast({ title: "Failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setProcessing(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Social Account Verification</h2>
        <p className="text-muted-foreground">Review pending Instagram and TikTok account submissions</p>
      </div>

      <Card>
        <CardHeader><CardTitle>Pending Review ({accounts.length})</CardTitle></CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : accounts.length === 0 ? (
            <p className="text-center py-8 text-muted-foreground">No pending accounts</p>
          ) : (
            <div className="space-y-3">
              {accounts.map((a) => (
                <div key={a.id} className="flex items-center justify-between rounded-lg border border-border p-4">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">@{a.username}</span>
                      <Badge variant="secondary" className="capitalize">{a.platform}</Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Submitted {format(new Date(a.createdAt), "MMM d, yyyy · HH:mm")}
                    </p>
                    {a.profileUrl && (
                      <a href={a.profileUrl} target="_blank" rel="noreferrer"
                        className="inline-flex items-center gap-1 text-xs text-primary hover:underline">
                        View profile <ExternalLink className="h-3 w-3" />
                      </a>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => openApprove(a)}>
                      <CheckCircle className="mr-1 h-4 w-4 text-green-400" />Approve
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => openReject(a)}>
                      <XCircle className="mr-1 h-4 w-4 text-destructive" />Reject
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Approve dialog */}
      <Dialog open={approveOpen} onOpenChange={setApproveOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Approve @{selected?.username} ({selected?.platform})</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Visit their profile and enter the real stats below.
          </p>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label>Followers <span className="text-destructive">*</span></Label>
              <Input type="number" placeholder="10000" value={stats.followerCount}
                onChange={(e) => setStats((s) => ({ ...s, followerCount: e.target.value }))} />
            </div>
            <div className="space-y-1">
              <Label>Following</Label>
              <Input type="number" placeholder="500" value={stats.followingCount}
                onChange={(e) => setStats((s) => ({ ...s, followingCount: e.target.value }))} />
            </div>
            <div className="space-y-1">
              <Label>Engagement Rate (%)</Label>
              <Input type="number" placeholder="3.5" step="0.1" value={stats.engagementRate}
                onChange={(e) => setStats((s) => ({ ...s, engagementRate: e.target.value }))} />
            </div>
            <div className="space-y-1">
              <Label>Account Age (days) <span className="text-destructive">*</span></Label>
              <Input type="number" placeholder="365" value={stats.accountAgeDays}
                onChange={(e) => setStats((s) => ({ ...s, accountAgeDays: e.target.value }))} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setApproveOpen(false)}>Cancel</Button>
            <Button onClick={handleApprove} disabled={processing}>Approve Account</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reject dialog */}
      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reject @{selected?.username}</DialogTitle>
          </DialogHeader>
          <div className="space-y-1">
            <Label>Reason</Label>
            <Input placeholder="Profile not found / stats don't match…"
              value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectOpen(false)}>Cancel</Button>
            <Button variant="destructive" onClick={handleReject} disabled={processing || !rejectReason.trim()}>
              Reject
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
