"use client";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { campaignsApi } from "@/lib/api";
import type { Campaign } from "@/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/use-toast";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { format } from "date-fns";

const schema = z.object({
  title: z.string().min(3).max(100),
  description: z.string().min(10).max(2000),
  targetUrl: z.string().url("Must be a valid URL"),
  baseRepostRate: z.coerce.number().min(1, "Minimum payout is ₦1"),
  minFollowers: z.coerce.number().min(0),
  minInfluenceScore: z.coerce.number().min(0).max(100),
  maxParticipants: z.coerce.number().min(1),
  endDate: z.string().min(1, "Required"),
});
type FormData = z.infer<typeof schema>;

export default function EditCampaignPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const { register, handleSubmit, reset, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  useEffect(() => {
    campaignsApi.get(id)
      .then((r) => {
        const c = r.data.data;
        setCampaign(c);
        reset({
          title: c.title,
          description: c.description,
          targetUrl: c.targetUrl,
          baseRepostRate: c.baseRepostRate,
          minFollowers: c.minFollowers,
          minInfluenceScore: c.minInfluenceScore,
          maxParticipants: c.maxParticipants,
          endDate: format(new Date(c.endDate), "yyyy-MM-dd"),
        });
      })
      .catch(() => toast({ title: "Failed to load advert", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [id]);

  const onSubmit = async (data: FormData) => {
    setSaving(true);
    try {
      await campaignsApi.update(id, {
        ...data,
        endDate: data.endDate ? new Date(data.endDate).toISOString() : undefined,
      });
      toast({ title: "Advert updated" });
      router.push(`/dashboard/campaigns/${id}`);
    } catch (err: any) {
      toast({ title: "Failed to update", description: err?.response?.data?.message, variant: "destructive" });
    } finally { setSaving(false); }
  };

  const field = (name: keyof FormData, label: string, type = "text", placeholder = "") => (
    <div className="space-y-1">
      <Label htmlFor={name}>{label}</Label>
      <Input id={name} type={type} placeholder={placeholder} {...register(name)} />
      {errors[name] && <p className="text-xs text-destructive">{errors[name]?.message as string}</p>}
    </div>
  );

  if (loading) return <Skeleton className="h-96 w-full" />;
  if (!campaign) return <p>Advert not found.</p>;

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/dashboard/campaigns/${id}`}><ArrowLeft className="h-4 w-4" /></Link>
        </Button>
        <div>
          <h2 className="text-2xl font-bold">Edit Advert</h2>
          <p className="text-muted-foreground capitalize">{campaign.platform} · {campaign.status}</p>
        </div>
      </div>

      <Card>
        <CardHeader><CardTitle>Advert Details</CardTitle></CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {field("title", "Title")}

            <div className="space-y-1">
              <Label>Description</Label>
              <Textarea rows={4} {...register("description")} />
              {errors.description && <p className="text-xs text-destructive">{errors.description.message}</p>}
            </div>

            {field("targetUrl", "Target URL", "url")}

            <div className="grid grid-cols-2 gap-4">
              {field("baseRepostRate", "Base Payout per Repost ($)", "number")}
              {field("maxParticipants", "Max Participants", "number")}
            </div>

            <div className="grid grid-cols-2 gap-4">
              {field("minFollowers", "Min Followers", "number")}
              {field("minInfluenceScore", "Min Influence Score", "number")}
            </div>

            {field("endDate", "End Date", "date")}

            <p className="text-xs text-muted-foreground">
              Note: platform and budget cannot be changed after creation.
            </p>

            <Button type="submit" className="w-full" disabled={saving}>
              {saving ? "Saving…" : "Save Changes"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
