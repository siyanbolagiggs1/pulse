"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function ConnectCompletePage() {
  const router = useRouter();
  useEffect(() => { router.replace("/dashboard/wallet"); }, []);
  return null;
}
