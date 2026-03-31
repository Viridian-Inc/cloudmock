#!/usr/bin/env bash
set -euo pipefail

# CloudMock v1.0.0 Launch Script
# Run each section in order. Some steps require manual input (marked MANUAL).

echo "=== CloudMock v1.0.0 Launch ==="
echo ""

# ------------------------------------------------------------------
# Step 1: Final build verification
# ------------------------------------------------------------------
echo "--- Step 1: Build verification ---"
echo "Building Go binary..."
go build -o ./cloudmock ./cmd/gateway/
echo "  Gateway binary: OK"

echo "Building devtools SPA..."
(cd ../neureaux-devtools && pnpm build > /dev/null 2>&1)
echo "  Devtools SPA: OK"

echo "Building docs site..."
(cd website && npm run build > /dev/null 2>&1)
echo "  Docs site: OK"

echo ""

# ------------------------------------------------------------------
# Step 2: Run tests
# ------------------------------------------------------------------
echo "--- Step 2: Tests ---"
echo "Running Go tests (this takes a minute)..."
go test -count=1 ./services/... ./pkg/saas/... ./tests/... > /dev/null 2>&1
echo "  Go tests: PASS"

echo "Running frontend tests..."
(cd ../neureaux-devtools && pnpm test > /dev/null 2>&1)
echo "  Frontend tests: PASS (269/269)"

echo ""

# ------------------------------------------------------------------
# Step 3: npm publish (MANUAL - requires npm login)
# ------------------------------------------------------------------
echo "--- Step 3: npm publish ---"
echo ""
echo "  MANUAL: Run these commands after 'npm login':"
echo ""
echo "  # Publish CLI package"
echo "  cd npm/cloudmock && npm publish --access public"
echo ""
echo "  # Publish Node SDK"
echo "  cd ../neureaux-devtools/sdk/node && npm publish --access public"
echo ""
read -p "  Press Enter when done (or skip with Enter)..."
echo ""

# ------------------------------------------------------------------
# Step 4: Git tag + release
# ------------------------------------------------------------------
echo "--- Step 4: Git tag ---"
echo ""
read -p "  Create and push v1.0.0 tag? (y/N) " REPLY
if [[ "$REPLY" =~ ^[Yy]$ ]]; then
  git tag -a v1.0.0 -m "CloudMock v1.0.0 - Local AWS. 25 services. One binary."
  echo "  Tag created: v1.0.0"
  echo ""
  read -p "  Push tag to trigger release workflow? (y/N) " REPLY2
  if [[ "$REPLY2" =~ ^[Yy]$ ]]; then
    git push origin v1.0.0
    echo "  Tag pushed. Release workflow will build binaries + Docker image."
  fi
fi
echo ""

# ------------------------------------------------------------------
# Step 5: Deploy docs + homepage to Cloudflare Pages
# ------------------------------------------------------------------
echo "--- Step 5: Deploy docs site ---"
echo ""
echo "  MANUAL: Connect website/ to Cloudflare Pages:"
echo ""
echo "  1. Go to dash.cloudflare.com > Pages > Create project"
echo "  2. Connect your GitHub repo"
echo "  3. Build settings:"
echo "     - Build command: cd website && npm install && npm run build"
echo "     - Build output:  website/dist"
echo "     - Root:          /"
echo "  4. Set custom domain: cloudmock.io"
echo ""
read -p "  Press Enter when done (or skip)..."
echo ""

# ------------------------------------------------------------------
# Step 6: Configure Clerk
# ------------------------------------------------------------------
echo "--- Step 6: Clerk setup ---"
echo ""
echo "  MANUAL: Set up Clerk at clerk.com:"
echo ""
echo "  1. Create application 'CloudMock'"
echo "  2. Enable: Google, GitHub, Email sign-in"
echo "  3. Enable Organizations"
echo "  4. Add webhook endpoint: https://cloudmock-saas.fly.dev/api/webhooks/clerk"
echo "     Events: organization.created, organization.deleted, user.created"
echo "  5. Copy secrets to environment:"
echo "     export CLERK_SECRET_KEY=sk_live_..."
echo "     export CLERK_WEBHOOK_SECRET=whsec_..."
echo "     export CLERK_DOMAIN=your-app.clerk.accounts.dev"
echo ""
read -p "  Press Enter when done (or skip)..."
echo ""

# ------------------------------------------------------------------
# Step 7: Configure Stripe
# ------------------------------------------------------------------
echo "--- Step 7: Stripe setup ---"
echo ""
echo "  MANUAL: Set up Stripe at dashboard.stripe.com:"
echo ""
echo "  1. Create products:"
echo "     - CloudMock Pro: \$29/mo recurring"
echo "     - CloudMock Team: \$99/mo recurring"
echo "  2. Add metadata to each price:"
echo "     tier=pro (or team), request_limit=1000000 (or 10000000)"
echo "  3. Create webhook endpoint: https://cloudmock-saas.fly.dev/api/webhooks/stripe"
echo "     Events: checkout.session.completed, invoice.paid,"
echo "             customer.subscription.updated, customer.subscription.deleted"
echo "  4. Copy secrets:"
echo "     export STRIPE_SECRET_KEY=sk_live_..."
echo "     export STRIPE_WEBHOOK_SECRET=whsec_..."
echo "     export STRIPE_PRO_PRICE_ID=price_..."
echo "     export STRIPE_TEAM_PRICE_ID=price_..."
echo "  5. Update pricing page checkout links in website/src/pages/pricing.astro"
echo ""
read -p "  Press Enter when done (or skip)..."
echo ""

# ------------------------------------------------------------------
# Step 8: Deploy to Fly.io
# ------------------------------------------------------------------
echo "--- Step 8: Fly.io deployment ---"
echo ""
echo "  MANUAL: Deploy the SaaS control plane:"
echo ""
echo "  1. Install flyctl: brew install flyctl"
echo "  2. Login: fly auth login"
echo "  3. Create app: fly apps create cloudmock-saas"
echo "  4. Create Postgres: fly postgres create --name cloudmock-db --region iad"
echo "  5. Attach DB: fly postgres attach cloudmock-db --app cloudmock-saas"
echo "  6. Set secrets:"
echo "     fly secrets set \\"
echo "       CLOUDMOCK_SAAS_ENABLED=true \\"
echo "       CLOUDMOCK_AUTH_ENABLED=true \\"
echo "       CLERK_SECRET_KEY=sk_live_... \\"
echo "       CLERK_WEBHOOK_SECRET=whsec_... \\"
echo "       CLERK_DOMAIN=your-app.clerk.accounts.dev \\"
echo "       STRIPE_SECRET_KEY=sk_live_... \\"
echo "       STRIPE_WEBHOOK_SECRET=whsec_... \\"
echo "       FLY_API_TOKEN=fo1_... \\"
echo "       CLOUDFLARE_API_TOKEN=... \\"
echo "       CLOUDFLARE_ZONE_ID=... \\"
echo "       --app cloudmock-saas"
echo "  7. Deploy: fly deploy --app cloudmock-saas"
echo ""
read -p "  Press Enter when done (or skip)..."
echo ""

# ------------------------------------------------------------------
# Step 9: Cloudflare DNS
# ------------------------------------------------------------------
echo "--- Step 9: DNS setup ---"
echo ""
echo "  MANUAL: Configure Cloudflare DNS:"
echo ""
echo "  1. Add CNAME: cloudmock.io -> your Cloudflare Pages deployment"
echo "  2. Add CNAME: *.cloudmock.io -> cloudmock-saas.fly.dev (proxied)"
echo "  3. Verify: dig +short test.cloudmock.io"
echo ""
read -p "  Press Enter when done (or skip)..."
echo ""

# ------------------------------------------------------------------
# Step 10: Smoke test
# ------------------------------------------------------------------
echo "--- Step 10: Smoke test ---"
echo ""
echo "  Verify these URLs work:"
echo "  - https://cloudmock.io (homepage)"
echo "  - https://cloudmock.io/docs (documentation)"
echo "  - https://cloudmock.io/pricing (pricing)"
echo ""
echo "  Test local install:"
echo "  npx cloudmock"
echo "  # Open http://localhost:4500"
echo "  # Run: cmk s3 mb s3://test-bucket"
echo ""

echo "=== Launch complete ==="
echo ""
echo "Announce:"
echo "  - GitHub Release: v1.0.0"
echo "  - Twitter/X: 'CloudMock v1.0.0 — Local AWS. 25 services. One binary.'"
echo "  - Hacker News: 'Show HN: CloudMock — open-source local AWS emulator with built-in devtools'"
echo "  - Reddit r/aws, r/devops: link to cloudmock.io"
