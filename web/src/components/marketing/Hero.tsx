import Link from "next/link";
import { Button } from "@/components/ui/button";

export function Hero() {
  return (
    <section className="mx-auto max-w-6xl px-4 py-20 text-center sm:py-28">
      <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
        Community-powered social promotion
      </h1>
      <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
        Businesses run repost campaigns. Promoters earn money sharing them.
      </p>
      <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
        <Button asChild size="lg"><Link href="/register">Get Started</Link></Button>
        <Button asChild size="lg" variant="outline"><Link href="/login">Sign in</Link></Button>
      </div>
    </section>
  );
}
