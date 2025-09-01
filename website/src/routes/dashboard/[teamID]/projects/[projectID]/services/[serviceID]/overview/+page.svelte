<script lang="ts">
	import { page } from '$app/state';
	import { getContext } from 'svelte';
	import { invalidate } from '$app/navigation';
	import InfoIcon from '~icons/lucide/info';
	import ContainerIcon from '~icons/lucide/container';
	import ServiceQuickActions from '$lib/components/service-quick-actions.svelte';
	import ServiceLogsSheet from '$lib/components/service-logs-sheet.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import type { Service, ServiceContainer } from '$lib/schemas/service';
	import { ServiceState } from '$lib/schemas/service';
	import type { PageData } from './$types';

	// Props from page loader
	let { data }: { data: PageData } = $props();

	// Get layout data from context
	const getLayoutData = getContext<() => { service: Service | null }>('layoutData');
	const layoutData = $derived(getLayoutData?.() ?? { service: null });
	let service = $derived(layoutData.service);

	// Extract route parameters
	const teamID = $derived(page.params.teamID!);
	const projectID = $derived(page.params.projectID!);
	const serviceID = $derived(page.params.serviceID!);

	// Containers data from page loader
	const containers = $derived(data.containers || []);
	const containersError = $derived(data.error || null);

	// Quick Actions state
	let quickActionsRef = $state<ServiceQuickActions>();
	let showLogSheet = $state(false);
	let logMessages: Array<{
		id: string;
		timestamp: string;
		type: 'log' | 'error' | 'info' | 'status' | 'step';
		message: string;
	}> = $state([]);

	// Add log message with timestamp and unique ID
	function addLogMessage(type: 'log' | 'error' | 'info' | 'status' | 'step', message: string) {
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

	// Handle SSE stream from response (copied from layout)
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
								case 'step':
									addLogMessage('log', data.message);
									break;
								case 'info':
									// Check if this is a completion message (has state property)
									if (data.state) {
										if (service) {
											service.state = data.state;
										}
										addLogMessage('status', `Operation completed. Service state: ${data.state}`);
										quickActionsRef?.resetStates();
										// Reload container data when operation completes
										invalidate(`project:${projectID}`);
										return;
									} else {
										addLogMessage('info', data.message);
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
		} catch (error) {
			addLogMessage('error', `Error reading SSE stream: ${error}`);
			quickActionsRef?.resetStates();
		}
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

	// Container status border variants
	function getContainerBorderClass(status?: string) {
		if (!status) return 'border-l-secondary';

		switch (status) {
			case 'running':
				return 'border-l-green-500';
			case 'stopped':
			case 'exited':
			case 'removed':
				return 'border-l-red-500';
			default:
				return 'border-l-secondary';
		}
	}

</script>

<div class="flex h-full flex-col gap-6 p-6">
	<!-- Header -->
	<div class="flex items-center gap-3">
		<div class="rounded-lg border border-primary/20 bg-primary/10 p-3">
			<InfoIcon class="h-6 w-6 text-primary" />
		</div>
		<div>
			<h1 class="text-2xl font-semibold text-foreground">Service Overview</h1>
			<p class="text-sm text-muted-foreground">
				View comprehensive information about your service status, metrics, and activity
			</p>
		</div>
	</div>

	{#if service}
		<!-- Service Status Card -->
		<Card.Root>
			<Card.Header class="pb-3">
				<div class="flex items-center justify-between">
					<div class="space-y-1">
						<h3 class="text-lg font-medium">Service Status</h3>
						<div class="flex items-center gap-3">
							<span class="text-sm text-muted-foreground">Current State:</span>
							<span
								class="rounded-full border px-3 py-1 text-sm font-medium capitalize {getStatusClass(
									service?.state
								)}"
							>
								{service?.state || 'Unknown'}
							</span>
						</div>
					</div>
					<!-- Quick Actions -->
					<div class="flex-shrink-0">
						<ServiceQuickActions
							bind:this={quickActionsRef}
							bind:service
							{teamID}
							{projectID}
							{serviceID}
							onOperationStart={handleSSEResponse}
							onShowLogs={() => (showLogSheet = true)}
						/>
					</div>
				</div>
			</Card.Header>
			{#if service.description}
				<Card.Content class="pt-0">
					<p class="text-sm text-muted-foreground">
						<span class="font-medium">Description:</span> {service.description}
					</p>
				</Card.Content>
			{/if}
		</Card.Root>

		<!-- Service Containers -->
		<Card.Root>
			<Card.Header>
				<div class="flex items-center gap-2">
					<ContainerIcon class="h-5 w-5 text-primary" />
					<h3 class="text-lg font-medium">Containers</h3>
				</div>
			</Card.Header>
			<Card.Content>
				{#if containersError}
					<div class="text-center py-8">
						<p class="text-destructive text-sm">{containersError}</p>
					</div>
				{:else if containers.length > 0}
					<div class="space-y-3">
						{#each containers as container (container.id)}
							<div class="flex items-center gap-3 rounded-lg border border-border/50 border-l-4 bg-card/30 p-4 transition-colors hover:bg-card/60 {getContainerBorderClass(container.state)}">
								<ContainerIcon class="h-5 w-5 text-muted-foreground flex-shrink-0" />
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2 mb-1">
										<h4 class="font-medium text-sm truncate">{container.container_name}</h4>
										<span class="text-xs text-muted-foreground capitalize">
											{container.state}
										</span>
									</div>
									<div class="flex items-center gap-4 text-xs text-muted-foreground">
										{#if container.container_id}
											<span>
												<span class="font-medium">ID:</span> 
												<span class="font-mono">{container.container_id.slice(0, 12)}...</span>
											</span>
										{/if}
										<span>
											<span class="font-medium">Created:</span> 
											{new Date(container.created_at).toLocaleDateString()}
										</span>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="text-center py-8">
						<div class="rounded-full border border-muted bg-muted/30 p-4 mx-auto w-fit mb-4">
							<ContainerIcon class="h-8 w-8 text-muted-foreground/50" />
						</div>
						<div class="space-y-1">
							<p class="font-medium text-sm">No containers found</p>
							<p class="text-xs text-muted-foreground">
								Containers will appear here when the service is running
							</p>
						</div>
					</div>
				{/if}
			</Card.Content>
		</Card.Root>

		<!-- Additional service information cards can be added here -->
		<Card.Root class="flex-1">
			<Card.Content class="flex h-full items-center justify-center pt-6">
				<p class="text-muted-foreground">Additional service metrics and information coming soon...</p>
			</Card.Content>
		</Card.Root>
	{:else}
		<!-- No service data -->
		<div class="flex-1 flex items-center justify-center text-muted-foreground">
			<p>No service data available</p>
		</div>
	{/if}
</div>

<!-- Service Operation Logs Sheet -->
<ServiceLogsSheet bind:open={showLogSheet} messages={logMessages} />