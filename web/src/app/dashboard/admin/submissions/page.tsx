"use client";
import { useEffect, useState } from "react";
import { submissionsApi } from "@/lib/api";
import type { CampaignSubmission } from "@/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { formatCurrency, apiFileUrl } from "@/lib/utils";
import { format } from "date-fns";
import { Check, X } from "lucide-react";
import { toast } from "@/components/ui/use-toast";

export default function AdminSubmissionsPage() {
  const [submissions, setSubmissions] = useState<CampaignSubmission[]>([]);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState("pending");
  const [acting, setActing] = useState<string | null>(null);
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState("");

  const load = () => {
    const params: Record<string, string> = {};
    if (status !== "all") params.status = status;
    setLoading(true);
    submissionsApi.list(params)
      .then((r) => setSubmissions(r.data.data))
      .catch(() => toast({ title: "Failed to load", variant: "destructive" }))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [status]);

  const handleApprove = async (id: string) => {
    setActing(id);
    try {
      await submissionsApi.approve(id);
      toast({ title: "Submission approved" });
      setSubmissions((prev) => prev.filter((s) => s.id !== id));
    } catch {
      toast({ title: "Failed to approve", variant: "destructive" });
    } finally { setActing(null); }
  };

  const handleReject = async () => {
    if (!rejectId) return;
    setActing(rejectId);
    try {
      await submissionsApi.reject(rejectId, reason);
      toast({ title: "Submission rejected" });
      setSubmissions((prev) => prev.filter((s) => s.id !== rejectId));
      setRejectId(null);
      setReason("");
    } catch {
      toast({ title: "Failed to reject", variant: "destructive" });
    } finally { setActing(null); }
  };

  const statusVariant: Record<string, "success" | "warning" | "destructive" | "secondary"> = {
    approved: "success", pending: "warning", rejected: "destructive",
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Submission Review</h2>
          <p className="text-muted-foreground">Approve or reject promoter submissions</p>
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="approved">Approved</SelectItem>
            <SelectItem value="rejected">Rejected</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 space-y-3">{[...Array(4)].map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
          ) : submissions.length === 0 ? (
            <p className="py-12 text-center text-muted-foreground">No submissions</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Payout</TableHead>
                  <TableHead>Promoter Earning</TableHead>
                  <TableHead>Submitted</TableHead>
                  <TableHead>Proof</TableHead>
                  <TableHead>Screenshot</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {submissions.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell><Badge variant={statusVariant[s.status] ?? "secondary"}>{s.status}</Badge></TableCell>
                    <TableCell>{formatCurrency(s.finalAmount)}</TableCell>
                    <TableCell className="text-green-400">{formatCurrency(s.promoterEarning)}</TableCell>
                    <TableCell className="text-muted-foreground text-sm">{format(new Date(s.submittedAt), "MMM d, yyyy")}</TableCell>
                    <TableCell>
                      <a href={s.repostUrl} target="_blank" rel="noreferrer" className="text-primary hover:underline text-sm">Post</a>
                    </TableCell>
                    <TableCell>
                      {s.screenshotUrl && (
                        <a href={apiFileUrl(s.screenshotUrl)} target="_blank" rel="noreferrer" className="text-primary hover:underline text-sm">Screenshot</a>
                      )}
                    </TableCell>
                    <TableCell>
                      {s.status === "pending" && (
                        <div className="flex gap-2">
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-green-400 hover:text-green-300"
                            onClick={() => handleApprove(s.id)} disabled={acting === s.id}>
                            <Check className="h-4 w-4" />
                          </Button>
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-destructive hover:text-red-400"
                            onClick={() => { setRejectId(s.id); setReason(""); }} disabled={acting === s.id}>
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

      <Dialog open={!!rejectId} onOpenChange={(o) => { if (!o) { setRejectId(null); setReason(""); } }}>
        <DialogContent>
          <DialogHeader><DialogTitle>Reject Submission</DialogTitle></DialogHeader>
          <Textarea
            placeholder="Reason for rejection (optional)…"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={3}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => { setRejectId(null); setReason(""); }}>Cancel</Button>
            <Button variant="destructive" onClick={handleReject} disabled={!!acting}>Reject</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
