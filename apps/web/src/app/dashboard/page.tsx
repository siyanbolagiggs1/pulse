"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";

export default function DashboardPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);

  useEffect(() => {
    if (!user) return;
    if (user.role === "admin") router.replace("/dashboard/admin");
    else if (user.role === "business") router.replace("/dashboard/campaigns");
    else router.replace("/dashboard/marketplace");
  }, [user]);

  return null;
}
