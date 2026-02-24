# Website Service

The public-facing marketing website for the InfraLayer platform built with modern web technologies.

## Service Purpose

This is the InfraLayer marketing website built with Astro 5, a modern static site generator. The site serves as the primary public interface for the InfraLayer platform, showcasing the AI SRE copilot capabilities and providing information about cloud infrastructure management solutions. The website includes marketing content, blog posts, and serves as the entry point for potential users to learn about and engage with the InfraLayer platform.

## Development Commands

```bash
# Development
npm run dev         # Start development server with host binding
npm run start      # Alternative dev command (astro dev only)

# Build and preview
npm run build      # Build for production
npm run preview    # Preview production build locally

# Direct Astro CLI access
npm run astro      # Run Astro CLI commands
```

## Architecture Overview

### Framework Stack
- **Astro 5**: Static site generator with island architecture
- **Alpine.js**: Lightweight reactive framework for interactivity  
- **Tailwind CSS**: Utility-first CSS framework
- **TypeScript**: Type safety throughout the codebase

### Content Management
- **Content Collections**: Blog posts organized by language (en/it) in `src/content/blog/`
- **Type-safe schema**: Content validation via Zod schemas in `src/content/config.ts`
- **Markdown processing**: Enhanced with rehype plugins for autolinked headings

### Key Features
- **PWA Support**: Progressive Web App capabilities via @vite-pwa/astro
- **SEO Optimization**: Meta tags, sitemaps, and OpenGraph integration
- **Internationalization**: Multi-language blog content structure
- **Performance**: Content visibility optimizations and view transitions

## Project Structure

```
src/
├── components/        # Reusable Astro components
│   ├── ui/           # Base UI components (button, link)
│   └── navbar/       # Navigation components
├── content/          # Content collections (blog posts)
├── layouts/          # Page layouts (Layout.astro)
├── pages/            # File-based routing
├── utils/            # Utility functions
└── assets/           # Static assets (images, icons)
```

## Path Aliases

The TypeScript configuration includes these path aliases:
- `@lib/*` → `src/lib/*`
- `@utils/*` → `src/utils/*`
- `@components/*` → `src/components/*`
- `@layouts/*` → `src/layouts/*`
- `@assets/*` → `src/assets/*`
- `@pages/*` → `src/pages/*`

## Styling and Design

- **Primary Color**: `#0023C4` (defined in tailwind.config.cjs)
- **Typography**: Inter Variable font for sans-serif text
- **Components**: Custom components built with Astro and styled with Tailwind
- **Responsive Design**: Mobile-first approach with Tailwind utilities

## Content Creation

Blog posts follow this frontmatter schema:
```yaml
draft: boolean
title: string
snippet: string
image:
  src: string
  alt: string
publishDate: string (ISO format)
author: string (defaults to "YourCompany")
category: string
tags: string[]
```

## Build Configuration

- **Site URL**: https://infralayer.dev
- **Output**: Static site generation
- **PWA**: Configured with auto-update registration
- **Markdown**: Enhanced with slug generation and autolinked headings

## Recent Development Learnings

### Security & Dependency Management
- **Vulnerability Resolution**: Resolved CVE-2025-5889 (ReDoS) in brace-expansion package
  - Used `npm update brace-expansion` to update from vulnerable 2.0.1 to secure 2.0.2
  - Verified fix with `npm audit` showing 0 vulnerabilities
  - Targeted updates more effective than removing package-lock.json for security patches

### GitHub Actions CI/CD Pipeline
- **Automated Deployment**: Configured GitHub Actions for Cloudflare Pages deployment
- **Build Process**: Astro build → Cloudflare Pages deployment on master branch pushes
- **Dependency Caching**: npm cache optimization for faster CI builds

### Website Enhancement Features
- **Development Status Indicator**: Added visual indicator in hero section showing project status
- **Vision Page**: Created dedicated `/vision` page with compelling content copy
- **Navigation Updates**: Enhanced navbar with Vision link and improved responsive design
- **Mobile Optimization**: Ensured responsive design across all new components

### Security Best Practices
- **Commit Message Standards**: Security fixes should include CVE references and detailed descriptions
- **Dependabot Integration**: Proactive monitoring and resolution of security alerts
- **Version Pinning**: Careful management of dependency versions in package-lock.json