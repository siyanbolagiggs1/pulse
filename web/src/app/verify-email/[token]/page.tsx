"use client";
import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { authApi } from "@/lib/api";
import { CheckCircle, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import Link from "next/link";

export default function VerifyEmailPage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const called = useRef(false);

  useEffect(() => {
    if (called.current) return;
    called.current = true;

    authApi.verifyEmail(token)
      .then(() => {
        setStatus("success");
        setTimeout(() => router.push("/login"), 3000);
      })
      .catch(() => setStatus("error"));
  }, [token]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="text-center space-y-4 max-w-md px-6">
        {status === "loading" && (
          <>
            <div className="mx-auto h-12 w-12 animate-spin rounded-full border-2 border-primary border-t-transparent" />
            <p className="text-muted-foreground">Verifying your email…</p>
          </>
        )}
        {status === "success" && (
          <>
            <CheckCircle className="mx-auto h-12 w-12 text-green-400" />
            <h2 className="text-2xl font-bold">Email Verified!</h2>
            <p className="text-muted-foreground">Your account is now active. Redirecting to login…</p>
            <Button asChild><Link href="/login">Go to Login</Link></Button>
          </>
        )}
        {status === "error" && (
          <>
            <XCircle className="mx-auto h-12 w-12 text-destructive" />
            <h2 className="text-2xl font-bold">Verification Failed</h2>
            <p className="text-muted-foreground">This link may have expired or already been used.</p>
            <Button asChild variant="outline"><Link href="/login">Back to Login</Link></Button>
          </>
        )}
      </div>
    </div>
  );
}
