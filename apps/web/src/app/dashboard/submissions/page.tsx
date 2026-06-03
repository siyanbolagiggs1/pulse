"use client";
import { useEffect, useState } from "react";
import { submissionsApi } from "@/lib/api";
import type { CampaignSubmission } from "@/types";
import { useAuthStore } from "@/store/auth";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Pagination } from "@/components/ui/pagination";
import { formatCurrency, apiFileUrl } from "@/lib/utils";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

const statusVariant: Record<string, "success" | "warning" | "destructive" | "secondary"> = {
  approved: "success", pending: "warning", rejected: "destructive",
};

const PAGE_SIZE = 20;

export default function SubmissionsPage() {
  const user = useAuthStore((s) => s.user);
  const [submissions, setSubmissions] = useState<CampaignSubmission[]>([]);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState("all");
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);

  useEffect(() => {
    setPage(1);
  }, [status]);

  useEffect(() => {
    const params: Record<string, string | number> = { page, limit: PAGE_SIZE };
    if (status !== "all") params.status = status;
    setLoading(true);
    submissionsApi.list(params)
      .then((r) => {
        setSubmissions(r.data.data);
        setPages(r.data.meta?.pages ?? 1);
      })
      .catch(() => toast({ title: "Failed to load submissions", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [status, page]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Submissions</h2>
          <p className="text-muted-foreground">
            {user?.role === "promoter" ? "Track your submission status" : "Review incoming submissions"}
          </p>
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
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Status</TableHead>
                    <TableHead>Payout</TableHead>
                    <TableHead>Promoter Earning</TableHead>
                    <TableHead>Submitted</TableHead>
                    <TableHead>Proof</TableHead>
                    <TableHead>Screenshot</TableHead>
                    {user?.role === "promoter" && <TableHead>Release Date</TableHead>}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {submissions.map((s) => (
                    <TableRow key={s.id}>
                      <TableCell><Badge variant={statusVariant[s.status] ?? "secondary"}>{s.status}</Badge></TableCell>
                      <TableCell>{formatCurrency(s.finalAmount)}</TableCell>
                      <TableCell className="text-green-400">{formatCurrency(s.promoterEarning)}</TableCell>
                      <TableCell className="text-muted-foreground">{format(new Date(s.submittedAt), "MMM d, yyyy")}</TableCell>
                      <TableCell>
                        <a href={s.repostUrl} target="_blank" rel="noreferrer" className="text-primary hover:underline text-sm">View</a>
                      </TableCell>
                      <TableCell>
                        {s.screenshotUrl && (
                          <a href={apiFileUrl(s.screenshotUrl)} target="_blank" rel="noreferrer" className="text-primary hover:underline text-sm">Screenshot</a>
                        )}
                      </TableCell>
                      {user?.role === "promoter" && (
                        <TableCell className="text-muted-foreground text-sm">
                          {s.payoutReleasedAt ? format(new Date(s.payoutReleasedAt), "MMM d") : "—"}
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="px-6 pb-4">
                <Pagination page={page} pages={pages} onChange={setPage} />
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
