import { Megaphone, Coins } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const columns = [
  {
    icon: Megaphone,
    title: "For businesses",
    description:
      "Post a repost campaign with a budget and a target link. Promoters share it, and you pay only for approved reposts.",
  },
  {
    icon: Coins,
    title: "For promoters",
    description:
      "Browse open campaigns, repost the ones you like, and get paid to your wallet once your submission is approved.",
  },
];

export function HowItWorks() {
  return (
    <section id="how-it-works" className="mx-auto max-w-6xl px-4 py-16">
      <h2 className="text-center text-2xl font-bold sm:text-3xl">How it works</h2>
      <div className="mt-10 grid gap-6 sm:grid-cols-2">
        {columns.map(({ icon: Icon, title, description }) => (
          <Card key={title}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Icon className="h-5 w-5 text-primary" />
                {title}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">{description}</p>
            </CardContent>
          </Card>
        ))}
      </div>
    </section>
  );
}
