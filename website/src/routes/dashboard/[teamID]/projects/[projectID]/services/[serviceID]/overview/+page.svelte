<script lang="ts">
	import { page } from '$app/state';
	import { getContext } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';
	import ServiceLogsSheet from '$lib/components/ServiceLogsSheet.svelte';
	import { authPatch } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import PlayIcon from '~icons/lucide/play';
	import StopIcon from '~icons/lucide/square';
	import RestartIcon from '~icons/lucide/rotate-cw';
	import ActivityIcon from '~icons/lucide/activity';
	import ServerIcon from '~icons/lucide/server';
	import TerminalIcon from '~icons/lucide/terminal';
	import type { Service } from '$lib/schemas/service';
	import { ServiceState } from '$lib/schemas/service';
	import type { Team } from '$lib/schemas/team';
	import type { Project } from '$lib/schemas/project';

	// Get layout data with service information
	interface LayoutData {
		team: Team | null;
		project: Project | null;
		service: Service;
	}
	const data = getContext<LayoutData>('layoutData');
	let service: Service = $state(data.service);

	// Get route params
	const { teamID, projectID, serviceID } = page.params;

	// Loading states
	let isStarting = $state(false);
	let isStopping = $state(false);
	let isRestarting = $state(false);

	// Log sheet state
	let showLogSheet = $state(false);
	let logMessages: Array<{
		id: string;
		timestamp: string;
		type: 'log' | 'error' | 'info' | 'status';
		message: string;
	}> = $state([]);

	// Service status variants
	function getStatusClass(status: string) {
		switch (status) {
			case ServiceState.RUNNING:
				return 'bg-green-100 text-green-800 border-green-200';
			case ServiceState.STOPPED:
				return 'bg-gray-100 text-gray-800 border-gray-200';
			case 'error':
				return 'bg-red-100 text-red-800 border-red-200';
			case ServiceState.STARTING:
				return 'bg-yellow-100 text-yellow-800 border-yellow-200';
			case ServiceState.STOPPING:
				return 'bg-orange-100 text-orange-800 border-orange-200';
			case ServiceState.RESTARTING:
				return 'bg-blue-100 text-blue-800 border-blue-200';
			default:
				return 'bg-gray-100 text-gray-800 border-gray-200';
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
	async function handleSSEResponse(response: Response, operationType: string) {
		// Open log sheet when operation starts
		showLogSheet = true;
		addLogMessage('info', `Starting ${operationType} operation...`);

		if (!response.body) {
			addLogMessage('error', 'No response body for SSE stream');
			resetLoadingStates();
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
										service.state = data.final_state;
										addLogMessage(
											'status',
											`Operation completed. Service state: ${data.final_state}`
										);
										resetLoadingStates();
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
									service.state = data.final_state;
									addLogMessage(
										'status',
										`Operation completed. Service state: ${data.final_state}`
									);
									resetLoadingStates();
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
			resetLoadingStates();
		}
	}

	function resetLoadingStates() {
		isStarting = false;
		isStopping = false;
		isRestarting = false;
	}

	// Action handlers
	async function startService() {
		if (service.state !== ServiceState.STOPPED) return;

		isStarting = true;
		service.state = ServiceState.STARTING;

		try {
			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/state`,
				{ state: 'start' }
			);

			if (response.ok) {
				await handleSSEResponse(response, 'start');
			} else {
				console.error('Failed to start service:', response.statusText);
				service.state = ServiceState.STOPPED;
				isStarting = false;
			}
		} catch (error) {
			console.error('Error starting service:', error);
			service.state = ServiceState.STOPPED;
			isStarting = false;
		}
	}

	async function stopService() {
		if (service.state !== ServiceState.RUNNING) return;

		isStopping = true;
		service.state = ServiceState.STOPPING;

		try {
			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/state`,
				{ state: 'stop' }
			);

			if (response.ok) {
				await handleSSEResponse(response, 'stop');
			} else {
				console.error('Failed to stop service:', response.statusText);
				service.state = ServiceState.RUNNING;
				isStopping = false;
			}
		} catch (error) {
			console.error('Error stopping service:', error);
			service.state = ServiceState.RUNNING;
			isStopping = false;
		}
	}

	async function restartService() {
		if (service.state !== ServiceState.RUNNING) return;

		isRestarting = true;
		service.state = ServiceState.RESTARTING;

		try {
			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/state`,
				{ state: 'restart' }
			);

			if (response.ok) {
				await handleSSEResponse(response, 'restart');
			} else {
				console.error('Failed to restart service:', response.statusText);
				service.state = ServiceState.RUNNING;
				isRestarting = false;
			}
		} catch (error) {
			console.error('Error restarting service:', error);
			service.state = ServiceState.RUNNING;
			isRestarting = false;
		}
	}
</script>

<div class="container mx-auto space-y-6 p-6">
	<!-- Header Section -->
	<div class="flex items-center justify-between">
		<div class="space-y-1">
			<div class="flex items-center gap-2">
				<ServerIcon class="h-6 w-6" />
				<h1 class="text-2xl font-bold">{service.name}</h1>
				<span
					class="rounded-full border px-2 py-1 text-xs font-medium capitalize {getStatusClass(
						service.state
					)}"
				>
					{service.state}
				</span>
			</div>
			<p class="text-muted-foreground">Service overview and management</p>
		</div>
	</div>

	<!-- Quick Actions -->
	<Card>
		<CardHeader>
			<CardTitle class="flex items-center gap-2">
				<ActivityIcon class="h-5 w-5" />
				Quick Actions
			</CardTitle>
			<CardDescription>Manage your service state and operations</CardDescription>
		</CardHeader>
		<CardContent>
			<div class="flex flex-wrap gap-2">
				<Button
					onclick={startService}
					disabled={service.state !== ServiceState.STOPPED || isStarting}
					class="flex items-center gap-2"
				>
					<PlayIcon class="h-4 w-4" />
					{isStarting ? 'Starting...' : 'Start'}
				</Button>
				<Button
					variant="destructive"
					onclick={stopService}
					disabled={service.state !== ServiceState.RUNNING || isStopping}
					class="flex items-center gap-2"
				>
					<StopIcon class="h-4 w-4" />
					{isStopping ? 'Stopping...' : 'Stop'}
				</Button>
				<Button
					variant="outline"
					onclick={restartService}
					disabled={service.state !== ServiceState.RUNNING || isRestarting}
					class="flex items-center gap-2"
				>
					<RestartIcon class="h-4 w-4" />
					{isRestarting ? 'Restarting...' : 'Restart'}
				</Button>
				<Button
					variant="secondary"
					onclick={() => (showLogSheet = true)}
					class="flex items-center gap-2"
				>
					<TerminalIcon class="h-4 w-4" />
					View Logs
				</Button>
			</div>
		</CardContent>
	</Card>

	<!-- Service Details -->
	<Card>
		<CardHeader>
			<CardTitle>Service Details</CardTitle>
			<CardDescription>Configuration and metadata for this service</CardDescription>
		</CardHeader>
		<CardContent>
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				<div class="space-y-2">
					<div class="text-sm font-medium text-muted-foreground">Service ID</div>
					<div class="font-mono text-sm">{service.id}</div>
				</div>
				<div class="space-y-2">
					<div class="text-sm font-medium text-muted-foreground">Name</div>
					<div>{service.name}</div>
				</div>
				<div class="space-y-2">
					<div class="text-sm font-medium text-muted-foreground">Description</div>
					<div>{service.description || 'No description provided'}</div>
				</div>
				<div class="space-y-2">
					<div class="text-sm font-medium text-muted-foreground">Created</div>
					<div>{new Date(service.created_at).toLocaleDateString()}</div>
				</div>
				{#if service.updated_at}
					<div class="space-y-2">
						<div class="text-sm font-medium text-muted-foreground">Last Updated</div>
						<div>{new Date(service.updated_at).toLocaleDateString()}</div>
					</div>
				{/if}
			</div>
		</CardContent>
	</Card>
</div>

<!-- Service Operation Logs Sheet -->
<ServiceLogsSheet bind:open={showLogSheet} messages={logMessages} />
