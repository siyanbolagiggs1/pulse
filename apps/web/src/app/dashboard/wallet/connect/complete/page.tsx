"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { walletApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { toast } from "@/components/ui/use-toast";

export default function ConnectCompletePage() {
  const router = useRouter();
  const updateUser = useAuthStore((s) => s.updateUser);
  const user = useAuthStore((s) => s.user);

  useEffect(() => {
    walletApi.getConnectStatus()
      .then((res: any) => {
        if (user) updateUser({ ...user, stripeConnectStatus: res.data.data.status });
        toast({ title: "Stripe Connect updated", description: `Status: ${res.data.data.status}` });
      })
      .catch(() => toast({ title: "Could not verify status", variant: "destructive" }))
      .finally(() => router.replace("/dashboard/wallet"));
  }, []);

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        <p className="text-muted-foreground">Verifying your Stripe account…</p>
      </div>
    </div>
  );
}
