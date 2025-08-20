<script lang="ts">
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import { useSidebar } from "$lib/components/ui/sidebar/index.js";
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { invalidate } from '$app/navigation';
	import ChevronsUpDownIcon from "~icons/lucide/chevrons-up-down";
	import LucidePlus from '~icons/lucide/plus';
	import type { Team } from '$lib/schemas/team';
	import { generateTeamAvatar } from '$lib/utils/avatar';

	let { teams, currentTeam }: { teams: Team[]; currentTeam: Team | null } = $props();
	const sidebar = useSidebar();

	const activeTeam = $derived(currentTeam || teams[0] || null);

	async function switchTeam(team: Team) {
		const currentPath = page.url.pathname;
		const newPath = currentPath.replace(/^\/dashboard\/[^/]+/, `/dashboard/${team.id}`);
		await goto(newPath);
		await invalidate('team:current');
	}
</script>

<Sidebar.Menu>
	<Sidebar.MenuItem>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Sidebar.MenuButton
						{...props}
						size="lg"
						class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
					>
						{#if activeTeam}
							<img
								src={generateTeamAvatar(activeTeam.id, 32)}
								alt={activeTeam.name}
								class="aspect-square size-8 rounded-lg"
							/>
							<div class="grid flex-1 text-left text-sm leading-tight">
								<span class="truncate font-medium">
									{activeTeam.name}
								</span>
								<span class="truncate text-xs text-muted-foreground">Team</span>
							</div>
						{:else}
							<div class="grid flex-1 text-left text-sm leading-tight">
								<span class="truncate font-medium">No team selected</span>
							</div>
						{/if}
						<ChevronsUpDownIcon class="ml-auto" />
					</Sidebar.MenuButton>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content
				class="w-(--bits-dropdown-menu-anchor-width) min-w-56 rounded-lg"
				align="start"
				side={sidebar.isMobile ? "bottom" : "right"}
				sideOffset={4}
			>
				<DropdownMenu.Label class="text-muted-foreground text-xs">Teams</DropdownMenu.Label>
				{#each teams as team (team.id)}
					<DropdownMenu.Item onSelect={() => switchTeam(team)} class="gap-2 p-2">
						<img
							src={generateTeamAvatar(team.id, 24)}
							alt={team.name}
							class="size-6 rounded-md border"
						/>
						{team.name}
					</DropdownMenu.Item>
				{/each}
				<DropdownMenu.Separator />
				<DropdownMenu.Item class="gap-2 p-2">
					<div
						class="flex size-6 items-center justify-center rounded-md border bg-transparent"
					>
						<LucidePlus class="size-4" />
					</div>
					<div class="text-muted-foreground font-medium">Add team</div>
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.MenuItem>
</Sidebar.Menu>
