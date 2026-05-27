"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { campaignsApi } from "@/lib/api";
import type { Campaign } from "@/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency, formatNumber } from "@/lib/utils";
import { Search, Users, TrendingUp } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

export default function MarketplacePage() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [platform, setPlatform] = useState("all");

  useEffect(() => {
    const params: Record<string, string> = {};
    if (platform !== "all") params.platform = platform;
    campaignsApi.list(params)
      .then((r) => setCampaigns(r.data.data))
      .catch(() => toast({ title: "Failed to load marketplace", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [platform]);

  const filtered = campaigns.filter((c) =>
    !search || c.title.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Marketplace</h2>
        <p className="text-muted-foreground">Browse campaigns and earn by sharing</p>
      </div>

      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input className="pl-9" placeholder="Search campaigns…" value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
        <Select value={platform} onValueChange={setPlatform}>
          <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All platforms</SelectItem>
            <SelectItem value="instagram">Instagram</SelectItem>
            <SelectItem value="twitter">Twitter / X</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {loading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(6)].map((_, i) => <Skeleton key={i} className="h-52" />)}
        </div>
      ) : filtered.length === 0 ? (
        <Card><CardContent className="py-12 text-center text-muted-foreground">No campaigns found</CardContent></Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filtered.map((c) => (
            <Card key={c.id} className="flex flex-col hover:border-primary/50 transition-colors">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <CardTitle className="text-base line-clamp-1">{c.title}</CardTitle>
                  <Badge variant="secondary" className="capitalize shrink-0">{c.platform}</Badge>
                </div>
                <CardDescription className="line-clamp-2">{c.description}</CardDescription>
              </CardHeader>
              <CardContent className="flex-1 space-y-3">
                <div className="flex items-center justify-between text-sm">
                  <span className="font-semibold text-green-400">Up to {formatCurrency(c.baseRepostRate * 1.5)}</span>
                  <span className="text-muted-foreground">base {formatCurrency(c.baseRepostRate)}</span>
                </div>
                <div className="flex gap-3 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1"><Users className="h-3 w-3" />{c.currentParticipants}/{c.maxParticipants}</span>
                  <span className="flex items-center gap-1"><TrendingUp className="h-3 w-3" />{formatNumber(c.minFollowers)}+ followers</span>
                </div>
                <p className="text-xs text-muted-foreground">Ends {format(new Date(c.endDate), "MMM d")}</p>
                <Button asChild className="w-full" size="sm">
                  <Link href={`/dashboard/marketplace/${c.id}`}>View Campaign</Link>
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
