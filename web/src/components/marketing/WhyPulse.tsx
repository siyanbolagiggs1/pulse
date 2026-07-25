import { X, Check } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const ads = [
  "Shown to strangers picked by an algorithm",
  "Engagement stops the moment the budget runs out",
  "No relationship with the people who see it",
];

const pulse = [
  "Shared by real people your audience already follows and trusts",
  "Rewards the followers you already have, not just strangers",
  "Grows your own reach as promoters bring in new followers",
];

export function WhyPulse() {
  return (
    <section className="mx-auto max-w-6xl px-4 py-16">
      <h2 className="text-center text-2xl font-bold sm:text-3xl">Not just another ad buy</h2>
      <p className="mx-auto mt-3 max-w-2xl text-center text-muted-foreground">
        Paid ads rent attention. Pulse turns your own community into your growth engine.
      </p>
      <div className="mt-10 grid gap-6 sm:grid-cols-2">
        <Card>
          <CardHeader><CardTitle className="text-muted-foreground">Generic paid ads</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {ads.map((line) => (
              <div key={line} className="flex items-start gap-2">
                <X className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">{line}</p>
              </div>
            ))}
          </CardContent>
        </Card>
        <Card className="border-primary/30">
          <CardHeader><CardTitle className="text-primary">Pulse</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {pulse.map((line) => (
              <div key={line} className="flex items-start gap-2">
                <Check className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                <p className="text-sm">{line}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
