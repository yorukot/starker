<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import LucideGitBranch from '~icons/lucide/git-branch';
	import LucideLoader2 from '~icons/lucide/loader-2';
	import LucideCheck from '~icons/lucide/check';
	import LucideX from '~icons/lucide/x';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { createGitServiceSchema, type CreateGitServiceForm } from '$lib/schemas/request/service';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { toast } from 'svelte-sonner';
	import type { PageData } from './$types';
	import { getValidToken } from '$lib/api/auth';

	let { data }: { data: PageData } = $props();

	const project = $derived(data.project);
	const servers = $derived(data.servers || []);
	const error = $derived(data.error || null);

	let streamingLogs: string[] = $state([]);
	let isStreaming = $state(false);
	let streamCompleted = $state(false);
	let streamError = $state<string | null>(null);
	let createdServiceId = $state<string | null>(null);

	const {
		form,
		errors,
		isSubmitting,
		data: formData
	} = createForm<CreateGitServiceForm>({
		extend: validator({ schema: createGitServiceSchema }),
		initialValues: {
			name: '',
			description: '',
			server_id: '',
			repo_url: '',
			branch: 'main',
			docker_compose_file_path: '',
			auto_deploy: false
		},
		onSubmit: async (values) => {
			try {
				streamingLogs = [];
				isStreaming = true;
				streamCompleted = false;
				streamError = null;
				createdServiceId = null;

				const token = await getValidToken();
				if (!token) {
					toast.error('Authentication required. Please log in again.');
					return;
				}

				const response = await fetch(
					`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/projects/${page.params.projectID}/services/git`,
					{
						method: 'POST',
						headers: {
							Authorization: `Bearer ${token}`,
							'Content-Type': 'application/json'
						},
						body: JSON.stringify(values)
					}
				);

				if (!response.ok) {
					const errorData = await response.json().catch(() => ({}));
					toast.error(errorData.message || 'Failed to create git service');
					isStreaming = false;
					return;
				}

				// Handle SSE stream
				const reader = response.body?.getReader();
				const decoder = new TextDecoder();

				if (!reader) {
					toast.error('Failed to read response stream');
					isStreaming = false;
					return;
				}

				while (true) {
					const { value, done } = await reader.read();
					if (done) break;

					const chunk = decoder.decode(value);
					const lines = chunk.split('\n');

					for (const line of lines) {
						if (line.startsWith('data: ')) {
							try {
								const data = JSON.parse(line.substring(6));

								switch (data.type) {
									case 'log':
										streamingLogs = [...streamingLogs, data.message];
										break;
									case 'error':
										streamError = data.message;
										isStreaming = false;
										toast.error(data.message);
										return;
									case 'success':
										streamingLogs = [...streamingLogs, data.message];
										createdServiceId = data.service_id;
										streamCompleted = true;
										isStreaming = false;
										toast.success('Git service created successfully!');

										// Navigate to service detail page after a short delay
										setTimeout(() => {
											goto(
												`/dashboard/${page.params.teamID}/projects/${page.params.projectID}/services/${data.service_id}`
											);
										}, 2000);
										return;
								}
							} catch (err) {
								console.error('Error parsing SSE data:', err);
							}
						}
					}
				}
			} catch (err) {
				console.error('Error creating git service:', err);
				toast.error('Failed to create git service. Please try again.');
				isStreaming = false;
				streamError = err instanceof Error ? err.message : 'Unknown error';
			}
		}
	});

	function goBack() {
		goto(`/dashboard/${page.params.teamID}/projects/${page.params.projectID}/services/new`);
	}

	function scrollToBottom() {
		const logsContainer = document.getElementById('streaming-logs');
		if (logsContainer) {
			logsContainer.scrollTop = logsContainer.scrollHeight;
		}
	}

	// Auto-scroll logs to bottom when new logs arrive
	$effect(() => {
		if (streamingLogs.length > 0) {
			setTimeout(scrollToBottom, 100);
		}
	});
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<Button variant="ghost" size="sm" onclick={goBack} class="gap-2">
			<LucideArrowLeft class="h-4 w-4" />
			Back to Service Types
		</Button>
	</div>

	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else}
		<!-- Header -->
		<div class="mb-6">
			<div class="flex items-center gap-3">
				<div class="rounded-lg border border-primary/20 bg-primary/10 p-3">
					<LucideGitBranch class="h-6 w-6 text-primary" />
				</div>
				<div>
					<h1 class="text-2xl font-semibold text-foreground">Create Git Service</h1>
					<p class="text-sm text-muted-foreground">
						Deploy a service from a Git repository to {project?.name || 'your project'}
					</p>
				</div>
			</div>
		</div>

		<!-- Streaming Progress (shown when streaming) -->
		{#if isStreaming || streamCompleted || streamError}
			<Card.Root class="mb-6">
				<Card.Header>
					<div class="flex items-center gap-2">
						{#if isStreaming}
							<LucideLoader2 class="h-5 w-5 animate-spin text-primary" />
							<Card.Title>Creating Git Service...</Card.Title>
						{:else if streamCompleted}
							<LucideCheck class="h-5 w-5 text-green-600" />
							<Card.Title>Git Service Created Successfully!</Card.Title>
						{:else if streamError}
							<LucideX class="h-5 w-5 text-red-600" />
							<Card.Title>Git Service Creation Failed</Card.Title>
						{/if}
					</div>
				</Card.Header>
				<Card.Content>
					<div class="space-y-2">
						<Label>Progress Logs:</Label>
						<div
							id="streaming-logs"
							class="max-h-96 overflow-y-auto rounded-md border bg-muted/10 p-3 font-mono text-xs"
						>
							{#if streamingLogs.length === 0}
								<p class="text-muted-foreground">Waiting for logs...</p>
							{:else}
								{#each streamingLogs as log}
									<div class="mb-1 break-words whitespace-pre-wrap">
										{log}
									</div>
								{/each}
							{/if}
						</div>
						{#if streamError}
							<div class="mt-2 text-sm text-destructive">
								Error: {streamError}
							</div>
						{/if}
						{#if streamCompleted && createdServiceId}
							<div class="mt-2 text-sm text-green-600">
								Service ID: {createdServiceId}
							</div>
						{/if}
					</div>
				</Card.Content>
			</Card.Root>
		{/if}

		<!-- Form -->
		<form
			use:form
			class="space-y-6"
			class:pointer-events-none={isStreaming}
			class:opacity-75={isStreaming}
		>
			<Card.Root>
				<Card.Header>
					<Card.Title>Service Configuration</Card.Title>
					<Card.Description>Configure your service name and description</Card.Description>
				</Card.Header>
				<Card.Content class="space-y-4">
					<div class="space-y-2">
						<Label for="name">Service Name *</Label>
						<Input
							id="name"
							name="name"
							placeholder="my-web-app"
							class={$errors.name ? 'border-destructive' : ''}
						/>
						{#if $errors.name}
							<p class="text-sm text-destructive">{$errors.name[0]}</p>
						{/if}
					</div>

					<div class="space-y-2">
						<Label for="description">Description</Label>
						<Textarea
							id="description"
							name="description"
							placeholder="A brief description of your service (optional)"
							rows={3}
							class={$errors.description ? 'border-destructive' : ''}
						/>
						{#if $errors.description}
							<p class="text-sm text-destructive">{$errors.description[0]}</p>
						{/if}
					</div>

					<div class="space-y-2">
						<Label for="server_id">Target Server *</Label>
						{#if servers.length === 0}
							<div class="flex flex-col gap-2">
								<p class="text-sm text-muted-foreground">
									No servers available. You need to add a server before creating services.
								</p>
								<Button
									variant="outline"
									size="sm"
									onclick={() => goto(`/dashboard/${page.params.teamID}/servers/new`)}
								>
									Add Server
								</Button>
							</div>
						{:else}
							<Select.Root type="single" bind:value={$formData.server_id}>
								<Select.Trigger class="w-full {$errors.server_id ? 'border-destructive' : ''}">
									{$formData.server_id
										? servers.find((s) => s.id === $formData.server_id)?.name
										: 'Select a server'}
								</Select.Trigger>
								<Select.Content>
									{#each servers as server (server.id)}
										<Select.Item value={server.id} label={server.name}>
											<div class="flex flex-col">
												<div class="font-medium">{server.name}</div>
												{#if server.description}
													<div class="text-xs text-muted-foreground">{server.description}</div>
												{/if}
											</div>
										</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
							{#if $errors.server_id}
								<p class="text-sm text-destructive">{$errors.server_id[0]}</p>
							{/if}
						{/if}
					</div>
				</Card.Content>
			</Card.Root>

			<Card.Root>
				<Card.Header>
					<Card.Title>Repository Configuration</Card.Title>
					<Card.Description>Configure the Git repository to clone and deploy</Card.Description>
				</Card.Header>
				<Card.Content class="space-y-4">
					<div class="space-y-2">
						<Label for="repo_url">Repository URL *</Label>
						<Input
							id="repo_url"
							name="repo_url"
							placeholder="https://github.com/user/my-app.git"
							class={$errors.repo_url ? 'border-destructive' : ''}
						/>
						{#if $errors.repo_url}
							<p class="text-sm text-destructive">{$errors.repo_url[0]}</p>
						{/if}
						<p class="text-xs text-muted-foreground">
							Supports both HTTPS and SSH URLs. Must be accessible from the target server.
						</p>
					</div>

					<div class="space-y-2">
						<Label for="branch">Branch *</Label>
						<Input
							id="branch"
							name="branch"
							placeholder="main"
							class={$errors.branch ? 'border-destructive' : ''}
						/>
						{#if $errors.branch}
							<p class="text-sm text-destructive">{$errors.branch[0]}</p>
						{/if}
						<p class="text-xs text-muted-foreground">The Git branch to clone and deploy from.</p>
					</div>

					<div class="space-y-2">
						<Label for="docker_compose_file_path">Custom Docker Compose File Path</Label>
						<Input
							id="docker_compose_file_path"
							name="docker_compose_file_path"
							placeholder="docker-compose.prod.yml"
							class={$errors.docker_compose_file_path ? 'border-destructive' : ''}
						/>
						{#if $errors.docker_compose_file_path}
							<p class="text-sm text-destructive">{$errors.docker_compose_file_path[0]}</p>
						{/if}
						<p class="text-xs text-muted-foreground">
							Optional: Specify a custom path to the Docker Compose file. If not provided, common
							locations will be automatically searched.
						</p>
					</div>
				</Card.Content>
			</Card.Root>

			<Card.Root>
				<Card.Header>
					<Card.Title>Deployment Settings</Card.Title>
					<Card.Description>Configure deployment behavior and automation</Card.Description>
				</Card.Header>
				<Card.Content class="space-y-4">
					<div class="flex items-center space-x-2">
						<input
							type="checkbox"
							id="auto_deploy"
							name="auto_deploy"
							bind:checked={$formData.auto_deploy}
							class="h-4 w-4 rounded border border-border text-primary focus:ring-2 focus:ring-primary focus:ring-offset-2"
						/>
						<Label for="auto_deploy" class="cursor-pointer">
							Enable auto-deploy on repository changes
						</Label>
					</div>
					<p class="text-xs text-muted-foreground">
						When enabled, the service will automatically update when changes are detected in the
						repository.
					</p>
				</Card.Content>
			</Card.Root>

			<!-- Actions -->
			<div class="flex items-center justify-end gap-4">
				<Button variant="outline" type="button" onclick={goBack} disabled={isStreaming}>
					Cancel
				</Button>
				<Button
					type="submit"
					disabled={$isSubmitting || isStreaming || servers.length === 0}
					class="gap-2"
				>
					{#if $isSubmitting || isStreaming}
						<LucideLoader2 class="h-4 w-4 animate-spin" />
						Creating Service...
					{:else}
						<LucideGitBranch class="h-4 w-4" />
						Create Git Service
					{/if}
				</Button>
			</div>
		</form>
	{/if}
</div>
