"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { walletApi } from "@/lib/api";
import { toast } from "@/components/ui/use-toast";

export default function ConnectRefreshPage() {
  const router = useRouter();

  useEffect(() => {
    walletApi.createConnect()
      .then((res: any) => { window.location.href = res.data.data.url; })
      .catch(() => {
        toast({ title: "Could not refresh onboarding link", variant: "destructive" });
        router.replace("/dashboard/wallet");
      });
  }, []);

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        <p className="text-muted-foreground">Refreshing onboarding link…</p>
      </div>
    </div>
  );
}
