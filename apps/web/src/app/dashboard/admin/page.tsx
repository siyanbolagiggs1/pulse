"use client";
import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/lib/utils";
import { Users, Megaphone, FileCheck, DollarSign } from "lucide-react";
import { toast } from "@/components/ui/use-toast";

interface Stats {
  users: { total: number; businesses: number; promoters: number; admins: number; suspended: number };
  campaigns: { total: number; active: number; completed: number; totalBudget: number };
  submissions: { total: number; pending: number; approved: number; rejected: number };
  financials: { totalTopups: number; totalPayouts: number; totalCommission: number; pendingWithdrawals: number };
}

export default function AdminDashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    adminApi.getStats()
      .then((r) => setStats(r.data.data))
      .catch(() => toast({ title: "Failed to load stats", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Platform Overview</h2>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[...Array(8)].map((_, i) => <Skeleton key={i} className="h-28" />)}
      </div>
    </div>
  );

  if (!stats) return null;

  const cards = [
    { label: "Total Users", value: stats.users.total, sub: `${stats.users.suspended} suspended`, icon: Users, color: "text-blue-400" },
    { label: "Businesses", value: stats.users.businesses, sub: "registered", icon: Users, color: "text-purple-400" },
    { label: "Promoters", value: stats.users.promoters, sub: "registered", icon: Users, color: "text-pink-400" },
    { label: "Active Campaigns", value: stats.campaigns.active, sub: `${stats.campaigns.total} total`, icon: Megaphone, color: "text-yellow-400" },
    { label: "Pending Submissions", value: stats.submissions.pending, sub: `${stats.submissions.total} total`, icon: FileCheck, color: "text-orange-400" },
    { label: "Total Revenue", value: formatCurrency(stats.financials.totalCommission), sub: "platform commission", icon: DollarSign, color: "text-green-400" },
    { label: "Total Payouts", value: formatCurrency(stats.financials.totalPayouts), sub: "to promoters", icon: DollarSign, color: "text-teal-400" },
    { label: "Pending Withdrawals", value: formatCurrency(stats.financials.pendingWithdrawals), sub: "awaiting approval", icon: DollarSign, color: "text-red-400" },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Platform Overview</h2>
        <p className="text-muted-foreground">Real-time platform metrics</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {cards.map(({ label, value, sub, icon: Icon, color }) => (
          <Card key={label}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
              <Icon className={`h-4 w-4 ${color}`} />
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">{value}</p>
              <p className="text-xs text-muted-foreground mt-1">{sub}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Submission Breakdown</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {[
              { label: "Approved", value: stats.submissions.approved, color: "bg-green-400" },
              { label: "Pending", value: stats.submissions.pending, color: "bg-yellow-400" },
              { label: "Rejected", value: stats.submissions.rejected, color: "bg-red-400" },
            ].map(({ label, value, color }) => {
              const pct = stats.submissions.total ? Math.round((value / stats.submissions.total) * 100) : 0;
              return (
                <div key={label} className="space-y-1">
                  <div className="flex justify-between text-sm">
                    <span>{label}</span>
                    <span className="font-medium">{value} ({pct}%)</span>
                  </div>
                  <div className="h-2 rounded-full bg-muted overflow-hidden">
                    <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
                  </div>
                </div>
              );
            })}
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Campaign Breakdown</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {[
              { label: "Active", value: stats.campaigns.active, color: "bg-green-400" },
              { label: "Completed", value: stats.campaigns.completed, color: "bg-blue-400" },
            ].map(({ label, value, color }) => {
              const pct = stats.campaigns.total ? Math.round((value / stats.campaigns.total) * 100) : 0;
              return (
                <div key={label} className="space-y-1">
                  <div className="flex justify-between text-sm">
                    <span>{label}</span>
                    <span className="font-medium">{value} ({pct}%)</span>
                  </div>
                  <div className="h-2 rounded-full bg-muted overflow-hidden">
                    <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
                  </div>
                </div>
              );
            })}
            <div className="pt-2 border-t">
              <p className="text-sm text-muted-foreground">Total Campaign Budget</p>
              <p className="text-xl font-bold">{formatCurrency(stats.campaigns.totalBudget)}</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
