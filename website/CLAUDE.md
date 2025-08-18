# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build & Run
- `npm run dev` or `pnpm dev` - Start development server with hot reload
- `npm run build` or `pnpm build` - Create production build
- `npm run preview` or `pnpm preview` - Preview production build locally

### Code Quality
- `npm run lint` or `pnpm lint` - Run Prettier and ESLint checks
- `npm run format` or `pnpm format` - Format code with Prettier
- `npm run check` or `pnpm check` - Type check with svelte-check
- `npm run check:watch` or `pnpm check:watch` - Type check in watch mode

## Project Architecture

This is a SvelteKit website using Svelte 5 with TypeScript, TailwindCSS 4.0, and shadcn-svelte components.

### Core Structure
- `src/routes/` - SvelteKit file-based routing
  - `+page.svelte` - Page components
  - `+layout.svelte` - Layout components
  - `auth/` - Authentication pages (login, register)
  - `dashboard/` - Main application dashboard with sidebar layout
- `src/lib/` - Shared library code
  - `components/` - Reusable Svelte components
    - `ui/` - shadcn-svelte UI primitives (button, card, sidebar, etc.)
    - `auth/` - Authentication-specific components
    - `sidebar/` - Navigation components with team/project structure
  - `schemas/` - Validation schemas using Valibot
  - `utils.ts` - Utility functions (cn class merger, TypeScript helpers)
  - `hooks/` - Svelte 5 runes and reactive utilities

### UI Component System
Uses shadcn-svelte component library with:
- TailwindCSS 4.0 with forms and typography plugins
- Lucide icons via unplugin-icons
- Component composition patterns with Svelte 5 snippets and runes
- Consistent styling through `cn()` utility combining clsx and tailwind-merge

### Authentication Flow
- Form validation using Valibot with comprehensive password requirements
- Registration schema enforces strong passwords (uppercase, lowercase, numbers, 8+ chars)
- Components structured for login/register workflows

### Navigation Structure
- Hierarchical navigation: Teams → Projects → Services
- Sidebar with collapsible sections for Projects, Servers, Keys, Settings, Teams
- Team switcher component for multi-tenancy support

### Key Dependencies
- **Framework:** SvelteKit with Svelte 5
- **Styling:** TailwindCSS 4.0, shadcn-svelte components
- **Validation:** Valibot for form schemas, Zod also available
- **Forms:** sveltekit-superforms for enhanced form handling
- **Icons:** Lucide icons via unplugin-icons
- **Development:** TypeScript, ESLint, Prettier with Svelte plugins

### Development Workflow
1. Install dependencies with `pnpm install`
2. Start development server with `pnpm dev`
3. Access application at `http://localhost:5173`
4. Use `pnpm check` for type checking during development
5. Format code with `pnpm format` before committing

### Component Patterns
- Use Svelte 5 runes (`$state`, `$derived`, `$props`) for reactivity
- Component composition with `{@render children?.()}`
- TypeScript prop typing with `ComponentProps<typeof Component>`
- Icon imports via `~icons/` prefix (e.g., `~icons/lucide/folder-open`)