// TODO: Integrate Stripe Customer Portal
// When ready:
// 1. Add stripe package: npm install stripe
// 2. Create API route at app/api/billing/portal/route.ts
//    that calls stripe.billingPortal.sessions.create({ customer: orgStripeId, return_url })
// 3. Replace the href="#" button below with a form POST to that route

export default function BillingPage() {
  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-100">Billing</h1>
        <p className="mt-1 text-sm text-gray-400">
          Manage your subscription and payment methods
        </p>
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900 p-8 text-center max-w-md mx-auto mt-12">
        <div className="mb-4 flex items-center justify-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-gray-800">
            <svg
              className="h-6 w-6 text-[#52b788]"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"
              />
            </svg>
          </div>
        </div>

        <h2 className="text-lg font-semibold text-gray-100">
          Manage Billing via Stripe
        </h2>
        <p className="mt-2 text-sm text-gray-400">
          View invoices, update your payment method, and manage your
          subscription through the Stripe Customer Portal.
        </p>

        <a
          href="#"
          className="mt-6 inline-flex items-center gap-2 rounded-md bg-[#52b788] px-5 py-2.5 text-sm font-medium text-gray-950 hover:bg-[#3d9a6f]"
          onClick={(e) => e.preventDefault()}
        >
          Open Stripe Portal
          <svg
            className="h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
            />
          </svg>
        </a>
        <p className="mt-3 text-xs text-gray-600">
          Stripe integration coming soon
        </p>
      </div>
    </div>
  );
}
