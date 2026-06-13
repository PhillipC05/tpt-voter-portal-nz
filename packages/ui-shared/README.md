# @tpt-nz/ui-shared

Shared React component library for all TPT NZ civic applications. Built with tsup (ESM + CJS + TypeScript declarations).

---

## Installation

### Within this monorepo (pnpm workspaces)

```jsonc
// In your app's package.json:
{
  "dependencies": {
    "@tpt-nz/ui-shared": "workspace:*"
  }
}
```

pnpm resolves `workspace:*` to the local package — no npm hit required during development.

### From npm (external projects)

The package is published to npm as `@tpt-nz/ui-shared` on each version tag via the GitHub Actions release workflow.

```bash
npm install @tpt-nz/ui-shared
# or
pnpm add @tpt-nz/ui-shared
```

---

## Components

### `RealMeButton`

DIA brand-compliant "Login with RealMe" button. Required for any RealMe integration.

```tsx
import { RealMeButton } from '@tpt-nz/ui-shared'

// Basic login
<RealMeButton />

// Verified Identity
<RealMeButton level="verified" returnUrl="/incorporate" />
```

Props: `loginUrl`, `level` (`'login' | 'verified'`), `returnUrl`, `className`

### `VerifiedBadge`

Colour-coded assurance level chip.

```tsx
import { VerifiedBadge } from '@tpt-nz/ui-shared'

<VerifiedBadge level="verified" />  // Green — "RealMe Verified"
<VerifiedBadge level="login" />     // Blue  — "RealMe Login"
<VerifiedBadge level="none" />      // Grey  — "Unverified"
```

### `IdentityCard`

Displays verified name and address. Only shows identity fields when `assuranceLevel === 'verified'`.

```tsx
import { IdentityCard } from '@tpt-nz/ui-shared'

<IdentityCard
  fullName="Jane Smith"
  assuranceLevel="verified"
  address={{ unit: '', number: '1', street: 'Queen Street', city: 'Auckland', postcode: '1010' }}
/>
```

### `ConsentNotice`

Privacy Act 2020 Principle 3 compliant consent gate. Requires checkbox acknowledgement before redirecting to RealMe.

```tsx
import { ConsentNotice } from '@tpt-nz/ui-shared'

<ConsentNotice
  serviceName="Business Incorporation Service"
  dataRequested={['Legal full name', 'Date of birth', 'Residential address']}
  purpose="To verify your identity before filing with the Companies Register."
  onAccept={() => router.push('/auth/realme/login')}
  onDecline={() => router.back()}
/>
```

### `QRCode`

Canvas-based QR code renderer.

```tsx
import { QRCode } from '@tpt-nz/ui-shared'

<QRCode value="https://credentials.tpt.nz/verify/abc123" size={200} />
```

---

## Development

```bash
# Build
pnpm --filter @tpt-nz/ui-shared build

# Watch mode
pnpm --filter @tpt-nz/ui-shared dev

# Tests
pnpm --filter @tpt-nz/ui-shared test

# Type check
pnpm --filter @tpt-nz/ui-shared typecheck
```

## Styles

Import `globals.css` in your app's root layout to get the NZ Government Design System custom properties and Tailwind base styles:

```tsx
import '@tpt-nz/ui-shared/src/styles/globals.css'
```

The `tailwind.config.ts` export the NZ Government brand colour palette as Tailwind tokens under the `nzgds` key (e.g. `text-nzgds-blue`, `bg-nzgds-green`).
