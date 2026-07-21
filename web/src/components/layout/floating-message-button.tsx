"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { MessageCircle } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { useRealtime } from "@/providers/realtime";

// Persistent shortcut to Messages, visible on every dashboard screen for
// regular (non-admin) users, who participate in chat. Hidden on the
// Messages pages themselves since a button to get there is redundant once
// you're there.
export function FloatingMessageButton() {
  const user = useAuthStore((s) => s.user);
  const pathname = usePathname();
  const { unreadMessages } = useRealtime();

  if (user?.role === "admin") return null;
  if (pathname.startsWith("/dashboard/messages")) return null;

  return (
    <Link
      href="/dashboard/messages"
      className="fixed bottom-6 right-6 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105"
      aria-label="Messages"
    >
      <MessageCircle className="h-6 w-6" />
      {unreadMessages > 0 && (
        <span className="absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-destructive px-1 text-[11px] font-medium text-destructive-foreground">
          {unreadMessages > 9 ? "9+" : unreadMessages}
        </span>
      )}
    </Link>
  );
}
