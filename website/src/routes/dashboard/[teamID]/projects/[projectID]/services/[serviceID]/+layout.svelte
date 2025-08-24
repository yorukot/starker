<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { setContext } from 'svelte';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import type { Snippet } from 'svelte';
	import type { Team } from '$lib/schemas/team';
	import type { Project } from '$lib/schemas/project';
	import type { Service } from '$lib/schemas/service';

	interface LayoutData {
		team: Team | null;
		project: Project | null;
		service: Service | null;
	}

	let { children, data }: { children: Snippet; data: LayoutData } = $props();
	
	// Set context so child components can access layout data
	setContext('layoutData', data);

	// Extract route parameters
	const teamID = $derived(page.params.teamID);
	const projectID = $derived(page.params.projectID);
	const serviceID = $derived(page.params.serviceID);

	// Navigation tabs configuration
	const tabs = [
		{ id: 'overview', label: 'Overview', path: 'overview' },
		{ id: 'domains', label: 'Domains', path: 'domains' },
		{ id: 'logs', label: 'Logs', path: 'logs' },
		{ id: 'settings', label: 'Settings', path: 'settings' }
	];

	// Determine active tab based on current route
	const activeTab = $derived(() => {
		const pathname = page.url.pathname;
		const segments = pathname.split('/');
		const lastSegment = segments[segments.length - 1];

		// Find matching tab by path
		const matchingTab = tabs.find((tab) => tab.path === lastSegment);
		return matchingTab ? matchingTab.id : 'overview';
	});

	// Handle tab navigation
	function handleTabChange(tabId: string) {
		const tab = tabs.find((t) => t.id === tabId);
		if (tab) {
			const newPath = `/dashboard/${teamID}/projects/${projectID}/${serviceID}/compose/${tab.path}`;
			goto(newPath);
		}
	}

	// Navigation functions
	function navigateTo(path: string) {
		goto(path);
	}
</script>

<style>
	.scrollbar-hide {
		-ms-overflow-style: none;  /* IE and Edge */
		scrollbar-width: none;  /* Firefox */
	}
	.scrollbar-hide::-webkit-scrollbar {
		display: none;  /* Chrome, Safari, Opera */
	}
</style>

<div class="flex h-full w-full flex-col">
	<!-- Header Section -->
	<div class="px-6 py-4">
		<Breadcrumb.Root>
			<Breadcrumb.List>
				<Breadcrumb.Item>
					<Breadcrumb.Link onclick={() => navigateTo(`/dashboard/${teamID}`)}>
						{data?.team?.name || 'Dashboard'}
					</Breadcrumb.Link>
				</Breadcrumb.Item>
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Link onclick={() => navigateTo(`/dashboard/${teamID}/projects`)}>
						Projects
					</Breadcrumb.Link>
				</Breadcrumb.Item>
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Link onclick={() => navigateTo(`/dashboard/${teamID}/projects/${projectID}`)}>
						{data?.project?.name || projectID}
					</Breadcrumb.Link>
				</Breadcrumb.Item>
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Page>{data?.service?.name || serviceID}</Breadcrumb.Page>
				</Breadcrumb.Item>
			</Breadcrumb.List>
		</Breadcrumb.Root>
	</div>

	<!-- Navigation Tabs -->
	<div class="border-b border-border overflow-x-auto scrollbar-hide">
		<div class="px-6">
			<Tabs.Root value={activeTab()} onValueChange={handleTabChange}>
				<Tabs.List>
					{#each tabs as tab (tab.id)}
						<Tabs.Trigger value={tab.id}>
							{tab.label}
						</Tabs.Trigger>
					{/each}
				</Tabs.List>
			</Tabs.Root>
		</div>
	</div>

	<div class="flex-1 overflow-hidden">
		{@render children?.()}
	</div>
</div>
