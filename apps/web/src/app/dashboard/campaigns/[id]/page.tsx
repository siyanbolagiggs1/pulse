"use client";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { campaignsApi, submissionsApi } from "@/lib/api";
import type { Campaign, CampaignSubmission } from "@/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { formatCurrency } from "@/lib/utils";
import { ArrowLeft, Check, X, Pencil } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

const statusVariant: Record<string, "success" | "warning" | "destructive" | "secondary"> = {
  approved: "success", pending: "warning", rejected: "destructive",
};

export default function CampaignDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [submissions, setSubmissions] = useState<CampaignSubmission[]>([]);
  const [loading, setLoading] = useState(true);
  const [rejectTarget, setRejectTarget] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [processing, setProcessing] = useState(false);

  useEffect(() => {
    Promise.all([
      campaignsApi.get(id),
      submissionsApi.list({ campaignId: id, limit: 50 }),
    ])
      .then(([cr, sr]) => { setCampaign(cr.data.data); setSubmissions(sr.data.data); })
      .catch(() => toast({ title: "Error", description: "Failed to load", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [id]);

  const approve = async (subId: string) => {
    setProcessing(true);
    try {
      await submissionsApi.approve(subId);
      setSubmissions((prev) => prev.map((s) => s.id === subId ? { ...s, status: "approved" } : s));
      toast({ title: "Approved" });
    } catch (err: any) {
      toast({ title: "Error", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setProcessing(false); }
  };

  const reject = async () => {
    if (!rejectTarget || !rejectReason.trim()) return;
    setProcessing(true);
    try {
      await submissionsApi.reject(rejectTarget, rejectReason);
      setSubmissions((prev) => prev.map((s) => s.id === rejectTarget ? { ...s, status: "rejected" } : s));
      setRejectTarget(null); setRejectReason("");
      toast({ title: "Rejected" });
    } catch (err: any) {
      toast({ title: "Error", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setProcessing(false); }
  };

  if (loading) return <Skeleton className="h-96 w-full" />;
  if (!campaign) return <p>Campaign not found.</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild><Link href="/dashboard/campaigns"><ArrowLeft className="h-4 w-4" /></Link></Button>
          <div>
            <h2 className="text-2xl font-bold">{campaign.title}</h2>
            <p className="text-muted-foreground capitalize">{campaign.platform} · {campaign.status}</p>
          </div>
        </div>
        {campaign.status !== "completed" && campaign.status !== "cancelled" && (
          <Button variant="outline" size="sm" asChild>
            <Link href={`/dashboard/campaigns/${id}/edit`}><Pencil className="mr-2 h-4 w-4" />Edit</Link>
          </Button>
        )}
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        {[
          ["Budget Remaining", formatCurrency(campaign.remainingBudget)],
          ["Participants", `${campaign.currentParticipants} / ${campaign.maxParticipants}`],
          ["Base Payout", formatCurrency(campaign.baseRepostRate)],
        ].map(([label, value]) => (
          <Card key={label}><CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">{label}</p>
            <p className="text-2xl font-bold">{value}</p>
          </CardContent></Card>
        ))}
      </div>

      <Card>
        <CardHeader><CardTitle>Submissions ({submissions.length})</CardTitle></CardHeader>
        <CardContent>
          {submissions.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">No submissions yet</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Payout</TableHead>
                  <TableHead>Submitted</TableHead>
                  <TableHead>Proof</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {submissions.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell><Badge variant={statusVariant[s.status] ?? "secondary"}>{s.status}</Badge></TableCell>
                    <TableCell>{formatCurrency(s.finalAmount)}</TableCell>
                    <TableCell className="text-muted-foreground">{format(new Date(s.submittedAt), "MMM d")}</TableCell>
                    <TableCell>
                      <a href={s.repostUrl} target="_blank" rel="noreferrer" className="text-primary hover:underline text-sm">View post</a>
                    </TableCell>
                    <TableCell>
                      {s.status === "pending" && (
                        <div className="flex gap-1">
                          <Button size="icon" variant="ghost" className="h-7 w-7 text-green-500" onClick={() => approve(s.id)} disabled={processing}>
                            <Check className="h-4 w-4" />
                          </Button>
                          <Button size="icon" variant="ghost" className="h-7 w-7 text-destructive" onClick={() => setRejectTarget(s.id)} disabled={processing}>
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

      <Dialog open={!!rejectTarget} onOpenChange={(o) => { if (!o) { setRejectTarget(null); setRejectReason(""); } }}>
        <DialogContent>
          <DialogHeader><DialogTitle>Reject Submission</DialogTitle></DialogHeader>
          <div className="space-y-2">
            <Label>Reason (required)</Label>
            <Textarea placeholder="Explain why this submission is being rejected…" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={reject} disabled={!rejectReason.trim() || processing}>Reject</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
