"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { Link2 } from "lucide-react";
import { campaignsApi } from "@/lib/api";
import type { Campaign, LinkPreview } from "@/types";
import { Skeleton } from "@/components/ui/skeleton";

interface CampaignPreviewCardProps {
  campaign: Campaign;
  href: string;
  /** Secondary row under the title — differs by context (status/participants vs. payout rate). */
  footer: React.ReactNode;
}

export function CampaignPreviewCard({ campaign, href, footer }: CampaignPreviewCardProps) {
  const [preview, setPreview] = useState<LinkPreview | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    campaignsApi
      .getLinkPreview(campaign.id)
      .then((res) => { if (!cancelled) setPreview(res.data.data); })
      .catch(() => { /* card still renders fine without a preview */ })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [campaign.id]);

  return (
    <Link
      href={href}
      className="flex w-56 shrink-0 snap-start flex-col overflow-hidden rounded-lg border border-border bg-card hover:bg-accent"
    >
      <div className="relative h-28 w-full shrink-0 bg-muted">
        {loading ? (
          <Skeleton className="h-full w-full rounded-none" />
        ) : preview?.image ? (
          // eslint-disable-next-line @next/next/no-img-element -- arbitrary external campaign target domains, not in next.config images.remotePatterns
          <img
            src={preview.image}
            alt=""
            className="h-full w-full object-cover"
            loading="lazy"
            onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <Link2 className="h-6 w-6 text-muted-foreground" />
          </div>
        )}
      </div>
      <div className="flex flex-1 flex-col gap-1 p-3">
        <p className="line-clamp-2 text-sm font-medium">{campaign.title}</p>
        <p className="truncate text-xs text-muted-foreground">
          {preview?.domain ?? " "}
        </p>
        <div className="mt-auto flex items-center justify-between pt-1">{footer}</div>
      </div>
    </Link>
  );
}
