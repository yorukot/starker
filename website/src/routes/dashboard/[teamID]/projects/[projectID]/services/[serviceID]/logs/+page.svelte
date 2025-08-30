<script lang="ts">
	import { page } from '$app/state';
	import * as Select from '$lib/components/ui/select/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import LogsViewer from '$lib/components/logs-viewer.svelte';
  import LogsIcon from '~icons/lucide/file-text';
	import PlayIcon from '~icons/lucide/play';
	import PauseIcon from '~icons/lucide/pause';
	import AlertCircleIcon from '~icons/lucide/alert-circle';
	import type { PageData } from './$types';
	import { getValidToken } from '$lib/api/auth';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';

	interface LogMessage {
		id: string;
		timestamp: string;
		type: 'log' | 'error' | 'info' | 'status';
		message: string;
	}

	let { data }: { data: PageData } = $props();

	// Extract route parameters
	const teamID = page.params.teamID!;
	const projectID = page.params.projectID!;
	const serviceID = page.params.serviceID!;

	// Reactive data
	const containers = $derived(data.containers || []);
	const service = $derived(data.service);
	const error = $derived(data.error || null);

	// State management
	let selectedContainerID = $state<string>('');
	let isStreaming = $state(false);
	let streamError = $state<string | null>(null);
	let logMessages: LogMessage[] = $state([]);
	let streamController: AbortController | null = $state(null);

	// Log options
	let includeTail = $state(true);
	let tailLines = $state('100');
	let includeTimestamps = $state(true);
	let followLogs = $state(true);

	// Computed values
	const selectedContainer = $derived(containers.find((c) => c.id === selectedContainerID) || null);

	const selectedContainerName = $derived(selectedContainer?.container_name || 'Select a container');

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

	// Start streaming logs from selected container
	async function startLogStream() {
		if (!selectedContainerID) {
			streamError = 'Please select a container first';
			return;
		}

		try {
			streamError = null;
			logMessages = [];
			isStreaming = true;

			const token = await getValidToken();
			if (!token) {
				streamError = 'Authentication required. Please log in again.';
				isStreaming = false;
				return;
			}

			// Build query parameters
			const params = new URLSearchParams();
			if (followLogs) params.append('follow', 'true');
			if (includeTail && tailLines) params.append('tail', tailLines);
			if (includeTimestamps) params.append('timestamps', 'true');

			const url = `${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/containers/${selectedContainerID}/logs?${params.toString()}`;

			// Create abort controller for cancelling the stream
			streamController = new AbortController();

			addLogMessage(
				'info',
				`Starting log stream for container: ${selectedContainer?.container_name}`
			);

			// Use fetch with Authorization header (like the service state SSE)
			const response = await fetch(url, {
				method: 'GET',
				headers: {
					Authorization: `Bearer ${token}`,
					Accept: 'text/event-stream',
					'Cache-Control': 'no-cache'
				},
				signal: streamController.signal
			});

			if (!response.ok) {
				const errorText = await response.text();
				streamError = `Failed to start log stream: ${response.status} ${errorText}`;
				isStreaming = false;
				return;
			}

			if (!response.body) {
				streamError = 'No response body for log stream';
				isStreaming = false;
				return;
			}

			addLogMessage('status', 'Log stream connected');

			// Process the SSE stream using the same pattern as service state
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
									case 'info':
										addLogMessage('info', data.message);
										break;
									default:
										// Handle raw log messages
										addLogMessage('log', data.message || JSON.stringify(data));
										break;
								}
							} catch (error: unknown) {
								// Handle raw text messages (non-JSON data)
								addLogMessage('log', jsonData);
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
								case 'info':
									addLogMessage('info', data.message);
									break;
								default:
									addLogMessage('log', data.message || JSON.stringify(data));
									break;
							}
						} catch (error: unknown) {
							addLogMessage('log', jsonData);
						}
					}
				}
			} catch (error: unknown) {
				if (error instanceof Error && error.name !== 'AbortError') {
					addLogMessage('error', `Error reading log stream: ${error.message}`);
				}
			}
		} catch (error: unknown) {
			if (error instanceof Error && error.name !== 'AbortError') {
				console.error('Error starting log stream:', error);
				streamError = error.message;
			} else if (!(error instanceof Error)) {
				streamError = 'Unknown error occurred';
			}
			isStreaming = false;
		}
	}

	// Stop streaming logs
	function stopLogStream() {
		if (streamController) {
			streamController.abort();
			streamController = null;
		}
		isStreaming = false;
		addLogMessage('status', 'Log stream stopped');
	}

	// Clear logs
	function clearLogs() {
		logMessages = [];
	}

	// Handle container selection change
	function handleContainerChange(containerID: string) {
		// Stop current stream if running
		if (isStreaming) {
			stopLogStream();
		}

		selectedContainerID = containerID;
		streamError = null;

		// Auto-start streaming for the new container
		if (containerID) {
			setTimeout(() => startLogStream(), 100);
		}
	}

	// Cleanup on component destroy
	function cleanup() {
		if (streamController) {
			streamController.abort();
		}
	}
</script>

<svelte:window onbeforeunload={cleanup} />

<div class="flex h-full flex-col gap-6 p-6">
	{#if error}
		<Alert.Root variant="destructive">
			<AlertCircleIcon class="h-4 w-4" />
			<Alert.Title>Error</Alert.Title>
			<Alert.Description>{error}</Alert.Description>
		</Alert.Root>
	{:else}
		<!-- Header -->
		<div class="flex items-center gap-3">
			<div class="rounded-lg border border-primary/20 bg-primary/10 p-3">
				<LogsIcon class="h-6 w-6 text-primary" />
			</div>
			<div>
				<h1 class="text-2xl font-semibold text-foreground">Container Logs</h1>
				<p class="text-sm text-muted-foreground">
					{service?.name
						? `View real-time logs for ${service.name} containers`
						: 'View container logs'}
				</p>
			</div>
		</div>

		{#if containers.length === 0}
			<Alert.Root>
				<AlertCircleIcon class="h-4 w-4" />
				<Alert.Title>No containers found</Alert.Title>
				<Alert.Description>
					This service doesn't have any containers yet. Containers will appear here once the service
					is deployed.
				</Alert.Description>
			</Alert.Root>
		{:else}
			<!-- Container Selection and Options -->
			<Card.Root>
				<Card.Header>
					<Card.Title>Container Selection</Card.Title>
					<Card.Description>Choose a container to view its logs</Card.Description>
				</Card.Header>
				<Card.Content class="space-y-4">
					<!-- Container Selector -->
					<div class="space-y-2">
						<Label>Container</Label>
						<Select.Root
							type="single"
							value={selectedContainerID}
							onValueChange={handleContainerChange}
						>
							<Select.Trigger class="w-full">
								{selectedContainerName}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each containers as container (container.id)}
										<Select.Item value={container.id} label={container.container_name}>
											<div class="flex flex-col">
												<div class="font-medium">{container.container_name}</div>
												<div class="text-xs text-muted-foreground">
													State: {container.state}
													{#if container.container_id}
														" ID: {container.container_id.slice(0, 12)}...
													{/if}
												</div>
											</div>
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</div>

					<!-- Log Options -->
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
						<div class="flex items-center space-x-2">
							<Checkbox id="timestamps" bind:checked={includeTimestamps} />
							<Label for="timestamps" class="cursor-pointer text-sm">Include timestamps</Label>
						</div>

						<div class="flex items-center space-x-2">
							<Checkbox id="follow" bind:checked={followLogs} />
							<Label for="follow" class="cursor-pointer text-sm">Follow log output</Label>
						</div>

						<div class="flex items-center space-x-2">
							<Checkbox id="tail" bind:checked={includeTail} />
							<Label for="tail" class="cursor-pointer text-sm">Show recent lines:</Label>
							<Input
								type="number"
								bind:value={tailLines}
								disabled={!includeTail}
								class="h-8 w-20"
								placeholder="100"
								min="1"
								max="1000"
							/>
						</div>
					</div>

					<!-- Stream Control -->
					<div class="flex items-center justify-between pt-2">
						<div class="flex items-center gap-2">
							<Button
								variant="outline"
								size="sm"
								onclick={isStreaming ? stopLogStream : startLogStream}
								disabled={!selectedContainerID}
								class="gap-2"
							>
								{#if isStreaming}
									<PauseIcon class="h-4 w-4" />
									Stop Stream
								{:else}
									<PlayIcon class="h-4 w-4" />
									Start Stream
								{/if}
							</Button>

							<Button variant="outline" size="sm" onclick={clearLogs}>Clear Logs</Button>
						</div>

						{#if isStreaming}
							<div class="flex items-center gap-2 text-sm text-muted-foreground">
								<div class="h-2 w-2 animate-pulse rounded-full bg-green-500"></div>
								Streaming live logs
							</div>
						{/if}
					</div>
				</Card.Content>
			</Card.Root>

			{#if streamError}
				<Alert.Root variant="destructive">
					<AlertCircleIcon class="h-4 w-4" />
					<Alert.Title>Stream Error</Alert.Title>
					<Alert.Description>{streamError}</Alert.Description>
				</Alert.Root>
			{/if}

			<!-- Logs Display -->
			<div class="min-h-0 flex-1">
				<LogsViewer
					messages={logMessages}
					title="Container Logs"
					description={selectedContainer
						? `Logs from ${selectedContainer.container_name}`
						: 'Select a container to view logs'}
					class="max-h-screen"
				/>
			</div>
		{/if}
	{/if}
</div>
