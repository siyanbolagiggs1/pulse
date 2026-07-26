import Link from "next/link";
import { Megaphone, Coins, Landmark, Link2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const businessSteps = [
  "Go to Wallet and top up — your campaign budget is paid from it.",
  "Go to My Adverts → New Advert. Set a title, target link, budget, and payout per repost.",
  "Your advert goes live in the Earn Hub for promoters to find and repost.",
  "Each approved repost pays out from your budget automatically, no extra steps.",
];

const promoterSteps = [
  "Connect a social account and add a bank account (see below) before you can submit or withdraw.",
  "Go to Earn Hub and browse open adverts.",
  "Repost the one you pick, then submit the repost link and a screenshot.",
  "Once your submission is approved, the payout is added to your wallet immediately.",
  "Go to Wallet and withdraw to your bank account whenever you like.",
];

const bankSteps = [
  "Go to Profile, find the Bank Account section, and click Add.",
  "Search for your bank and select it.",
  "Enter your account number — your account name is verified and filled in automatically.",
  "Save. You can update it anytime the same way.",
];

const socialSteps = [
  "Go to Profile, find the Social Accounts section, and click Connect.",
  "Choose Instagram or Twitter/X and enter your handle.",
  "Click Submit for Review — this isn't instant, an admin verifies the account before it's usable.",
  "Once approved, it's ready to use for campaign submissions.",
];

function StepList({ steps }: { steps: string[] }) {
  return (
    <ol className="space-y-4">
      {steps.map((step, i) => (
        <li key={step} className="flex items-start gap-3">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">
            {i + 1}
          </span>
          <p className="text-sm text-muted-foreground">{step}</p>
        </li>
      ))}
    </ol>
  );
}

export default function HowToUsePage() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">How to Use</h2>
        <p className="text-muted-foreground">How to run adverts, and how to repost and earn on Pulse</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Megaphone className="h-5 w-5 text-primary" />
              Running an advert
            </CardTitle>
          </CardHeader>
          <CardContent>
            <StepList steps={businessSteps} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Coins className="h-5 w-5 text-primary" />
              Reposting and earning
            </CardTitle>
          </CardHeader>
          <CardContent>
            <StepList steps={promoterSteps} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Landmark className="h-5 w-5 text-primary" />
              Adding a bank account
            </CardTitle>
          </CardHeader>
          <CardContent>
            <StepList steps={bankSteps} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Link2 className="h-5 w-5 text-primary" />
              Linking a social account
            </CardTitle>
          </CardHeader>
          <CardContent>
            <StepList steps={socialSteps} />
          </CardContent>
        </Card>
      </div>

      <p className="text-center text-sm text-muted-foreground">
        Still stuck? <Link href="/dashboard/messages" className="text-primary hover:underline">Message support</Link> and we'll help you out.
      </p>
    </div>
  );
}
