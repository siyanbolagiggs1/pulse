export const metadata = { title: "Terms and Conditions — Pulse" };

export default function TermsPage() {
  return (
    <main className="mx-auto max-w-3xl px-4 py-16">
      <h1 className="text-3xl font-bold">Terms and Conditions</h1>
      <p className="mt-2 text-sm text-muted-foreground">Last updated: 26 July 2026</p>

      <div className="mt-8 space-y-8 text-sm leading-relaxed text-foreground">
        <p className="rounded-lg bg-accent p-4 text-muted-foreground">
          This is a draft prepared to reflect how Pulse actually operates and Nigerian law as it
          currently stands (the Federal Competition and Consumer Protection Act 2018 and the
          Nigeria Data Protection Act 2023). It is not a substitute for review by a qualified
          Nigerian lawyer before being relied on as a binding legal agreement.
        </p>

        <section>
          <h2 className="text-lg font-semibold">1. Who we are and what these Terms cover</h2>
          <p>
            These Terms are between you and Social Technologies (RC 9706684), a company
            registered in Nigeria with its registered address at F.G.C.I SSQ17, Ogidi, Senior
            Staff Quarters, beside Girls Hostel, Ilorin, Kwara State, Nigeria, which operates the
            Pulse platform ("Pulse", "we", "us").
          </p>
          <p className="mt-3">
            Pulse is a social engagement marketplace: businesses create repost campaigns and fund
            a budget for them; promoters repost that content on their own Instagram or Twitter/X
            accounts and earn money for approved reposts. These Terms govern your use of the
            Pulse website and app (the "Platform") and apply to every account holder, whether you
            use Pulse as a business or as a promoter.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">2. Eligibility and your account</h2>
          <p>
            You must be at least 18 years old and able to form a binding contract under Nigerian
            law to use Pulse. You're responsible for the accuracy of the information you provide
            at signup, for keeping your login credentials secure, and for all activity that
            happens under your account. Tell us immediately if you believe your account has been
            compromised.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">3. Wallet, payments, and fees</h2>
          <p>
            All payments on Pulse — business top-ups, promoter withdrawals, and campaign payouts
            — are processed through Paystack, a licensed payment service provider. Pulse itself
            does not hold a separate banking or payment-service licence; we rely on Paystack's
            licensed infrastructure to move money on our behalf. By topping up or withdrawing
            funds, you also agree to Paystack's own terms for the transaction.
          </p>
          <p className="mt-3">
            Businesses fund campaigns from their Pulse wallet. Pulse deducts a platform
            commission (currently 20% by default, and may change with notice) from each approved
            payout before crediting the promoter. Once a campaign's budget is locked to fund a
            campaign, it is spent as submissions are approved; unused budget from a deleted or
            completed campaign is returned to the business's wallet.
          </p>
          <p className="mt-3">
            Promoter earnings are added to your available wallet balance once a submission is
            approved, and may be withdrawn to a verified bank account at any time, subject to the
            minimum withdrawal amount shown in the app. Withdrawals are reviewed before funds are
            sent — we may decline or delay a withdrawal we reasonably believe is connected to
            fraud, a compromised account, or a breach of these Terms.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">4. Campaigns and submissions</h2>
          <p>
            Businesses are solely responsible for the content, claims, and legality of their
            campaigns and the pages they link to. Pulse does not review campaign links for
            accuracy before they go live and is not a party to any transaction between a business
            and its own customers.
          </p>
          <p className="mt-3">
            Promoters must genuinely repost the linked content on a social account they own and
            control, and submit accurate proof (the post link and a screenshot). Submitting a
            fake, deleted-after-submission, or otherwise misleading repost is a breach of these
            Terms and may result in submission rejection, a lowered trust score, loss of pending
            earnings tied to that submission, and account suspension.
          </p>
          <p className="mt-3">
            Submissions are reviewed by Pulse before approval. Approval or rejection decisions,
            and the influence/trust scoring used to support them, are made at Pulse's reasonable
            discretion based on the information available at the time.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">5. Prohibited conduct</h2>
          <p>You may not use Pulse to:</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            <li>Submit fraudulent, fabricated, or manipulated proof of a repost.</li>
            <li>Use bots, fake accounts, or purchased followers/engagement.</li>
            <li>Run campaigns linking to illegal, fraudulent, or deceptive content.</li>
            <li>Attempt to circumvent fraud detection, trust scoring, or withdrawal review.</li>
            <li>Interfere with the Platform's operation or another user's account.</li>
          </ul>
          <p className="mt-3">
            We may suspend or terminate accounts that breach this section, withhold funds
            reasonably connected to the breach pending investigation, and where required by law,
            report suspected fraud to the relevant authorities or to Paystack.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">6. Content and licence</h2>
          <p>
            Businesses retain ownership of their campaign content but grant Pulse a licence to
            display it on the Platform for the purpose of running the campaign. Promoters retain
            ownership of their own reposts but grant Pulse a licence to view and store repost
            links and screenshots submitted as proof, for as long as reasonably necessary to
            review, audit, and resolve disputes about that submission.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">7. Disclaimers and limitation of liability</h2>
          <p>
            The Platform is provided "as is." We don't guarantee that campaigns will attract a
            particular level of engagement, that a given submission will be approved, or that the
            Platform will be uninterrupted or error-free. To the fullest extent permitted under
            Nigerian law, Pulse is not liable for indirect, incidental, or consequential losses
            arising from your use of the Platform. In no event shall Pulse's total liability for
            any claim exceed the amount you paid to or through Pulse in the 12 months preceding
            the event giving rise to the claim.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">8. Indemnification</h2>
          <p>
            You agree to indemnify and hold Pulse, its officers, and employees harmless from any
            claim, loss, liability, or expense (including reasonable legal fees) arising out of or
            connected to: your use of the Platform, your breach of these Terms, the content of any
            campaign you create, or any repost or submission you make.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">9. Suspension and termination</h2>
          <p>
            You may close your account at any time from your Profile, subject to first
            withdrawing any available balance. We may suspend or terminate an account for breach
            of these Terms, suspected fraud, or a legal or regulatory requirement, and will
            explain why where we're able to.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">10. Changes to these Terms</h2>
          <p>
            We may update these Terms as Pulse evolves. We'll post the updated version here with
            a new "last updated" date, and for material changes we'll make reasonable efforts to
            notify you in-app before they take effect.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">11. Governing law and dispute resolution</h2>
          <p>
            These Terms are governed by, and construed in accordance with, the laws of the
            Federal Republic of Nigeria, including the Federal Competition and Consumer
            Protection Act 2018. Any dispute will first be raised with Pulse support for an
            attempt at informal resolution; if it isn't resolved that way, the dispute shall be
            submitted to the exclusive jurisdiction of the courts of Nigeria.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">12. Contact</h2>
          <p>Questions about these Terms can be sent through the Message Support option in the app, or to [insert support email].</p>
        </section>
      </div>
    </main>
  );
}
