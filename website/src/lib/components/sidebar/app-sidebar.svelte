<script lang="ts">
	import LucideFolderOpen from '~icons/lucide/folder-open';
	import LucideServer from '~icons/lucide/server';
	import LucideSettings from '~icons/lucide/settings';
	import LucideUsers from '~icons/lucide/users';
	import LucideKeyRound from '~icons/lucide/key-round';
	import { page } from '$app/state';

	import NavMain from './nav-main.svelte';
	import NavUser from './nav-user.svelte';
	import TeamSwitcher from './team-switcher.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import type { ComponentProps } from 'svelte';
	import type { Team } from '$lib/schemas/team';

	let {
		ref = $bindable(null),
		collapsible = 'icon',
		teams = [],
		currentTeam = null,
		...restProps
	}: ComponentProps<typeof Sidebar.Root> & {
		teams: Team[];
		currentTeam: Team | null;
	} = $props();

	const teamID = $derived(page.params.teamID)

	// This is sample data.
	const data = $derived({
		user: {
			name: 'shadcn',
			email: 'm@example.com',
			avatar: '/avatars/shadcn.jpg'
		},
		navMain: [
			{
				title: 'Projects',
				url: `/dashboard/${teamID}/projects`,
				icon: LucideFolderOpen,
				isActive: page.url.pathname.includes(`projects`)
			},
			{
				title: 'Servers',
				url: `/dashboard/${teamID}/servers`,
				icon: LucideServer,
				isActive: page.url.pathname.includes(`servers`)
			},
			{
				title: 'Keys',
				url: `/dashboard/${teamID}/keys`,
				icon: LucideKeyRound,
				isActive: page.url.pathname.includes(`keys`)
			},
			{
				title: 'Settings',
				url: `/dashboard/${teamID}/settings`,
				icon: LucideSettings,
				isActive: page.url.pathname.includes(`settings`)
			},
			{
				title: 'Teams',
				url: `/dashboard/${teamID}/teams`,
				icon: LucideUsers,
				isActive: page.url.pathname.includes(`teams`)
			}
		]
	});
</script>

<Sidebar.Root {collapsible} {...restProps}>
	<Sidebar.Header>
		<TeamSwitcher {teams} {currentTeam} />
	</Sidebar.Header>
	<Sidebar.Content>
		<NavMain items={data.navMain} />
	</Sidebar.Content>
	<Sidebar.Footer>
		<NavUser user={data.user} />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
