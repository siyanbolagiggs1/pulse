"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
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
import { Trash2, Plus, ExternalLink, Loader2, Landmark, Pencil, AlertTriangle } from "lucide-react";
import { toast } from "@/components/ui/use-toast";

type Platform = "instagram" | "twitter" | "tiktok";

const maskAccountNumber = (n: string) => (n.length <= 4 ? n : `•••• ${n.slice(-4)}`);

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
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const updateUser = useAuthStore((s) => s.updateUser);
  const clearAuth = useAuthStore((s) => s.clearAuth);

  const [name, setName] = useState(user?.name ?? "");
  const [savingProfile, setSavingProfile] = useState(false);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deletingAccount, setDeletingAccount] = useState(false);

  const [banks, setBanks] = useState<{ code: string; name: string }[]>([]);
  const [bankOpen, setBankOpen] = useState(false);
  const [savingBank, setSavingBank] = useState(false);
  const [bankSearch, setBankSearch] = useState("");
  const [bankCode, setBankCode] = useState("");
  const [accountNumber, setAccountNumber] = useState("");

  const [accounts, setAccounts] = useState<SocialAccount[]>([]);
  const [loadingAccounts, setLoadingAccounts] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const [connectOpen, setConnectOpen] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [platform, setPlatform] = useState<Platform>("twitter");
  const [username, setUsername] = useState("");

  const profileUrlPrefixes: Record<Platform, string> = {
    instagram: "https://instagram.com/",
    twitter: "https://twitter.com/",
    tiktok: "https://tiktok.com/@",
  };

  const buildProfileUrl = (p: Platform, u: string) =>
    `${profileUrlPrefixes[p]}${u.replace(/^@/, "")}`;

  useEffect(() => {
    if (!user || user.role === "admin") return;
    setLoadingAccounts(true);
    usersApi.getMe()
      .then((r) => setAccounts(r.data.data.socialAccounts ?? []))
      .catch(() => toast({ title: "Failed to load accounts", variant: "destructive" }))
      .finally(() => setLoadingAccounts(false));
  }, [user?.id]);

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

  const handleRemoveSocialAccount = async (id: string) => {
    setDeletingId(id);
    try {
      await usersApi.deleteSocialAccount(id);
      setAccounts((prev) => prev.filter((a) => a.id !== id));
      toast({ title: "Account removed" });
    } catch {
      toast({ title: "Failed to remove account", variant: "destructive" });
    } finally { setDeletingId(null); }
  };

  const handleDeleteMyAccount = async () => {
    setDeletingAccount(true);
    try {
      await usersApi.deleteAccount();
      clearAuth();
      toast({ title: "Account deleted" });
      router.push("/signin");
    } catch (err: any) {
      toast({
        title: "Could not delete account",
        description: err?.response?.data?.message,
        variant: "destructive",
      });
      setDeletingAccount(false);
      setDeleteOpen(false);
    }
  };

  const handleConnect = async () => {
    const clean = username.trim().replace(/^@/, "");
    if (!clean) return toast({ title: "Username is required", variant: "destructive" });

    setConnecting(true);
    try {
      const res = await usersApi.connectSocialAccount({
        platform,
        username: clean,
        profileUrl: buildProfileUrl(platform, clean),
      });
      // A reconnect reuses the same account id (see backend), so upsert
      // rather than append to avoid showing a duplicate row.
      setAccounts((prev) => {
        const idx = prev.findIndex((a) => a.id === res.data.data.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = res.data.data;
          return next;
        }
        return [...prev, res.data.data];
      });
      setConnectOpen(false);
      setUsername("");
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
    setUsername("");
  };

  const openBankDialog = () => {
    setBankSearch("");
    setBankCode("");
    setAccountNumber("");
    setBankOpen(true);
    if (banks.length === 0) {
      usersApi.listBanks()
        .then((r) => setBanks(r.data.data ?? []))
        .catch(() => toast({ title: "Failed to load bank list", variant: "destructive" }));
    }
  };

  const handleSaveBankAccount = async () => {
    if (!bankCode) return toast({ title: "Select a bank", variant: "destructive" });
    if (!accountNumber.trim()) return toast({ title: "Account number is required", variant: "destructive" });

    setSavingBank(true);
    try {
      const res = await usersApi.setBankAccount({ bankCode, accountNumber: accountNumber.trim() });
      if (user) updateUser({ ...user, bankAccount: res.data.data });
      setBankOpen(false);
      toast({ title: "Bank account saved", description: `Verified: ${res.data.data.accountName}` });
    } catch (err: any) {
      toast({ title: "Could not verify account", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setSavingBank(false); }
  };

  const filteredBanks = bankSearch.trim()
    ? banks.filter((b) => b.name.toLowerCase().includes(bankSearch.trim().toLowerCase()))
    : banks;

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
            {user && user.role !== "admin" && (
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

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Payout Bank Account</CardTitle>
            <Button size="sm" variant={user?.bankAccount ? "outline" : "default"} onClick={openBankDialog}>
              {user?.bankAccount ? <><Pencil className="mr-2 h-4 w-4" />Update</> : <><Plus className="mr-2 h-4 w-4" />Add</>}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {user?.bankAccount ? (
            <div className="flex items-center gap-3 rounded-lg border border-border p-3">
              <Landmark className="h-8 w-8 text-muted-foreground shrink-0" />
              <div>
                <p className="font-medium">{user.bankAccount.bankName}</p>
                <p className="text-sm text-muted-foreground">{maskAccountNumber(user.bankAccount.accountNumber)} · {user.bankAccount.accountName}</p>
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No bank account on file — required before you can request a withdrawal.
            </p>
          )}
        </CardContent>
      </Card>

      {user && user.role !== "admin" && (
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
                          <span>Tier {a.tier}</span>
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
                      onClick={() => handleRemoveSocialAccount(a.id)}
                      disabled={deletingId === a.id}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
            <p className="mt-4 text-xs text-muted-foreground">
              Tier 1: 100–500 followers · Tier 2: 501–1,000 · Tier 3: 1,001–1,500 · Tier 4: 1,501–2,000 · and so on in 500-follower increments.
              Gained enough followers to reach the next tier? Disconnect and reconnect your account so we can re-verify it —
              you can request re-verification once every 30 days per account.
            </p>
          </CardContent>
        </Card>
      )}

      <Card className="border-destructive/40">
        <CardHeader><CardTitle className="text-destructive">Danger Zone</CardTitle></CardHeader>
        <CardContent className="flex items-center justify-between gap-4">
          <p className="text-sm text-muted-foreground">
            Permanently delete your account and everything tied to it. Your wallet balance must be zero first.
          </p>
          <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />Delete Account
          </Button>
        </CardContent>
      </Card>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>Delete your account?</DialogTitle></DialogHeader>
          <div className="flex gap-3 rounded-md bg-destructive/10 border border-destructive/30 p-3 text-sm">
            <AlertTriangle className="h-5 w-5 shrink-0 text-destructive" />
            <p>
              This permanently deletes your profile, wallet, transactions, campaigns/submissions, messages and
              connected accounts. This cannot be undone. If your wallet balance isn&apos;t zero, this will fail.
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)} disabled={deletingAccount}>Cancel</Button>
            <Button variant="destructive" onClick={handleDeleteMyAccount} disabled={deletingAccount}>
              {deletingAccount ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Deleting…</> : "Yes, delete my account"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={connectOpen} onOpenChange={(o) => { setConnectOpen(o); if (!o) resetDialog(); }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>Connect Social Account</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1">
              <Label>Platform</Label>
              <Select value={platform} onValueChange={(v: Platform) => { setPlatform(v); setUsername(""); }}>
                <SelectTrigger className="bg-background"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="twitter">Twitter / X</SelectItem>
                  <SelectItem value="instagram">Instagram</SelectItem>
                  <SelectItem value="tiktok">TikTok</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1">
              <Label>Username</Label>
              <div className="flex items-center rounded-md border border-input bg-background focus-within:ring-2 focus-within:ring-ring">
                <span className="pl-3 text-sm text-muted-foreground">@</span>
                <input
                  className="flex-1 bg-transparent px-2 py-2 text-sm outline-none placeholder:text-muted-foreground"
                  placeholder="yourhandle"
                  value={username}
                  onChange={(e) => setUsername(e.target.value.replace(/^@/, ""))}
                />
              </div>
              {username.trim() && (
                <p className="text-xs text-muted-foreground">
                  Profile: {buildProfileUrl(platform, username.trim())}
                </p>
              )}
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

      <Dialog open={bankOpen} onOpenChange={setBankOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>Payout Bank Account</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1">
              <Label>Bank</Label>
              <Input
                placeholder="Search banks…"
                value={bankSearch}
                onChange={(e) => setBankSearch(e.target.value)}
              />
              <Select value={bankCode} onValueChange={setBankCode}>
                <SelectTrigger className="bg-background"><SelectValue placeholder="Select bank" /></SelectTrigger>
                <SelectContent>
                  {filteredBanks.length === 0 ? (
                    <div className="px-2 py-1.5 text-sm text-muted-foreground">
                      {banks.length === 0 ? "Loading…" : "No matches"}
                    </div>
                  ) : filteredBanks.map((b) => (
                    <SelectItem key={b.code} value={b.code}>{b.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1">
              <Label>Account Number</Label>
              <Input
                placeholder="0123456789"
                value={accountNumber}
                onChange={(e) => setAccountNumber(e.target.value.replace(/\D/g, ""))}
              />
            </div>

            <p className="text-xs text-muted-foreground">
              We verify this account with your bank before saving it — the account holder's name will be shown to confirm it's really yours.
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBankOpen(false)}>Cancel</Button>
            <Button onClick={handleSaveBankAccount} disabled={savingBank}>
              {savingBank ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Verifying…</> : "Verify & Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
