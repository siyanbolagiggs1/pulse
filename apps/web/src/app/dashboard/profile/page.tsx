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
import { Trash2, Plus, ExternalLink } from "lucide-react";
import { toast } from "@/components/ui/use-toast";

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
  const [form, setForm] = useState({
    platform: "instagram",
    username: "",
    profileUrl: "",
    followerCount: "",
    followingCount: "",
    engagementRate: "",
    accountAgeDays: "",
  });

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
    if (!form.username || !form.profileUrl || !form.followerCount) {
      return toast({ title: "Please fill all required fields", variant: "destructive" });
    }
    setConnecting(true);
    try {
      const res = await usersApi.connectSocialAccount({
        platform: form.platform,
        username: form.username,
        profileUrl: form.profileUrl,
        followerCount: Number(form.followerCount),
        followingCount: Number(form.followingCount) || 0,
        engagementRate: Number(form.engagementRate) || 0,
        accountAgeDays: Number(form.accountAgeDays) || 0,
      });
      setAccounts((prev) => [...prev, res.data.data]);
      setConnectOpen(false);
      setForm({ platform: "instagram", username: "", profileUrl: "", followerCount: "", followingCount: "", engagementRate: "", accountAgeDays: "" });
      toast({ title: "Account connected" });
    } catch (err: any) {
      toast({ title: "Failed to connect", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setConnecting(false); }
  };

  const field = (key: keyof typeof form) => ({
    value: form[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [key]: e.target.value })),
  });

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Profile</h2>
        <p className="text-muted-foreground">Manage your account details</p>
      </div>

      {/* Account info */}
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

      {/* Social accounts — promoters only */}
      {user?.role === "promoter" && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Social Accounts</CardTitle>
              <Button size="sm" onClick={() => setConnectOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Connect
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
                      <div className="flex items-center gap-2">
                        <span className="font-medium">@{a.username}</span>
                        <Badge variant="secondary" className="capitalize">{a.platform}</Badge>
                        {a.isVerified && <Badge variant="success">Verified</Badge>}
                      </div>
                      <div className="flex gap-4 text-xs text-muted-foreground">
                        <span>{formatNumber(a.followerCount)} followers</span>
                        <span>{a.engagementRate.toFixed(1)}% engagement</span>
                        <span>Score {a.influenceScore.toFixed(0)}</span>
                      </div>
                      <a
                        href={a.profileUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                      >
                        View profile <ExternalLink className="h-3 w-3" />
                      </a>
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

      {/* Stripe Connect status — promoters */}
      {user?.role === "promoter" && user.stripeConnectStatus && (
        <Card>
          <CardHeader><CardTitle>Payout Account</CardTitle></CardHeader>
          <CardContent>
            <div className="flex items-center gap-3">
              <Badge variant={user.stripeConnectStatus === "active" ? "success" : "warning"} className="capitalize">
                {user.stripeConnectStatus}
              </Badge>
              <span className="text-sm text-muted-foreground">Stripe Connect account</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Connect account dialog */}
      <Dialog open={connectOpen} onOpenChange={setConnectOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>Connect Social Account</DialogTitle></DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label>Platform</Label>
              <Select value={form.platform} onValueChange={(v) => setForm((f) => ({ ...f, platform: v }))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="instagram">Instagram</SelectItem>
                  <SelectItem value="twitter">Twitter / X</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>Username <span className="text-destructive">*</span></Label>
                <Input placeholder="yourhandle" {...field("username")} />
              </div>
              <div className="space-y-1">
                <Label>Account age (days) </Label>
                <Input type="number" placeholder="365" {...field("accountAgeDays")} />
              </div>
            </div>
            <div className="space-y-1">
              <Label>Profile URL <span className="text-destructive">*</span></Label>
              <Input placeholder="https://instagram.com/yourhandle" {...field("profileUrl")} />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>Followers <span className="text-destructive">*</span></Label>
                <Input type="number" placeholder="10000" {...field("followerCount")} />
              </div>
              <div className="space-y-1">
                <Label>Following</Label>
                <Input type="number" placeholder="500" {...field("followingCount")} />
              </div>
            </div>
            <div className="space-y-1">
              <Label>Engagement rate (%)</Label>
              <Input type="number" placeholder="3.5" step="0.1" {...field("engagementRate")} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConnectOpen(false)}>Cancel</Button>
            <Button onClick={handleConnect} disabled={connecting}>
              {connecting ? "Connecting…" : "Connect"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
