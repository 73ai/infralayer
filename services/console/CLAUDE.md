# InfraLayer Console Service

Modern React TypeScript console service for AI-powered infrastructure management with comprehensive authentication, onboarding, and integration workflows.

## Service Overview

The console service serves as the primary user interface for InfraLayer platform, providing:
- **User Authentication & Authorization**: Clerk-based OAuth integration with organization management
- **Organization Onboarding**: Multi-step wizard for new organization setup and metadata collection
- **Integration Management**: Configuration and management of external service integrations (Slack, GitHub, AWS, GCP, Datadog, PagerDuty)
- **Skills Creation**: Interactive GitHub Actions workflow editor with real-time YAML validation
- **Infrastructure Management**: User interface for natural language to infrastructure command conversion

## Architecture

### Technology Stack
- **Framework**: Vite + React 18 + TypeScript 5.7
- **Styling**: Tailwind CSS 3.4 + shadcn/ui component library
- **State Management**: MobX 6.13 with singleton stores
- **Authentication**: Clerk React SDK 5.24
- **Forms**: React Hook Form 7.52 + Zod 3.23 validation
- **Routing**: React Router DOM 7.6
- **Code Editor**: CodeMirror 6 with YAML support
- **WASM Integration**: Custom actionlint integration for GitHub Actions validation
- **UI Components**: Radix UI primitives with custom design system

### Project Structure
```
src/
├── components/           # Reusable UI components
│   ├── ui/              # shadcn/ui base components
│   ├── onboarding/      # Multi-step onboarding flow
│   └── data-table/      # Table components with sorting/filtering
├── pages/               # Route-specific page components
│   ├── integrations/    # Integration management pages
│   └── skills/          # Skills creation interface
├── hooks/               # Custom React hooks
├── stores/              # MobX state management
├── services/            # API client services
├── lib/                 # Utility functions and constants
├── types/               # TypeScript type definitions
└── assets/              # Static assets (fonts, icons)
```

### State Management Architecture
- **MobX Stores**: Centralized reactive state management
  - `UserStore`: User profile and organization metadata
  - `IntegrationStore`: External service integration state
- **Local Component State**: React hooks for UI-specific state
- **Form State**: React Hook Form for complex form workflows

### Authentication Flow
1. Clerk handles OAuth authentication and organization management
2. `ProtectedRoute` component enforces authentication requirements
3. `useOnboardingGuard` hook manages onboarding completion status
4. Organization metadata drives feature access and configuration

## Development

### Prerequisites
- Node.js ≥ 22.0.0
- npm ≥ 11.0.0

### Environment Setup
Create `.env.local`:
```env
VITE_CLERK_PUBLISHABLE_KEY=pk_test_...
VITE_API_BASE_URL=http://localhost:8080
```

### Development Commands
```bash
# Install dependencies
npm install

# Start development server (localhost:5173)
npm run dev

# Type checking and build
npm run build

# Linting
npm run lint

# Preview production build
npm run preview
```

### Development Server Configuration
- **Host**: Configured to accept connections from `app-local.infralayer.dev`
- **Port**: 5173 (Vite default)
- **Hot Reload**: Enabled with React Fast Refresh
- **Build Output**: `../wwwroot/` (shared with Go server)

## Testing

### Testing Strategy
- **Unit Tests**: Jest + React Testing Library for hooks and utilities
- **Component Tests**: React Testing Library for component behavior
- **Integration Tests**: Playwright for end-to-end workflows
- **WASM Testing**: Mocked WASM modules for actionlint functionality

### Test Structure
```
src/hooks/__tests__/     # Hook unit tests
playwright/              # E2E test specifications (if configured)
```

### Running Tests
```bash
# Unit tests (if Jest configured)
npm test

# E2E tests (if Playwright configured)
npx playwright test

# Type checking
npx tsc --noEmit
```

### Test Patterns
- **Hook Testing**: Use `@testing-library/react-hooks` for custom hooks
- **Component Testing**: Focus on user interactions and accessibility
- **Store Testing**: Test MobX store actions and computed values in isolation
- **WASM Mocking**: Mock WebAssembly modules for actionlint integration

## Standards and Conventions

### Code Organization
- **Component Structure**: Single responsibility, composition over inheritance
- **Hook Naming**: Prefix custom hooks with `use` (e.g., `useActionlint`, `useOnboardingGuard`)
- **Store Naming**: Suffix with `Store` (e.g., `UserStore`, `IntegrationStore`)
- **Type Definitions**: Colocate with implementation or in dedicated `types/` directory

### TypeScript Standards
- **Strict Mode**: Enabled with experimental decorators for MobX
- **Path Aliases**: Use `@/` for src imports
- **Type Safety**: Prefer explicit types over `any`, use discriminated unions
- **Interface Naming**: Descriptive names without `I` prefix

### Styling Guidelines
- **Tailwind CSS**: Utility-first approach with semantic component classes
- **Design Tokens**: Use CSS custom properties for theming (HSL color space)
- **Component Variants**: Use `class-variance-authority` for component variants
- **Responsive Design**: Mobile-first approach with semantic breakpoints

### Component Standards
- **Props Interface**: Define explicit interfaces for all component props
- **Event Handling**: Use descriptive event handler names (`onSubmit`, `onCancel`)
- **Accessibility**: Include ARIA labels and keyboard navigation support
- **Error Boundaries**: Implement error boundaries for robust error handling

### State Management Patterns
- **MobX Actions**: Use `runInAction` for async state updates
- **Computed Values**: Leverage computed properties for derived state
- **Store Lifecycle**: Implement `reset()` methods for clean state transitions
- **Error Handling**: Centralized error state management with user-friendly messages

### Form Handling
- **React Hook Form**: Use for complex forms with validation
- **Zod Schemas**: Define validation schemas for type safety
- **Error Display**: Show field-level and form-level error messages
- **Loading States**: Indicate form submission progress

### Integration Patterns
- **API Services**: Centralized service layer for backend communication
- **Error Handling**: Consistent error handling with user feedback
- **Loading States**: Global and component-level loading indicators
- **Caching**: Local state caching for performance optimization

## Build and Deployment

### Build Configuration
- **Output Directory**: `../wwwroot/` (shared with Go backend)
- **Asset Optimization**: Vite handles bundling and optimization
- **Code Splitting**: Automatic route-based code splitting
- **TypeScript Compilation**: Project references for incremental builds

### Production Considerations
- **Environment Variables**: Use `VITE_` prefix for client-side variables
- **Bundle Analysis**: Monitor bundle size and performance
- **Static Assets**: Optimize images and fonts for web delivery
- **Security**: Configure CSP headers and secure cookie settings

### Deployment Process
1. Build console service: `npm run build`
2. Static files output to `../wwwroot/`
3. Go backend serves static files and handles API routes
4. CDN integration for asset delivery (if configured)

## WASM Integration

### Actionlint Integration
- **Purpose**: Real-time GitHub Actions workflow validation
- **Implementation**: Go compiled to WebAssembly
- **Loading**: Asynchronous WASM module loading with error handling
- **Caching**: Result caching for improved performance
- **Error Handling**: Graceful fallback when WASM unavailable

### WASM Development
- **Build Process**: Go source compiled to `actionlint.wasm`
- **Type Definitions**: TypeScript declarations for WASM exports
- **Testing**: Mock WASM modules for unit testing
- **Performance**: Debounced validation to minimize computation

## Security Considerations

### Authentication Security
- **Clerk Integration**: Secure token handling and automatic refresh
- **Route Protection**: All authenticated routes require valid session
- **Organization Isolation**: Multi-tenant data isolation
- **CSRF Protection**: Built-in protection with modern frameworks

### Data Handling
- **Input Validation**: Client and server-side validation
- **Sanitization**: Prevent XSS attacks in user-generated content
- **Local Storage**: Secure handling of client-side data
- **API Communication**: HTTPS-only communication with backend

### Development Security
- **Dependency Scanning**: Regular security audits of npm packages
- **Code Linting**: ESLint rules for security best practices
- **Environment Variables**: Secure handling of sensitive configuration
- **WASM Security**: Sandboxed execution of WebAssembly modules

## Performance Optimization

### Bundle Optimization
- **Code Splitting**: Route-based and component-based splitting
- **Tree Shaking**: Remove unused code from final bundle
- **Asset Optimization**: Compress images and optimize fonts
- **Lazy Loading**: Load components and routes on demand

### Runtime Performance
- **React Optimization**: Use React.memo and useMemo for expensive operations
- **MobX Optimization**: Minimize observable object creation
- **WASM Performance**: Cache validation results and debounce input
- **API Optimization**: Request deduplication and caching

### Monitoring
- **Performance Metrics**: Monitor Core Web Vitals
- **Bundle Analysis**: Regular bundle size monitoring
- **Error Tracking**: Client-side error reporting
- **User Experience**: Track user interaction patterns