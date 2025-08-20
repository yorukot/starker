<script lang="ts">
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';

	let {
		items
	}: {
		items: {
			title: string;
			url: string;
			// this should be `Component` after @lucide/svelte updates types
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			icon?: any;
			isActive?: boolean;
		}[];
	} = $props();
</script>

<Sidebar.Group>
	<Sidebar.Menu>
		{#each items as item (item.title)}
			<Sidebar.MenuItem>
				<a href={item.url} class="flex items-center gap-2">
					<Sidebar.MenuButton isActive={item.isActive} class="cursor-pointer">
						{#if item.icon}
							<item.icon class="h-4 w-4" />
						{/if}
						{item.title}
					</Sidebar.MenuButton>
				</a>
			</Sidebar.MenuItem>
		{/each}
	</Sidebar.Menu>
</Sidebar.Group>
