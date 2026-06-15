"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { campaignsApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "@/components/ui/use-toast";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";

const schema = z.object({
  title: z.string().min(3).max(100),
  description: z.string().min(10).max(2000),
  targetUrl: z.string().url("Must be a valid URL"),
  platform: z.enum(["instagram", "twitter"]),
  budget: z.coerce.number().min(50, "Minimum budget is $50"),
  baseRepostRate: z.coerce.number().min(1, "Minimum payout is $1"),
  minFollowers: z.coerce.number().min(0),
  minEngagementRate: z.coerce.number().min(0).max(100),
  minInfluenceScore: z.coerce.number().min(0).max(100),
  maxParticipants: z.coerce.number().min(1),
  startDate: z.string().min(1, "Required"),
  endDate: z.string().min(1, "Required"),
});
type FormData = z.infer<typeof schema>;

export default function NewCampaignPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [platform, setPlatform] = useState<"instagram" | "twitter">("instagram");

  const { register, handleSubmit, setValue, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { platform: "instagram", minFollowers: 100, minEngagementRate: 1, minInfluenceScore: 0, maxParticipants: 10 },
  });

  const onSubmit = async (data: FormData) => {
    setLoading(true);
    try {
      await campaignsApi.create({
        ...data,
        startDate: new Date(data.startDate).toISOString(),
        endDate: new Date(data.endDate).toISOString(),
      });
      toast({ title: "Campaign created!", description: "Budget locked from your wallet." });
      router.push("/dashboard/campaigns");
    } catch (err: any) {
      toast({ title: "Failed", description: err?.response?.data?.message ?? "Could not create campaign", variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  const field = (name: keyof FormData, label: string, type = "text", placeholder = "") => (
    <div className="space-y-1">
      <Label htmlFor={name}>{label}</Label>
      <Input id={name} type={type} placeholder={placeholder} {...register(name)} />
      {errors[name] && <p className="text-xs text-destructive">{errors[name]?.message as string}</p>}
    </div>
  );

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild><Link href="/dashboard/campaigns"><ArrowLeft className="h-4 w-4" /></Link></Button>
        <div>
          <h2 className="text-2xl font-bold">New Campaign</h2>
          <p className="text-muted-foreground">Budget is locked from your wallet on creation</p>
        </div>
      </div>

      <Card>
        <CardHeader><CardTitle>Campaign Details</CardTitle></CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {field("title", "Title", "text", "Summer sale repost campaign")}

            <div className="space-y-1">
              <Label>Description</Label>
              <Textarea placeholder="Describe what promoters will share and why..." rows={4} {...register("description")} />
              {errors.description && <p className="text-xs text-destructive">{errors.description.message}</p>}
            </div>

            {field("targetUrl", "Target URL", "url", "https://yoursite.com")}

            <div className="space-y-1">
              <Label>Platform</Label>
              <Select value={platform} onValueChange={(v: "instagram" | "twitter") => { setPlatform(v); setValue("platform", v); }}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="instagram">Instagram</SelectItem>
                  <SelectItem value="twitter">Twitter / X</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="grid grid-cols-2 gap-4">
              {field("budget", "Total Budget ($)", "number", "500")}
              {field("baseRepostRate", "Base Payout per Repost ($)", "number", "5")}
            </div>

            <div className="grid grid-cols-3 gap-4">
              {field("minFollowers", "Min Followers", "number", "100")}
              {field("minEngagementRate", "Min Engagement (%)", "number", "1")}
              {field("minInfluenceScore", "Min Influence Score", "number", "0")}
            </div>

            {field("maxParticipants", "Max Participants", "number", "50")}

            <div className="grid grid-cols-2 gap-4">
              {field("startDate", "Start Date", "date")}
              {field("endDate", "End Date", "date")}
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Creating…" : "Create Campaign"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
