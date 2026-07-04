"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  LayoutDashboard, Megaphone, Store, FileText, Wallet,
  Users, AlertTriangle, ArrowDownToLine, LogOut, UserCircle, ShieldCheck, MessageCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { useRealtime } from "@/providers/realtime";

type NavItem = { label: string; href: string; icon: React.ElementType };

const businessNav: NavItem[] = [
  { label: "My Adverts", href: "/dashboard/campaigns", icon: Megaphone },
  { label: "Submissions", href: "/dashboard/submissions", icon: FileText },
  { label: "Messages", href: "/dashboard/messages", icon: MessageCircle },
  { label: "Wallet", href: "/dashboard/wallet", icon: Wallet },
  { label: "Profile", href: "/dashboard/profile", icon: UserCircle },
];

const promoterNav: NavItem[] = [
  { label: "Marketplace", href: "/dashboard/marketplace", icon: Store },
  { label: "My Submissions", href: "/dashboard/submissions", icon: FileText },
  { label: "Messages", href: "/dashboard/messages", icon: MessageCircle },
  { label: "Wallet", href: "/dashboard/wallet", icon: Wallet },
  { label: "Profile", href: "/dashboard/profile", icon: UserCircle },
];

const adminNav: NavItem[] = [
  { label: "Overview", href: "/dashboard/admin", icon: LayoutDashboard },
  { label: "Users", href: "/dashboard/admin/users", icon: Users },
  { label: "Submissions", href: "/dashboard/admin/submissions", icon: FileText },
  { label: "Fraud Flags", href: "/dashboard/admin/fraud-flags", icon: AlertTriangle },
  { label: "Withdrawals", href: "/dashboard/admin/withdrawals", icon: ArrowDownToLine },
  { label: "Social Accounts", href: "/dashboard/admin/social-accounts", icon: ShieldCheck },
  { label: "Conversations", href: "/dashboard/admin/conversations", icon: MessageCircle },
  { label: "Profile", href: "/dashboard/profile", icon: UserCircle },
];

function NavContent({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const { unreadMessages } = useRealtime();

  const nav = user?.role === "admin" ? adminNav : user?.role === "business" ? businessNav : promoterNav;

  const handleLogout = async () => {
    try { await authApi.logout(); } catch {}
    clearAuth();
    router.push("/login");
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-16 items-center px-6 border-b border-border">
        <span className="text-xl font-bold text-primary">Pulse</span>
        {user?.role === "admin" && (
          <span className="ml-2 rounded-full bg-primary/20 px-2 py-0.5 text-xs text-primary">Admin</span>
        )}
      </div>

      <nav className="flex-1 space-y-1 p-3">
        {nav.map(({ label, href, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            onClick={onNavigate}
            className={cn(
              "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
              pathname.startsWith(href)
                ? "bg-primary/10 text-primary font-medium"
                : "text-muted-foreground hover:bg-accent hover:text-foreground"
            )}
          >
            <Icon className="h-4 w-4" />
            <span className="flex-1">{label}</span>
            {href === "/dashboard/messages" && unreadMessages > 0 && (
              <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-primary px-1 text-[11px] font-medium text-primary-foreground">
                {unreadMessages > 9 ? "9+" : unreadMessages}
              </span>
            )}
          </Link>
        ))}
      </nav>

      <div className="border-t border-border p-3">
        <div className="mb-2 px-3 py-1">
          <p className="text-sm font-medium truncate">{user?.name}</p>
          <p className="text-xs text-muted-foreground truncate">{user?.email}</p>
        </div>
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <LogOut className="h-4 w-4" />
          Sign out
        </button>
      </div>
    </div>
  );
}

export function Sidebar() {
  return (
    <aside className="hidden md:flex h-full w-60 flex-col border-r border-border bg-card">
      <NavContent />
    </aside>
  );
}

export function MobileSidebar({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Sheet open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <SheetContent>
        <NavContent onNavigate={onClose} />
      </SheetContent>
    </Sheet>
  );
}
