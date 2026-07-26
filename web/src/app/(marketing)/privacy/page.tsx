export const metadata = { title: "Privacy Policy — Pulse" };

export default function PrivacyPage() {
  return (
    <main className="mx-auto max-w-3xl px-4 py-16">
      <h1 className="text-3xl font-bold">Privacy Policy</h1>
      <p className="mt-2 text-sm text-muted-foreground">Last updated: 26 July 2026</p>

      <div className="mt-8 space-y-8 text-sm leading-relaxed text-foreground">
        <p className="rounded-lg bg-accent p-4 text-muted-foreground">
          This is a draft prepared to reflect Pulse's actual data practices and the Nigeria Data
          Protection Act 2023 (NDPA) as it currently stands. It is not a substitute for review by
          a qualified Nigerian lawyer or data protection professional before being relied on as a
          binding policy — in particular, Pulse should confirm whether it meets the NDPA's
          threshold for a "data controller of major importance" (processing more than 2,000
          people's data in 12 months, or 1,000 in 6 months) and register with the Nigeria Data
          Protection Commission (NDPC) accordingly once it does.
        </p>

        <section>
          <h2 className="text-lg font-semibold">1. What this covers</h2>
          <p>
            The data controller for the purposes of this policy is Social Technologies (RC
            9706684), a company registered in Nigeria with its registered address at F.G.C.I
            SSQ17, Ogidi, Senior Staff Quarters, beside Girls Hostel, Ilorin, Kwara State,
            Nigeria, which operates the Pulse platform ("Pulse", "we", "us").
          </p>
          <p className="mt-3">
            This policy explains what personal data Pulse collects when you use the Platform,
            why, who we share it with, and the rights you have over it under the Nigeria Data
            Protection Act 2023. "Personal data" means any information relating to you as an
            identified or identifiable person.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">2. What we collect</h2>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            <li><strong>Account data:</strong> name, email address, password (stored hashed, never in plain text), and role (business or promoter).</li>
            <li><strong>Payout data:</strong> bank name, account number, and the account name returned by our payment processor when you add a bank account.</li>
            <li><strong>Social account data:</strong> the handle/URL of any Instagram or Twitter/X account you connect, and follower/engagement information used to calculate your influence score.</li>
            <li><strong>Campaign and submission data:</strong> campaign details you create as a business; repost links and screenshots you submit as a promoter.</li>
            <li><strong>Wallet and transaction data:</strong> top-ups, payouts, withdrawals, and balances.</li>
            <li><strong>Support and chat data:</strong> messages you send through in-app support or campaign-related chat, including to our AI support assistant.</li>
            <li><strong>Technical data:</strong> IP address and basic device/browser information, collected automatically for security and fraud-prevention purposes.</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-semibold">3. Why we process it, and our lawful basis</h2>
          <p>
            We process your data to: create and run your account; process campaign payments and
            promoter payouts; verify your bank and social accounts; detect fraud and enforce our
            Terms; respond to support requests; and comply with legal obligations. Our lawful
            basis under the NDPA is primarily performance of the contract we enter into with you
            when you sign up (our Terms and Conditions), with fraud-prevention and legal
            compliance handled under legitimate interest and legal obligation respectively.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">4. Who we share it with</h2>
          <p>We share the minimum data necessary with the third parties that help us run Pulse:</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            <li><strong>Paystack</strong> — processes top-ups, withdrawals, and payouts; receives your bank details and transaction amounts to do so.</li>
            <li><strong>Brevo</strong> — sends verification, password-reset, and other transactional emails; receives your name and email address.</li>
            <li><strong>Groq and Google (Gemini)</strong> — power the AI support assistant that may reply to your support messages; receive the content of the message you send to support, not your account or financial data.</li>
          </ul>
          <p className="mt-3">
            We do not sell your personal data. We may disclose data where required by law, to
            investigate suspected fraud, or in connection with a merger, acquisition, or sale of
            Pulse's business, in which case you'll be notified.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">5. International transfers</h2>
          <p>
            Some of the processors above may store or process data outside Nigeria. Where we
            transfer personal data outside Nigeria, we do so only where the destination offers a
            level of protection adequate to the NDPA, or under a lawful transfer mechanism such
            as the recipient's binding data protection commitments.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">6. How long we keep it</h2>
          <p>
            We keep account and transaction data for as long as your account is active. Once you
            close your account, we erase or anonymise your personal data within 30 days, except
            where we're required to retain it for longer to meet legal, tax, or
            fraud-investigation obligations (for example, transaction records relevant to a
            dispute or a regulatory request). Chat and support messages are kept for as long as
            needed to resolve your query and for quality/training purposes for our support
            systems, subject to the same 30-day erasure commitment once no longer needed.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">7. Your rights</h2>
          <p>Under the NDPA, you have the right to:</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            <li>Access the personal data we hold about you.</li>
            <li>Correct inaccurate or incomplete data.</li>
            <li>Request erasure of your data, subject to our legal/financial record-keeping obligations.</li>
            <li>Receive a copy of your data in a portable format.</li>
            <li>Object to processing based on legitimate interest, including certain automated decisions.</li>
          </ul>
          <p className="mt-3">
            To exercise any of these rights, contact us through Message Support in the app or at
            siyanbolagiggs@gmail.com . We'll respond within the timeframe required by
            the NDPA.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">8. Security and breach notification</h2>
          <p>
            We use technical and organisational safeguards — including hashed passwords and
            restricted access to financial data — to protect your data. If a breach occurs that's
            likely to affect your rights, we'll notify the Nigeria Data Protection Commission
            without undue delay (and in any case within 72 hours of becoming aware of it) and
            notify you directly where the breach poses a high risk to you.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">9. Changes to this policy</h2>
          <p>
            We may update this policy as Pulse evolves. We'll post the updated version here with
            a new "last updated" date, and for material changes we'll make reasonable efforts to
            notify you in-app before they take effect.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">10. Contact</h2>
          <p>Questions about this policy or your data can be sent through Message Support in the app, or to siyanbolagiggs@gmail.com</p>
        </section>
      </div>
    </main>
  );
}
