"use client";
import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

interface FraudFlag {
  id: string;
  userId: string;
  reason: string;
  details: string;
  resolved: boolean;
  createdAt: string;
}

const reasonLabel: Record<string, string> = {
  low_follower_ratio: "Low Follower Ratio",
  abnormal_engagement: "Abnormal Engagement",
  duplicate_submission: "Duplicate Submission",
  rate_limit_exceeded: "Rate Limit Exceeded",
  admin_suspension: "Admin Suspension",
};

export default function AdminFraudFlagsPage() {
  const [flags, setFlags] = useState<FraudFlag[]>([]);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState<string | null>(null);

  const load = () => {
    setLoading(true);
    adminApi.listFraudFlags()
      .then((r) => setFlags(r.data.data))
      .catch(() => toast({ title: "Failed to load fraud flags", variant: "destructive" }))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const handleResolve = async (id: string) => {
    setActing(id);
    try {
      await adminApi.resolveFraudFlag(id);
      toast({ title: "Flag resolved" });
      setFlags((prev) => prev.map((f) => f.id === id ? { ...f, resolved: true } : f));
    } catch {
      toast({ title: "Failed to resolve", variant: "destructive" });
    } finally { setActing(null); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Fraud Flags</h2>
        <p className="text-muted-foreground">Review and resolve flagged activity</p>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 space-y-3">{[...Array(4)].map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
          ) : flags.length === 0 ? (
            <p className="py-12 text-center text-muted-foreground">No fraud flags</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Reason</TableHead>
                  <TableHead>Details</TableHead>
                  <TableHead>User ID</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Flagged</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {flags.map((f) => (
                  <TableRow key={f.id}>
                    <TableCell>
                      <Badge variant={f.resolved ? "secondary" : "destructive"}>
                        {reasonLabel[f.reason] ?? f.reason}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground max-w-xs truncate">{f.details || "—"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground font-mono">{f.userId}</TableCell>
                    <TableCell>
                      <Badge variant={f.resolved ? "success" : "warning"}>
                        {f.resolved ? "Resolved" : "Open"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">{format(new Date(f.createdAt), "MMM d, yyyy")}</TableCell>
                    <TableCell>
                      {!f.resolved && (
                        <Button size="sm" variant="outline" onClick={() => handleResolve(f.id)} disabled={acting === f.id}>
                          Resolve
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
