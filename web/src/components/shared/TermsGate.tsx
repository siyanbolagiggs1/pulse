"use client";
import { useState } from "react";
import Link from "next/link";
import { authApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "@/components/ui/use-toast";

export function TermsGate() {
  const { user, updateUser } = useAuthStore();
  const [loading, setLoading] = useState(false);

  const handleAccept = async () => {
    if (!user) return;
    setLoading(true);
    try {
      await authApi.acceptTerms();
      updateUser({ ...user, termsAccepted: true });
    } catch {
      toast({ title: "Something went wrong, please try again", variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>We've updated our Terms</CardTitle>
          <CardDescription>
            Please review and accept our Terms and Conditions and Privacy Policy to continue using Pulse.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-4 text-sm">
            <Link href="/terms" target="_blank" className="text-primary hover:underline">Read Terms and Conditions</Link>
            <Link href="/privacy" target="_blank" className="text-primary hover:underline">Read Privacy Policy</Link>
          </div>
          <Button className="w-full" onClick={handleAccept} disabled={loading}>
            {loading ? "Saving…" : "I Agree, Continue"}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
