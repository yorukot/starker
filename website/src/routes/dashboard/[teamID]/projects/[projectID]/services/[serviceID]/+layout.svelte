<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { setContext } from 'svelte';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import ServiceQuickActions from '$lib/components/ServiceQuickActions.svelte';
	import ServiceLogsSheet from '$lib/components/ServiceLogsSheet.svelte';
	import type { Snippet } from 'svelte';
	import type { Team } from '$lib/schemas/team';
	import type { Project } from '$lib/schemas/project';
	import type { Service } from '$lib/schemas/service';
	import { ServiceState } from '$lib/schemas/service';

	interface LayoutData {
		team: Team | null;
		project: Project | null;
		service: Service | null;
	}

	let { children, data }: { children: Snippet; data: LayoutData } = $props();

	// Set context so child components can access layout data with updated service state
	let service = $state(data.service);
	setContext('layoutData', () => ({ ...data, service }));

	// Quick Actions state
	let quickActionsRef = $state<ServiceQuickActions>();
	let showLogSheet = $state(false);
	let logMessages: Array<{
		id: string;
		timestamp: string;
		type: 'log' | 'error' | 'info' | 'status';
		message: string;
	}> = $state([]);

	// Extract route parameters
	const teamID = $derived(page.params.teamID!);
	const projectID = $derived(page.params.projectID!);
	const serviceID = $derived(page.params.serviceID!);

	// Navigation tabs configuration
	const tabs = [
		{ id: 'overview', label: 'Overview', path: 'overview' },
		{ id: 'compose', label: 'Compose', path: 'compose' },
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
		
		// Default to overview if no match found or if we're at the base service URL
		return matchingTab ? matchingTab.id : 'overview';
	});

	// Handle tab navigation
	function handleTabChange(tabId: string) {
		const tab = tabs.find((t) => t.id === tabId);
		if (tab) {
			const newPath = `/dashboard/${teamID}/projects/${projectID}/services/${serviceID}/${tab.path}`;
			goto(newPath);
		}
	}

	// Navigation functions
	function navigateTo(path: string) {
		goto(path);
	}

	// Service status variants
	function getStatusClass(status?: string) {
		if (!status) return 'bg-secondary/50 border-secondary/30';
		
		switch (status) {
			case ServiceState.RUNNING:
				return 'bg-success/50 border-success/30';
			case ServiceState.STOPPED:
				return 'bg-destructive/50 border-destructive/30';
			case 'error':
				return 'bg-destructive/50 border-destructive/30';
			case ServiceState.STARTING:
				return 'bg-secondary/50 border-secondary/30';
			case ServiceState.STOPPING:
				return 'bg-secondary/50 border-secondary/30';
			case ServiceState.RESTARTING:
				return 'bg-secondary/50 border-secondary/30';
			default:
				return 'bg-secondary/50 border-secondary/30';
		}
	}

	// Add log message with timestamp and unique ID
	function addLogMessage(type: 'log' | 'error' | 'info' | 'status', message: string) {
		logMessages = [
			...logMessages,
			{
				id: `${Date.now()}-${Math.random()}`,
				timestamp: new Date().toLocaleTimeString(),
				type,
				message
			}
		];
	}

	// Handle SSE stream from response
	async function handleSSEResponse(operationType: string, response: Response) {
		// Open log sheet when operation starts
		showLogSheet = true;
		addLogMessage('info', `Starting ${operationType} operation...`);

		if (!response.body) {
			addLogMessage('error', 'No response body for SSE stream');
			quickActionsRef?.resetStates();
			return;
		}

		const reader = response.body.getReader();
		const decoder = new TextDecoder();
		let buffer = '';

		try {
			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				// Add new data to buffer
				buffer += decoder.decode(value, { stream: true });

				// Process complete lines from buffer
				const lines = buffer.split('\n');

				// Keep the last (potentially incomplete) line in buffer
				buffer = lines.pop() || '';

				// Process complete lines
				for (const line of lines) {
					if (line.startsWith('data: ')) {
						const jsonData = line.slice(6).trim();
						if (!jsonData) continue;

						try {
							const data = JSON.parse(jsonData);

							switch (data.type) {
								case 'log':
									addLogMessage('log', data.message);
									break;
								case 'error':
									addLogMessage('error', data.message);
									break;
								case 'status':
									if (data.status === 'completed') {
										if (service) {
											service.state = data.final_state;
										}
										addLogMessage(
											'status',
											`Operation completed. Service state: ${data.final_state}`
										);
										quickActionsRef?.resetStates();
										return;
									} else {
										addLogMessage('status', data.message || `Status: ${data.status}`);
									}
									break;
								default:
									// Handle any other message types by treating them as log
									addLogMessage('log', data.message || JSON.stringify(data));
									break;
							}
						} catch (error) {
							addLogMessage('error', `Failed to parse SSE data: ${error}`);
						}
					}
				}
			}

			// Process any remaining data in buffer
			if (buffer.trim() && buffer.startsWith('data: ')) {
				const jsonData = buffer.slice(6).trim();
				if (jsonData) {
					try {
						const data = JSON.parse(jsonData);
						switch (data.type) {
							case 'log':
								addLogMessage('log', data.message);
								break;
							case 'error':
								addLogMessage('error', data.message);
								break;
							case 'status':
								if (data.status === 'completed') {
									if (service) {
										service.state = data.final_state;
									}
									addLogMessage(
										'status',
										`Operation completed. Service state: ${data.final_state}`
									);
									quickActionsRef?.resetStates();
									return;
								} else {
									addLogMessage('status', data.message || `Status: ${data.status}`);
								}
								break;
							default:
								addLogMessage('log', data.message || JSON.stringify(data));
								break;
						}
					} catch (error) {
						addLogMessage('error', `Failed to parse remaining SSE data: ${error}`);
					}
				}
			}
		} catch (error) {
			addLogMessage('error', `Error reading SSE stream: ${error}`);
			quickActionsRef?.resetStates();
		}
	}
</script>

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
					<Breadcrumb.Page class="flex items-center gap-2">
						{service?.name || serviceID}
						{#if service}
							<span
								class="rounded-full border px-2 py-1 text-xs font-medium capitalize {getStatusClass(
									service?.state
								)}"
							>
								{service?.state}
							</span>
						{/if}
					</Breadcrumb.Page>
				</Breadcrumb.Item>
			</Breadcrumb.List>
		</Breadcrumb.Root>
	</div>

	<!-- Navigation Tabs with Actions -->
	<div class="scrollbar-hide overflow-x-auto border-b border-border">
		<div class="flex items-center justify-between px-6">
			<Tabs.Root value={activeTab()} onValueChange={handleTabChange}>
				<Tabs.List>
					{#each tabs as tab (tab.id)}
						<Tabs.Trigger value={tab.id}>
							{tab.label}
						</Tabs.Trigger>
					{/each}
				</Tabs.List>
			</Tabs.Root>

			<!-- Quick Actions -->
			{#if service}
				<ServiceQuickActions
					bind:this={quickActionsRef}
					bind:service
					{teamID}
					{projectID}
					{serviceID}
					onOperationStart={handleSSEResponse}
					onShowLogs={() => (showLogSheet = true)}
				/>
			{/if}
		</div>
	</div>

	<div class="flex-1 overflow-hidden">
		{@render children?.()}
	</div>
</div>

<!-- Service Operation Logs Sheet -->
<ServiceLogsSheet bind:open={showLogSheet} messages={logMessages} />

<style>
	.scrollbar-hide {
		-ms-overflow-style: none; /* IE and Edge */
		scrollbar-width: none; /* Firefox */
	}
	.scrollbar-hide::-webkit-scrollbar {
		display: none; /* Chrome, Safari, Opera */
	}
</style>
