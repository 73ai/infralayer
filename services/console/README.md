# InfraLayer Console Service

React TypeScript console service for AI-powered infrastructure management with Clerk authentication and integration management.

## Quick Start

```bash
npm install                 # Install dependencies
npm run dev                # Start development server (localhost:5173)
npm run build              # Build for production
npm run lint               # Run ESLint
```

## Environment Setup

Create `.env.local`:
```env
VITE_CLERK_PUBLISHABLE_KEY=pk_test_...
VITE_API_BASE_URL=http://localhost:8080
```

## Key Features

- **Authentication**: Clerk OAuth integration
- **Organization Onboarding**: Multi-step setup workflow
- **Integration Management**: External service connections (Slack, GitHub, AWS, GCP, Datadog, PagerDuty)
- **UI**: Radix UI components with Tailwind CSS
- **State Management**: MobX stores

## Tech Stack

- Vite + React + TypeScript
- Tailwind CSS + shadcn/ui
- React Hook Form + Zod validation
- MobX for state management
- Clerk for authentication