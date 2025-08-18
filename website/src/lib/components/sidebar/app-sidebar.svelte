<script lang="ts" module>
	import LucideFolderOpen from '~icons/lucide/folder-open';
	import LucideServer from '~icons/lucide/server';
	import LucideSettings from '~icons/lucide/settings';
	import LucideUsers from '~icons/lucide/users';
	import LucideKeyRound from '~icons/lucide/key-round';
	
	// This is sample data.
	const data = {
		user: {
			name: "shadcn",
			email: "m@example.com",
			avatar: "/avatars/shadcn.jpg",
		},
		teams: [
			{
				name: "Acme Inc",
				logo: LucideFolderOpen,
				plan: "Enterprise",
			},
			{
				name: "Acme Corp.",
				logo: LucideFolderOpen,
				plan: "Startup",
			},
		],
		navMain: [
			{
				title: "Projects",
				url: "#",
				icon: LucideFolderOpen,
				isActive: true,
			},
			{
				title: "Servers",
				url: "#",
				icon: LucideServer,
			},
			{
				title: "Keys",
				url: "#",
				icon: LucideKeyRound,
			},
			{
				title: "Settings",
				url: "#",
				icon: LucideSettings,				
			},
			{
			  title: "Teams",
				url: "#",
				icon: LucideUsers,				
			}
		],
	};
</script>

<script lang="ts">
	import NavMain from "./nav-main.svelte";
	import NavUser from "./nav-user.svelte";
	import TeamSwitcher from "./team-switcher.svelte";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import type { ComponentProps } from "svelte";

	let {
		ref = $bindable(null),
		collapsible = "icon",
		...restProps
	}: ComponentProps<typeof Sidebar.Root> = $props();
</script>

<Sidebar.Root {collapsible} {...restProps}>
	<Sidebar.Header>
		<TeamSwitcher teams={data.teams} />
	</Sidebar.Header>
	<Sidebar.Content>
		<NavMain items={data.navMain} />
	</Sidebar.Content>
	<Sidebar.Footer>
		<NavUser user={data.user} />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
