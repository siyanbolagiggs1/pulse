"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { walletApi } from "@/lib/api";
import { CheckCircle, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import Link from "next/link";

export default function TopupCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const called = useRef(false);

  useEffect(() => {
    if (called.current) return;
    called.current = true;

    const reference = searchParams.get("reference") || searchParams.get("trxref");
    if (!reference) {
      setStatus("error");
      return;
    }

    walletApi.verifyTopup(reference)
      .then(() => {
        setStatus("success");
        setTimeout(() => router.push("/dashboard/wallet"), 3000);
      })
      .catch(() => setStatus("error"));
  }, []);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="text-center space-y-4 max-w-md px-6">
        {status === "loading" && (
          <>
            <div className="mx-auto h-12 w-12 animate-spin rounded-full border-2 border-primary border-t-transparent" />
            <p className="text-muted-foreground">Verifying your payment…</p>
          </>
        )}
        {status === "success" && (
          <>
            <CheckCircle className="mx-auto h-12 w-12 text-green-400" />
            <h2 className="text-2xl font-bold">Payment Successful!</h2>
            <p className="text-muted-foreground">Your wallet has been topped up. Redirecting…</p>
            <Button asChild><Link href="/dashboard/wallet">Go to Wallet</Link></Button>
          </>
        )}
        {status === "error" && (
          <>
            <XCircle className="mx-auto h-12 w-12 text-destructive" />
            <h2 className="text-2xl font-bold">Payment Failed</h2>
            <p className="text-muted-foreground">Your payment could not be verified. If you were charged, contact support.</p>
            <Button asChild variant="outline"><Link href="/dashboard/wallet">Back to Wallet</Link></Button>
          </>
        )}
      </div>
    </div>
  );
}
