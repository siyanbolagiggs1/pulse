"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { campaignsApi } from "@/lib/api";
import type { Campaign } from "@/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Pagination } from "@/components/ui/pagination";
import { formatCurrency } from "@/lib/utils";
import { Plus, Users, DollarSign, Pencil, Trash2 } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

const statusVariant: Record<string, "success" | "warning" | "secondary" | "destructive"> = {
  active: "success", draft: "secondary", paused: "warning",
  completed: "secondary", cancelled: "destructive",
};

const PAGE_SIZE = 12;

export default function CampaignsPage() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);
  const [deleteTarget, setDeleteTarget] = useState<Campaign | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = (p: number) => {
    setLoading(true);
    campaignsApi.getMy({ page: p, limit: PAGE_SIZE })
      .then((r) => {
        setCampaigns(r.data.data);
        setPages(r.data.meta?.pages ?? 1);
      })
      .catch(() => toast({ title: "Failed to load campaigns", variant: "destructive" }))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(page); }, [page]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await campaignsApi.delete(deleteTarget.id);
      toast({ title: "Campaign deleted", description: "Remaining budget refunded to wallet." });
      setDeleteTarget(null);
      load(page);
    } catch (err: any) {
      toast({ title: "Failed to delete", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setDeleting(false); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Campaigns</h2>
          <p className="text-muted-foreground">Manage your repost campaigns</p>
        </div>
        <Button asChild>
          <Link href="/dashboard/campaigns/new"><Plus className="mr-2 h-4 w-4" />New Campaign</Link>
        </Button>
      </div>

      {loading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(6)].map((_, i) => <Skeleton key={i} className="h-52" />)}
        </div>
      ) : campaigns.length === 0 ? (
        <Card><CardContent className="py-12 text-center text-muted-foreground">
          No campaigns yet. <Link href="/dashboard/campaigns/new" className="text-primary hover:underline">Create your first</Link>.
        </CardContent></Card>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {campaigns.map((c) => (
              <Card key={c.id} className="flex flex-col hover:border-primary/50 transition-colors">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-base line-clamp-1">{c.title}</CardTitle>
                    <Badge variant={statusVariant[c.status] ?? "secondary"} className="capitalize shrink-0">{c.status}</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground capitalize">{c.platform}</p>
                </CardHeader>
                <CardContent className="flex-1 space-y-3">
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div className="flex items-center gap-1 text-muted-foreground">
                      <DollarSign className="h-3 w-3" />
                      <span>{formatCurrency(c.remainingBudget)} left</span>
                    </div>
                    <div className="flex items-center gap-1 text-muted-foreground">
                      <Users className="h-3 w-3" />
                      <span>{c.currentParticipants}/{c.maxParticipants}</span>
                    </div>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Ends {format(new Date(c.endDate), "MMM d, yyyy")}
                  </p>
                  <div className="flex gap-2 pt-1">
                    <Button variant="outline" size="sm" className="flex-1" asChild>
                      <Link href={`/dashboard/campaigns/${c.id}`}>View</Link>
                    </Button>
                    {c.status !== "completed" && c.status !== "cancelled" && (
                      <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" asChild>
                        <Link href={`/dashboard/campaigns/${c.id}/edit`}><Pencil className="h-4 w-4" /></Link>
                      </Button>
                    )}
                    <Button
                      variant="ghost" size="icon"
                      className="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
                      onClick={() => setDeleteTarget(c)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
          <Pagination page={page} pages={pages} onChange={setPage} />
        </>
      )}

      <Dialog open={!!deleteTarget} onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}>
        <DialogContent>
          <DialogHeader><DialogTitle>Delete "{deleteTarget?.title}"?</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground">
            The remaining budget ({formatCurrency(deleteTarget?.remainingBudget ?? 0)}) will be refunded to your wallet. This cannot be undone.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "Deleting…" : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
