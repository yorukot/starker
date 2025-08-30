# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build & Run

- `pnpm dev` - Start development server with hot reload
- `pnpm build` - Create production build
- `pnpm preview` - Preview production build locally

### Code Quality

- `pnpm lint` - Run Prettier and ESLint checks
- `pnpm format` - Format code with Prettier
- `pnpm check` - Type check with svelte-check
- `pnpm check:watch` - Type check in watch mode

### Testing

- `pnpm test:ssh-key` - Run SSH key generation tests using tsx

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
  - `schemas/` - Validation schemas using Yup
  - `utils.ts` - Utility functions (cn class merger, TypeScript helpers)
  - `hooks/` - Svelte 5 runes and reactive utilities

### UI Component System

Uses shadcn-svelte component library with:

- TailwindCSS 4.0 with forms and typography plugins
- Lucide icons via unplugin-icons
- Component composition patterns with Svelte 5 snippets and runes
- Consistent styling through `cn()` utility combining clsx and tailwind-merge

### Authentication Flow

- Form validation using Yup with comprehensive password requirements
- Registration schema enforces strong passwords (uppercase, lowercase, numbers, 8+ chars)
- Components structured for login/register workflows

### Navigation Structure

- Hierarchical navigation: Teams → Projects → Services
- Sidebar with collapsible sections for Projects, Servers, Keys, Settings, Teams
- Team switcher component for multi-tenancy support

### Key Dependencies

- **Framework:** SvelteKit with Svelte 5
- **Styling:** TailwindCSS 4.0 with forms and typography plugins, shadcn-svelte components
- **Validation:** Yup for form schemas
- **Forms:** Felte with Yup validator integration
- **Icons:** Lucide icons via unplugin-icons with `~icons/` prefix
- **UI Components:** bits-ui, tailwind-variants, tailwind-merge, clsx
- **Development:** TypeScript, ESLint, Prettier with Svelte plugins, tw-animate-css
- **Cryptography:** node-forge and tweetnacl for SSH key generation (Ed25519 and RSA)
- **Code Editor:** CodeMirror 6 with YAML language support and One Dark theme
- **Avatars:** Dicebear collection for avatar generation
- **Notifications:** svelte-sonner for toast notifications
- **State Management:** mode-watcher for dark/light mode handling

### Configuration Files

- `components.json` - shadcn-svelte component registry configuration with slate base color
- `vite.config.ts` - Vite configuration with TailwindCSS 4.0, SvelteKit, and unplugin-icons
- `svelte.config.js` - SvelteKit configuration with auto adapter and alias setup
- `tsconfig.json` - TypeScript configuration extending SvelteKit defaults with unplugin-icons types
- `eslint.config.js` - ESLint configuration with TypeScript, Svelte, and Prettier integration

### Authentication System Architecture

- Token-based authentication with automatic refresh (5-minute intervals)
- JWT access tokens stored in sessionStorage with 15-minute expiration
- Refresh tokens handled via httpOnly cookies for security
- JWT validation via `isTokenValid()` and `refreshToken()` functions in `$lib/api/auth`
- Authenticated API client via `authFetch()` wrapper with automatic token refresh
- Automatic token refresh in dashboard layout on mount and periodic intervals
- Felte-compatible authenticated fetch function for form submissions
- Form submission with error handling for email conflicts and network errors

### Development Workflow

1. Install dependencies with `pnpm install` (preferred) or `npm install`
2. Start development server with `pnpm dev` or `npm run dev`
3. Access application at `http://localhost:5173`
4. Use `pnpm check` for type checking during development
5. Format code with `pnpm format` before committing
6. Run linting with `pnpm lint` to check Prettier and ESLint rules

### Component Patterns

- Use Svelte 5 runes (`$state`, `$derived`, `$props`) for reactivity
- Component composition with `{@render children?.()}`
- TypeScript prop typing with `ComponentProps<typeof Component>`
- Icon imports via `~icons/` prefix (e.g., `~icons/lucide/folder-open`)
- Utility types for component props: `WithoutChild`, `WithoutChildren`, `WithElementRef`
- Mobile responsive utilities via `IsMobile` class extending Svelte's `MediaQuery`

### SSH Key Management Architecture

- Client-side SSH key generation supporting Ed25519 and RSA key types
- Key size validation for RSA keys (2048, 3072, 4096 bits)
- SHA256 fingerprint generation for public keys
- OpenSSH private key format compliance with proper encoding
- Comment support for generated keys with validation (max 255 chars, no newlines)
- Type-safe error handling via `SSHKeyError` class with key type context

### Styling and Theme System

- TailwindCSS 4.0 with custom CSS variables and OKLCH color space
- Dark mode support with CSS variable overrides
- Custom radius system with sm/md/lg/xl variants
- Sidebar-specific theming variables for consistent navigation styling
- Animation utilities via `tw-animate-css` plugin

### Svelte Code Conventions

#### File Structure

- **Script Organization:** Use module script block (`<script lang="ts" module>`) for exports and shared logic, instance script block (`<script lang="ts">`) for component logic
- **TypeScript:** All components use TypeScript with strict typing
- **Script Block Order:** Module script first, then instance script, then markup

#### Import Conventions

- **UI Components:** Import from `$lib/components/ui/[component]/index.js`
- **Icons:** Use `~icons/` prefix (e.g., `~icons/lucide/folder-plus`, `~icons/lucide/chevrons-up-down`)
- **Type Imports:** Use `type` keyword for type-only imports
- **Schemas:** Import validation schemas from `$lib/schemas/`
- **API Utilities:** Import from `$lib/api/` for authentication and client functions

#### Component Patterns

- **Props:** Use `$props()` destructuring with TypeScript interfaces
- **State Management:** Use `$state()` for reactive state, `$derived()` for computed values
- **Refs:** Use `ref = $bindable(null)` for element references
- **Children:** Render with `{@render children?.()}` pattern
- **Conditional Rendering:** Use `#if` blocks with proper error state handling

#### Form Handling

- **Validation:** Use Felte with Yup validator integration via `@felte/validator-yup`
- **Form Creation:** `createForm<FormType>({ extend: validator({ schema }), onSubmit, onSuccess, onError })`
- **Error Display:** Use destructive styling with `border-destructive` class for invalid fields
- **Server Errors:** Handle with local `serverError` state and consistent error UI patterns
- **Submission State:** Use `$isSubmitting` for loading states and button disabling

#### Styling Guidelines

- **Class Utility:** Always use `cn()` function for conditional styling and class merging
- **Component Variants:** Use `tailwind-variants` for complex component styling systems
- **Error States:** Apply `border-destructive` class for form validation errors
- **Conditional Classes:** Use ternary operators within `cn()` for dynamic styling
- **Consistent Spacing:** Follow established gap and padding patterns from existing components

#### Type Safety

- **Component Props:** Use `WithElementRef` type for components that need element references
- **Type Exports:** Export component prop types from module script blocks
- **Interface Definitions:** Define clear TypeScript interfaces for all component props
- **Generic Types:** Use generics for reusable component patterns (e.g., form types)

#### State and Reactivity

- **Reactive State:** Use `$state()` for component-level reactive variables
- **Computed Values:** Use `$derived()` for values computed from other reactive state
- **Props Binding:** Use `$bindable()` for two-way binding on props
- **Stores Integration:** Import and use stores with proper reactivity patterns

#### Naming Conventions

- **Component Files:** Use kebab-case for file names (e.g., `team-switcher.svelte`)
- **Props:** Use camelCase for prop names
- **State Variables:** Use camelCase for state variables
- **CSS Classes:** Follow TailwindCSS naming conventions
- **Event Handlers:** Use descriptive names with action prefixes (e.g., `switchTeam`, `handleSubmit`)

# Important Instruction Reminders

## General Rules

Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary for achieving your goal.
ALWAYS prefer editing an existing file to creating a new one.
NEVER proactively create documentation files (\*.md) or README files. Only create documentation files if explicitly requested by the User.

## Coding Assistant Guidelines

You are my coding assistant. Follow these rules strictly:

I am using Svelte 5 together with shadcn-svelte.

For any knowledge related to Svelte, you must always check the official documentation first before giving me an answer.

If you encounter any problem or uncertainty, you must ask me questions instead of making assumptions or solving it on your own.

Always try to look at other pages or examples to see if there are similar implementations. If you find them, follow the same style and approach as those implementations.
