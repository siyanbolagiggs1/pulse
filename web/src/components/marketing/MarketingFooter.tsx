import Link from "next/link";

export function MarketingFooter() {
  return (
    <footer className="border-t border-border bg-card">
      <div className="mx-auto max-w-6xl px-4 py-8">
        <div className="flex flex-col items-center justify-between gap-4 sm:flex-row">
          <div>
            <p className="text-lg font-bold text-primary">Pulse</p>
            <p className="text-sm text-muted-foreground">Community-powered social promotion</p>
          </div>
          <nav className="flex items-center gap-6">
            <Link href="/login" className="text-sm text-muted-foreground hover:text-foreground">Sign in</Link>
            <Link href="/register" className="text-sm text-muted-foreground hover:text-foreground">Get started</Link>
          </nav>
        </div>
        <p className="mt-6 text-center text-xs text-muted-foreground sm:text-left">
          © {new Date().getFullYear()} Pulse. All rights reserved.
        </p>
      </div>
    </footer>
  );
}
