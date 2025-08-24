<script lang="ts">
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import { useSidebar } from '$lib/components/ui/sidebar/index.js';
	import { generateUserAvatar } from '$lib/utils/avatar';
	import BadgeCheckIcon from '~icons/lucide/badge-check';
	import LucideBell from '~icons/lucide/bell';
	import LucideChevronsUpDown from '~icons/lucide/chevrons-up-down';
	import LucideCreditCard from '~icons/lucide/credit-card';
	import LucideLogOut from '~icons/lucide/log-out';
	import LucideSparkles from '~icons/lucide/sparkles';
	import type { User } from '$lib/schemas/user';

	let { user }: { user: User | null } = $props();
	const sidebar = useSidebar();

	const avatarSrc = $derived(
		user?.avatar || (user?.display_name ? generateUserAvatar(user.id) : generateUserAvatar('User'))
	);
	const displayName = $derived(user?.display_name || 'User');
	const fallbackText = $derived(
		displayName
			.split(' ')
			.map((name) => name[0])
			.join('')
			.substring(0, 2)
			.toUpperCase()
	);
</script>

<Sidebar.Menu>
	<Sidebar.MenuItem>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Sidebar.MenuButton
						size="lg"
						class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
						{...props}
					>
						<Avatar.Root class="size-8 rounded-lg">
							<Avatar.Image src={avatarSrc} alt={displayName} />
							<Avatar.Fallback class="rounded-lg">{fallbackText}</Avatar.Fallback>
						</Avatar.Root>
						<div class="grid flex-1 text-left text-sm leading-tight">
							<span class="truncate font-medium">{displayName}</span>
						</div>
						<LucideChevronsUpDown class="ml-auto size-4" />
					</Sidebar.MenuButton>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content
				class="w-(--bits-dropdown-menu-anchor-width) min-w-56 rounded-lg"
				side={sidebar.isMobile ? 'bottom' : 'right'}
				align="end"
				sideOffset={4}
			>
				<DropdownMenu.Label class="p-0 font-normal">
					<div class="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
						<Avatar.Root class="size-8 rounded-lg">
							<Avatar.Image src={avatarSrc} alt={displayName} />
							<Avatar.Fallback class="rounded-lg">{fallbackText}</Avatar.Fallback>
						</Avatar.Root>
						<div class="grid flex-1 text-left text-sm leading-tight">
							<span class="truncate font-medium">{displayName}</span>
						</div>
					</div>
				</DropdownMenu.Label>
				<DropdownMenu.Separator />
				<DropdownMenu.Group>
					<DropdownMenu.Item>
						<LucideSparkles />
						Upgrade to Pro
					</DropdownMenu.Item>
				</DropdownMenu.Group>
				<DropdownMenu.Separator />
				<DropdownMenu.Group>
					<DropdownMenu.Item>
						<BadgeCheckIcon />
						Account
					</DropdownMenu.Item>
					<DropdownMenu.Item>
						<LucideCreditCard />
						Billing
					</DropdownMenu.Item>
					<DropdownMenu.Item>
						<LucideBell />
						Notifications
					</DropdownMenu.Item>
				</DropdownMenu.Group>
				<DropdownMenu.Separator />
				<DropdownMenu.Item>
					<LucideLogOut />
					Log out
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.MenuItem>
</Sidebar.Menu>
