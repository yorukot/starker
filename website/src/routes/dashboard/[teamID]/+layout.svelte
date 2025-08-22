<script lang="ts">
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import AppSidebar from '$lib/components/sidebar/app-sidebar.svelte';
	import type { Team } from '$lib/schemas/team';
	import type { Snippet } from 'svelte';

	interface LayoutData {
		teams: Team[];
		currentTeam: Team | null;
	}

	let { children, data }: { children: Snippet; data: LayoutData } = $props();

	const teams = $derived(data.teams);
	const currentTeam = $derived(data.currentTeam);
</script>

<Sidebar.Provider>
	<AppSidebar {teams} {currentTeam} />
	<main class="h-full w-full">
		<div class="flex min-h-[80vh] w-full justify-center p-6 md:p-8">
			<div class="w-full max-w-6xl">
				{@render children?.()}
			</div>
		</div>
	</main>
</Sidebar.Provider>
