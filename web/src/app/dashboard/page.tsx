"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { walletApi, campaignsApi } from "@/lib/api";
import type { Wallet, Campaign } from "@/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { CampaignPreviewCard } from "@/components/shared/CampaignPreviewCard";
import { formatCurrency } from "@/lib/utils";
import { Megaphone, Coins, ArrowRight } from "lucide-react";
import { toast } from "@/components/ui/use-toast";

const PREVIEW_COUNT = 3;

export default function DashboardPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);

  const [wallet, setWallet] = useState<Wallet | null>(null);
  const [myCampaigns, setMyCampaigns] = useState<Campaign[]>([]);
  const [marketplace, setMarketplace] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) return;
    if (user.role === "admin") {
      router.replace("/dashboard/admin");
      return;
    }

    Promise.all([
      walletApi.get(),
      campaignsApi.getMy({ page: 1, limit: PREVIEW_COUNT }),
      campaignsApi.list({}),
    ])
      .then(([w, mine, all]) => {
        setWallet(w.data.data);
        setMyCampaigns(mine.data.data);
        setMarketplace(all.data.data.slice(0, PREVIEW_COUNT));
      })
      .catch(() => toast({ title: "Failed to load overview", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [user]);

  if (!user || user.role === "admin") return null;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Welcome back, {user.name}</h2>
        <p className="text-muted-foreground">Here's what's happening across your account</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2"><Megaphone className="h-5 w-5" />Create Adverts</CardTitle>
              <Button asChild variant="ghost" size="sm">
                <Link href="/dashboard/campaigns">View all<ArrowRight className="ml-1 h-4 w-4" /></Link>
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex gap-3 overflow-x-auto pb-1">
                {[...Array(3)].map((_, i) => <Skeleton key={i} className="h-44 w-56 shrink-0 rounded-lg" />)}
              </div>
            ) : myCampaigns.length === 0 ? (
              <div className="py-6 text-center text-sm text-muted-foreground">
                No adverts yet.{" "}
                <Link href="/dashboard/campaigns/new" className="text-primary hover:underline">Create one</Link>
              </div>
            ) : (
              <div className="flex gap-3 overflow-x-auto pb-1 snap-x snap-mandatory">
                {myCampaigns.map((c) => (
                  <CampaignPreviewCard
                    key={c.id}
                    campaign={c}
                    href={`/dashboard/campaigns/${c.id}`}
                    footer={
                      <>
                        <span className="text-xs text-muted-foreground">
                          {c.currentParticipants}/{c.maxParticipants}
                        </span>
                        <Badge variant={c.status === "active" ? "success" : "secondary"} className="shrink-0">
                          {c.status}
                        </Badge>
                      </>
                    }
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2"><Coins className="h-5 w-5" />Earn Hub</CardTitle>
              <Button asChild variant="ghost" size="sm">
                <Link href="/dashboard/marketplace">View all<ArrowRight className="ml-1 h-4 w-4" /></Link>
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">Repost an advert and earn a payout once it's approved</p>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex gap-3 overflow-x-auto pb-1">
                {[...Array(3)].map((_, i) => <Skeleton key={i} className="h-44 w-56 shrink-0 rounded-lg" />)}
              </div>
            ) : marketplace.length === 0 ? (
              <p className="py-6 text-center text-sm text-muted-foreground">No open adverts right now.</p>
            ) : (
              <div className="flex gap-3 overflow-x-auto pb-1 snap-x snap-mandatory">
                {marketplace.map((c) => (
                  <CampaignPreviewCard
                    key={c.id}
                    campaign={c}
                    href={`/dashboard/marketplace/${c.id}`}
                    footer={
                      <span className="text-xs text-muted-foreground">
                        Base payout {formatCurrency(c.baseRepostRate)}
                      </span>
                    }
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {loading ? (
        <div className="grid gap-4 sm:grid-cols-4">{[...Array(4)].map((_, i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-4">
          <Card><CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">Available</p>
            <p className="text-2xl font-bold text-green-400">{formatCurrency(wallet?.availableBalance ?? 0)}</p>
          </CardContent></Card>
          <Card><CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">Pending</p>
            <p className="text-2xl font-bold text-yellow-400">{formatCurrency(wallet?.pendingBalance ?? 0)}</p>
          </CardContent></Card>
          <Card><CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">Total Earned</p>
            <p className="text-2xl font-bold">{formatCurrency(wallet?.totalEarned ?? 0)}</p>
          </CardContent></Card>
          <Card><CardContent className="pt-6">
            <p className="text-sm text-muted-foreground">Total Spent</p>
            <p className="text-2xl font-bold">{formatCurrency(wallet?.totalSpent ?? 0)}</p>
          </CardContent></Card>
        </div>
      )}
    </div>
  );
}
