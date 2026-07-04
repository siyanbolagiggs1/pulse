"use client";
import { useEffect, useState, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { campaignsApi, submissionsApi, usersApi } from "@/lib/api";
import type { Campaign, SocialAccount } from "@/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency, formatNumber } from "@/lib/utils";
import { ArrowLeft, Upload } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

export default function CampaignApplyPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [accounts, setAccounts] = useState<SocialAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedAccount, setSelectedAccount] = useState("");
  const [repostUrl, setRepostUrl] = useState("");
  const [screenshotUrl, setScreenshotUrl] = useState("");
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    Promise.all([campaignsApi.get(id), usersApi.getMe()])
      .then(([cr, ur]) => {
        setCampaign(cr.data.data);
        const allAccounts = ur.data.data.socialAccounts ?? [];
        setAccounts(allAccounts.filter((a: SocialAccount) => a.platform === cr.data.data.platform));
      })
      .catch(() => toast({ title: "Failed to load", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [id]);

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      const res = await submissionsApi.uploadScreenshot(file);
      setScreenshotUrl(res.data.data.url);
      toast({ title: "Screenshot uploaded" });
    } catch {
      toast({ title: "Upload failed", variant: "destructive" });
    } finally { setUploading(false); }
  };

  const handleSubmit = async () => {
    if (!selectedAccount || !repostUrl || !screenshotUrl) {
      return toast({ title: "Please fill all fields", variant: "destructive" });
    }
    setSubmitting(true);
    try {
      await submissionsApi.create({ campaignId: id, socialAccountId: selectedAccount, repostUrl, screenshotUrl });
      toast({ title: "Submission received!", description: "The business will review it." });
      router.push("/dashboard/submissions");
    } catch (err: any) {
      toast({ title: "Submission failed", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setSubmitting(false); }
  };

  if (loading) return <Skeleton className="h-96 w-full" />;
  if (!campaign) return <p>Advert not found.</p>;

  // Only admin-verified accounts are selectable. Raw follower count and
  // influence score aren't exposed to users, so we can't pre-filter on
  // campaign minimums here anymore — the server enforces the real
  // eligibility gate at submission time, surfacing a clear error if a
  // technically-ineligible account is chosen.
  const eligible = accounts.filter((a) => a.status === "active");

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild><Link href="/dashboard/marketplace"><ArrowLeft className="h-4 w-4" /></Link></Button>
        <div>
          <h2 className="text-2xl font-bold">{campaign.title}</h2>
          <p className="text-muted-foreground capitalize">{campaign.platform}</p>
        </div>
      </div>

      <Card>
        <CardHeader><CardTitle>Advert Details</CardTitle></CardHeader>
        <CardContent className="space-y-3 text-sm">
          <p>{campaign.description}</p>
          <a href={campaign.targetUrl} target="_blank" rel="noreferrer" className="text-primary hover:underline break-all">{campaign.targetUrl}</a>
          <div className="grid grid-cols-2 gap-3 pt-2">
            {[
              ["Base Payout", formatCurrency(campaign.baseRepostRate)],
              ["Ends", format(new Date(campaign.endDate), "MMM d, yyyy")],
              ["Min Followers", formatNumber(campaign.minFollowers)],
            ].map(([k, v]) => (
              <div key={k}><p className="text-muted-foreground">{k}</p><p className="font-medium">{v}</p></div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Submit Your Repost</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          {eligible.length === 0 ? (
            <p className="text-sm text-destructive">None of your {campaign.platform} accounts meet the eligibility requirements.</p>
          ) : (
            <>
              <div className="space-y-1">
                <Label>Select Social Account</Label>
                <Select value={selectedAccount} onValueChange={setSelectedAccount}>
                  <SelectTrigger><SelectValue placeholder="Choose account…" /></SelectTrigger>
                  <SelectContent>
                    {eligible.map((a) => (
                      <SelectItem key={a.id} value={a.id}>
                        @{a.username} · Tier {a.tier}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label>Repost URL</Label>
                <Input placeholder="https://instagram.com/p/…" value={repostUrl} onChange={(e) => setRepostUrl(e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label>Proof Screenshot</Label>
                <input type="file" ref={fileRef} accept="image/*" className="hidden" onChange={handleUpload} />
                <Button variant="outline" className="w-full" onClick={() => fileRef.current?.click()} disabled={uploading}>
                  <Upload className="mr-2 h-4 w-4" />
                  {uploading ? "Uploading…" : screenshotUrl ? "Screenshot uploaded ✓" : "Upload screenshot"}
                </Button>
              </div>
              <Button className="w-full" onClick={handleSubmit} disabled={submitting || !selectedAccount || !repostUrl || !screenshotUrl}>
                {submitting ? "Submitting…" : "Submit for Review"}
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
