"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { walletApi } from "@/lib/api";
import type { Wallet, Transaction, Withdrawal } from "@/types";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/lib/utils";
import { ArrowDownToLine, ArrowUpFromLine, ExternalLink } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

const maskAccountNumber = (n: string) => (n.length <= 4 ? n : `•••• ${n.slice(-4)}`);

export default function WalletPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const [wallet, setWallet] = useState<Wallet | null>(null);
  const [txs, setTxs] = useState<Transaction[]>([]);
  const [withdrawals, setWithdrawals] = useState<Withdrawal[]>([]);
  const [loading, setLoading] = useState(true);
  const [topupOpen, setTopupOpen] = useState(false);
  const [withdrawOpen, setWithdrawOpen] = useState(false);
  const [topupAmount, setTopupAmount] = useState("");
  const [withdrawAmount, setWithdrawAmount] = useState("");
  const [processing, setProcessing] = useState(false);

  const load = () => {
    const p: Promise<any>[] = [walletApi.get()];
    if (user?.role === "promoter") p.push(walletApi.getWithdrawals());
    Promise.all(p)
      .then(([wr, wdr]) => {
        setWallet(wr.data.data);
        setTxs(wr.data.data.recentTransactions ?? []);
        if (wdr) setWithdrawals(wdr.data.data ?? []);
      })
      .catch(() => toast({ title: "Error loading wallet", variant: "destructive" }))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const handleTopup = async () => {
    const amount = parseFloat(topupAmount);
    if (!amount || amount < 0.1) return toast({ title: "Minimum top-up is ₦0.10", variant: "destructive" });
    setProcessing(true);
    try {
      const res = await walletApi.createTopup(amount);
      if (res.data.data.authorizationUrl) {
        window.location.href = res.data.data.authorizationUrl;
      } else {
        // Dev mode — wallet credited directly.
        toast({ title: "Wallet topped up", description: `${amount.toFixed(2)} added to your balance.` });
        setTopupOpen(false);
        load();
      }
    } catch (err: any) {
      toast({ title: "Failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setProcessing(false); }
  };

  const handleWithdraw = async () => {
    const amount = parseFloat(withdrawAmount);
    if (!amount || amount < 0.1) return toast({ title: "Minimum withdrawal is ₦0.10", variant: "destructive" });
    setProcessing(true);
    try {
      await walletApi.withdraw(amount);
      toast({ title: "Withdrawal requested", description: "Admin will review and process it." });
      setWithdrawOpen(false);
      load();
    } catch (err: any) {
      toast({ title: "Failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setProcessing(false); }
  };

  if (loading) return <div className="space-y-4"><Skeleton className="h-32" /><Skeleton className="h-64" /></div>;

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Wallet</h2>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card><CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">Available</p>
          <p className="text-3xl font-bold text-green-400">{formatCurrency(wallet?.availableBalance ?? 0)}</p>
        </CardContent></Card>
        <Card><CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">Pending</p>
          <p className="text-3xl font-bold text-yellow-400">{formatCurrency(wallet?.pendingBalance ?? 0)}</p>
        </CardContent></Card>
        <Card><CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">{user?.role === "promoter" ? "Total Earned" : "Total Spent"}</p>
          <p className="text-3xl font-bold">{formatCurrency(user?.role === "promoter" ? (wallet?.totalEarned ?? 0) : (wallet?.totalSpent ?? 0))}</p>
        </CardContent></Card>
      </div>

      <div className="flex flex-wrap gap-3">
        {user?.role === "business" && (
          <Button onClick={() => setTopupOpen(true)}><ArrowUpFromLine className="mr-2 h-4 w-4" />Top Up</Button>
        )}
        <Button
          onClick={() => {
            if (!user?.bankAccount) {
              toast({
                title: "Add a payout bank account first",
                description: "Go to Profile to add one before requesting a withdrawal.",
              });
              router.push("/dashboard/profile");
              return;
            }
            setWithdrawOpen(true);
          }}
          disabled={!wallet || wallet.availableBalance < 0.1}
        >
          <ArrowDownToLine className="mr-2 h-4 w-4" />Withdraw
        </Button>
      </div>

      {user?.role === "promoter" && withdrawals.length > 0 && (
        <Card>
          <CardHeader><CardTitle>Withdrawals</CardTitle></CardHeader>
          <CardContent className="space-y-2">
            {withdrawals.map((w) => (
              <div key={w.id} className="flex items-center justify-between border-b border-border py-2 last:border-0">
                <div>
                  <p className="text-sm font-medium">{formatCurrency(w.amount)}</p>
                  <p className="text-xs text-muted-foreground">{format(new Date(w.requestedAt), "MMM d, yyyy")}</p>
                </div>
                <Badge variant={w.status === "completed" ? "success" : w.status === "failed" ? "destructive" : "warning"}>
                  {w.status}
                </Badge>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><CardTitle>Recent Transactions</CardTitle></CardHeader>
        <CardContent className="space-y-2">
          {txs.length === 0 ? (
            <p className="text-center py-8 text-muted-foreground">No transactions yet</p>
          ) : txs.map((tx) => (
            <div key={tx.id} className="flex items-center justify-between border-b border-border py-2 last:border-0">
              <div>
                <p className="text-sm">{tx.description}</p>
                <p className="text-xs text-muted-foreground">{format(new Date(tx.createdAt), "MMM d, yyyy · HH:mm")}</p>
              </div>
              <p className={`text-sm font-medium ${tx.amount >= 0 ? "text-green-400" : "text-red-400"}`}>
                {tx.amount >= 0 ? "+" : ""}{formatCurrency(tx.amount)}
              </p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Dialog open={topupOpen} onOpenChange={setTopupOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>Top Up Wallet</DialogTitle></DialogHeader>
          <div className="space-y-2">
            <Label>Amount (NGN)</Label>
            <Input type="number" min="0.1" step="0.01" placeholder="1.00" value={topupAmount} onChange={(e) => setTopupAmount(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setTopupOpen(false)}>Cancel</Button>
            <Button onClick={handleTopup} disabled={processing}>Pay with Paystack</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={withdrawOpen} onOpenChange={setWithdrawOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>Request Withdrawal</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground">Available: {formatCurrency(wallet?.availableBalance ?? 0)}</p>
          {user?.bankAccount && (
            <p className="text-sm text-muted-foreground">
              To: {user.bankAccount.bankName} · {maskAccountNumber(user.bankAccount.accountNumber)} ({user.bankAccount.accountName})
            </p>
          )}
          <div className="space-y-2">
            <Label>Amount (NGN, min ₦0.10)</Label>
            <Input type="number" min="0.1" step="0.01" max={wallet?.availableBalance} placeholder="50" value={withdrawAmount} onChange={(e) => setWithdrawAmount(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setWithdrawOpen(false)}>Cancel</Button>
            <Button onClick={handleWithdraw} disabled={processing}>Request</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
