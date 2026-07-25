import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const businessSteps = [
  "Top up your wallet — campaign budgets are paid from it.",
  "Create a campaign: set a title, target link, budget, and payout per repost.",
  "Your campaign goes live in the marketplace for promoters to find.",
  "Approved reposts are paid automatically from your locked budget.",
];

const promoterSteps = [
  "Connect your Instagram or Twitter account.",
  "Browse open campaigns and pick ones you want to repost.",
  "Repost it, then submit the link and a screenshot.",
  "Once your submission is approved, the payout lands in your wallet.",
  "Withdraw your balance to your bank account anytime.",
];

function StepList({ title, steps }: { title: string; steps: string[] }) {
  return (
    <Card>
      <CardHeader><CardTitle>{title}</CardTitle></CardHeader>
      <CardContent>
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
      </CardContent>
    </Card>
  );
}

export function GettingStarted() {
  return (
    <section className="mx-auto max-w-6xl px-4 py-16">
      <h2 className="text-center text-2xl font-bold sm:text-3xl">Getting started</h2>
      <div className="mt-10 grid gap-6 sm:grid-cols-2">
        <StepList title="Setting up a campaign" steps={businessSteps} />
        <StepList title="Reposting and earning" steps={promoterSteps} />
      </div>
    </section>
  );
}
