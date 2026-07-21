"use client";
import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { formatCurrency } from "@/lib/utils";
import { format } from "date-fns";
import { Check, X } from "lucide-react";
import { toast } from "@/components/ui/use-toast";

interface Withdrawal {
  id: string;
  userId: string;
  amount: number;
  status: string;
  payoutId?: string;
  requestedAt: string;
  processedAt?: string;
}

const statusVariant: Record<string, "warning" | "success" | "destructive" | "secondary"> = {
  pending: "warning", processing: "warning", completed: "success", failed: "destructive",
};

export default function AdminWithdrawalsPage() {
  const [withdrawals, setWithdrawals] = useState<Withdrawal[]>([]);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState("pending");
  const [acting, setActing] = useState<string | null>(null);
  const [rejectTarget, setRejectTarget] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState("");

  const load = () => {
    const params: Record<string, string> = {};
    if (status !== "all") params.status = status;
    setLoading(true);
    adminApi.listWithdrawals(params)
      .then((r) => setWithdrawals(r.data.data))
      .catch(() => toast({ title: "Failed to load withdrawals", variant: "destructive" }))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [status]);

  const handleApprove = async (id: string) => {
    setActing(id);
    try {
      await adminApi.approveWithdrawal(id);
      toast({ title: "Withdrawal approved", description: "Payout will be processed." });
      setWithdrawals((prev) => prev.filter((w) => w.id !== id));
    } catch (err: any) {
      toast({ title: "Approval failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setActing(null); }
  };

  const handleReject = async () => {
    if (!rejectTarget) return;
    if (rejectReason.trim().length < 5) {
      return toast({ title: "Reason must be at least 5 characters", variant: "destructive" });
    }
    setActing(rejectTarget);
    try {
      await adminApi.rejectWithdrawal(rejectTarget, rejectReason.trim());
      toast({ title: "Withdrawal rejected", description: "Balance refunded to promoter." });
      setWithdrawals((prev) => prev.filter((w) => w.id !== rejectTarget));
      setRejectTarget(null);
      setRejectReason("");
    } catch (err: any) {
      toast({ title: "Rejection failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setActing(null); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Withdrawals</h2>
          <p className="text-muted-foreground">Approve or reject payout requests</p>
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="processing">Processing</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 space-y-3">{[...Array(4)].map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
          ) : withdrawals.length === 0 ? (
            <p className="py-12 text-center text-muted-foreground">No withdrawals</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>User ID</TableHead>
                  <TableHead>Transfer ID</TableHead>
                  <TableHead>Requested</TableHead>
                  <TableHead>Processed</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {withdrawals.map((w) => (
                  <TableRow key={w.id}>
                    <TableCell><Badge variant={statusVariant[w.status] ?? "secondary"}>{w.status}</Badge></TableCell>
                    <TableCell className="font-medium">{formatCurrency(w.amount)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground font-mono">{w.userId}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{w.payoutId || "—"}</TableCell>
                    <TableCell className="text-muted-foreground text-sm">{format(new Date(w.requestedAt), "MMM d, yyyy")}</TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {w.processedAt && new Date(w.processedAt).getFullYear() > 1 ? format(new Date(w.processedAt), "MMM d, yyyy") : "—"}
                    </TableCell>
                    <TableCell>
                      {w.status === "pending" && (
                        <div className="flex gap-2">
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-green-400 hover:text-green-300"
                            onClick={() => handleApprove(w.id)} disabled={acting === w.id}>
                            <Check className="h-4 w-4" />
                          </Button>
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-destructive hover:text-red-400"
                            onClick={() => { setRejectTarget(w.id); setRejectReason(""); }} disabled={acting === w.id}>
                            <X className="h-4 w-4" />
                          </Button>
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={!!rejectTarget} onOpenChange={(o) => { if (!o) setRejectTarget(null); }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>Reject withdrawal</DialogTitle></DialogHeader>
          <div className="space-y-2">
            <Label>Reason (min 5 characters)</Label>
            <Textarea
              placeholder="Why is this withdrawal being rejected?"
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectTarget(null)} disabled={acting === rejectTarget}>Cancel</Button>
            <Button variant="destructive" onClick={handleReject} disabled={acting === rejectTarget}>
              {acting === rejectTarget ? "Rejecting…" : "Reject & Refund"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
