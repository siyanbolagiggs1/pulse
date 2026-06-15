"use client";
import { useEffect, useState } from "react";
import { useAuthStore } from "@/store/auth";
import { usersApi } from "@/lib/api";
import type { SocialAccount } from "@/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { formatNumber } from "@/lib/utils";
import { Trash2, Plus, ExternalLink, Loader2 } from "lucide-react";
import { toast } from "@/components/ui/use-toast";

type Platform = "instagram" | "twitter" | "tiktok";

const platformLabels: Record<Platform, string> = {
  instagram: "Instagram",
  twitter: "Twitter / X",
  tiktok: "TikTok",
};

const statusBadge = (status: string) => {
  if (status === "active") return <Badge variant="success">Verified</Badge>;
  if (status === "pending_review") return <Badge variant="warning">Pending Review</Badge>;
  if (status === "rejected") return <Badge variant="destructive">Rejected</Badge>;
  return null;
};

export default function ProfilePage() {
  const user = useAuthStore((s) => s.user);
  const updateUser = useAuthStore((s) => s.updateUser);

  const [name, setName] = useState(user?.name ?? "");
  const [savingProfile, setSavingProfile] = useState(false);

  const [accounts, setAccounts] = useState<SocialAccount[]>([]);
  const [loadingAccounts, setLoadingAccounts] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const [connectOpen, setConnectOpen] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [platform, setPlatform] = useState<Platform>("twitter");
  const [profileUrl, setProfileUrl] = useState("");

  useEffect(() => {
    if (user?.role !== "promoter") return;
    setLoadingAccounts(true);
    usersApi.getMe()
      .then((r) => setAccounts(r.data.data.socialAccounts ?? []))
      .catch(() => toast({ title: "Failed to load accounts", variant: "destructive" }))
      .finally(() => setLoadingAccounts(false));
  }, [user?.role]);

  const handleSaveProfile = async () => {
    if (!name.trim()) return;
    setSavingProfile(true);
    try {
      await usersApi.updateProfile({ name: name.trim() });
      if (user) updateUser({ ...user, name: name.trim() });
      toast({ title: "Profile updated" });
    } catch {
      toast({ title: "Failed to update profile", variant: "destructive" });
    } finally { setSavingProfile(false); }
  };

  const handleDeleteAccount = async (id: string) => {
    setDeletingId(id);
    try {
      await usersApi.deleteSocialAccount(id);
      setAccounts((prev) => prev.filter((a) => a.id !== id));
      toast({ title: "Account removed" });
    } catch {
      toast({ title: "Failed to remove account", variant: "destructive" });
    } finally { setDeletingId(null); }
  };

  const handleConnect = async () => {
    if (!profileUrl.trim()) return toast({ title: "Profile URL is required", variant: "destructive" });

    setConnecting(true);
    try {
      const res = await usersApi.connectSocialAccount({
        platform,
        profileUrl: profileUrl.trim(),
      });
      setAccounts((prev) => [...prev, res.data.data]);
      setConnectOpen(false);
      setProfileUrl("");

      toast({
        title: "Submitted for review",
        description: "An admin will verify your account details within 24 hours.",
      });
    } catch (err: any) {
      toast({ title: "Failed to connect", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setConnecting(false); }
  };

  const resetDialog = () => {
    setPlatform("twitter");
    setProfileUrl("");
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Profile</h2>
        <p className="text-muted-foreground">Manage your account details</p>
      </div>

      <Card>
        <CardHeader><CardTitle>Account Details</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/20 text-2xl font-bold text-primary">
              {user?.name?.[0]?.toUpperCase()}
            </div>
            <div>
              <p className="font-medium">{user?.name}</p>
              <p className="text-sm text-muted-foreground">{user?.email}</p>
              <Badge variant="secondary" className="mt-1 capitalize">{user?.role}</Badge>
            </div>
          </div>

          <Separator />

          <div className="space-y-2">
            <Label htmlFor="name">Display Name</Label>
            <div className="flex gap-2">
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} className="flex-1" />
              <Button onClick={handleSaveProfile} disabled={savingProfile || name === user?.name}>
                {savingProfile ? "Saving…" : "Save"}
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 pt-1 text-sm">
            <div>
              <p className="text-muted-foreground">Email verified</p>
              <p>{user?.isEmailVerified ? "Yes" : "No"}</p>
            </div>
            {user?.role === "promoter" && (
              <div>
                <p className="text-muted-foreground">Trust score</p>
                <p className={user.trustScore < 30 ? "text-destructive" : user.trustScore < 60 ? "text-yellow-400" : "text-green-400"}>
                  {user.trustScore} / 100
                </p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {user?.role === "promoter" && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Social Accounts</CardTitle>
              <Button size="sm" onClick={() => { resetDialog(); setConnectOpen(true); }}>
                <Plus className="mr-2 h-4 w-4" />Connect
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {loadingAccounts ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : accounts.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4 text-center">No accounts connected yet.</p>
            ) : (
              <div className="space-y-3">
                {accounts.map((a) => (
                  <div key={a.id} className="flex items-start justify-between rounded-lg border border-border p-3">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-medium">@{a.username}</span>
                        <Badge variant="secondary" className="capitalize">{a.platform}</Badge>
                        {statusBadge(a.status)}
                      </div>
                      {a.status === "active" && (
                        <div className="flex gap-4 text-xs text-muted-foreground">
                          <span>{formatNumber(a.followerCount)} followers</span>
                          <span>{a.engagementRate.toFixed(1)}% engagement</span>
                          <span>Score {a.influenceScore.toFixed(0)}</span>
                        </div>
                      )}
                      {a.status === "rejected" && a.rejectedReason && (
                        <p className="text-xs text-destructive">Reason: {a.rejectedReason}</p>
                      )}
                      {a.profileUrl && (
                        <a href={a.profileUrl} target="_blank" rel="noreferrer"
                          className="inline-flex items-center gap-1 text-xs text-primary hover:underline">
                          View profile <ExternalLink className="h-3 w-3" />
                        </a>
                      )}
                    </div>
                    <Button
                      variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive"
                      onClick={() => handleDeleteAccount(a.id)}
                      disabled={deletingId === a.id}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <Dialog open={connectOpen} onOpenChange={(o) => { setConnectOpen(o); if (!o) resetDialog(); }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>Connect Social Account</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1">
              <Label>Platform</Label>
              <Select value={platform} onValueChange={(v: Platform) => { setPlatform(v); setProfileUrl(""); }}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="twitter">Twitter / X</SelectItem>
                  <SelectItem value="instagram">Instagram</SelectItem>
                  <SelectItem value="tiktok">TikTok</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1">
              <Label>Profile URL</Label>
              <Input
                placeholder={
                  platform === "instagram" ? "https://instagram.com/yourhandle" :
                  platform === "twitter" ? "https://twitter.com/yourhandle" :
                  "https://tiktok.com/@yourhandle"
                }
                value={profileUrl}
                onChange={(e) => setProfileUrl(e.target.value)}
              />
            </div>
            <div className="rounded-md bg-yellow-500/10 border border-yellow-500/30 p-3 text-sm text-yellow-400">
              Your account requires admin verification. An admin will visit your profile and confirm your stats within 24 hours.
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setConnectOpen(false)}>Cancel</Button>
            <Button onClick={handleConnect} disabled={connecting}>
              {connecting ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Submitting…</> : "Submit for Review"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
