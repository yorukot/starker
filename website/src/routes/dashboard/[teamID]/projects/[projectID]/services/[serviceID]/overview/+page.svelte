<script lang="ts">
	import { page } from '$app/state';
	import { getContext } from 'svelte';
	import InfoIcon from '~icons/lucide/info';
	import ServiceQuickActions from '$lib/components/service-quick-actions.svelte';
	import ServiceLogsSheet from '$lib/components/service-logs-sheet.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import type { Service } from '$lib/schemas/service';
	import { ServiceState } from '$lib/schemas/service';

	// Get layout data from context
	const getLayoutData = getContext<() => { service: Service | null }>('layoutData');
	const layoutData = $derived(getLayoutData?.() ?? { service: null });
	let service = $derived(layoutData.service);

	// Extract route parameters
	const teamID = $derived(page.params.teamID!);
	const projectID = $derived(page.params.projectID!);
	const serviceID = $derived(page.params.serviceID!);

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
			<Card.Content class="pt-6">
				<div class="flex items-center justify-between">
					<div class="space-y-2">
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
						<div class="space-y-1">
							<p class="text-sm text-muted-foreground">
								<span class="font-medium">Name:</span> {service.name}
							</p>
							{#if service.description}
								<p class="text-sm text-muted-foreground">
									<span class="font-medium">Description:</span> {service.description}
								</p>
							{/if}
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