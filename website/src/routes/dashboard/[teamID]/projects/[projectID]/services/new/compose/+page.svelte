<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import LucideContainer from '~icons/lucide/container';
	import LucideLoader2 from '~icons/lucide/loader-2';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { createServiceSchema, type CreateServiceForm } from '$lib/schemas/request/service';
	import { authPost } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { toast } from 'svelte-sonner';
	import { CodeEditor } from '$lib/components/ui/code-editor';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const project = $derived(data.project);
	const servers = $derived(data.servers || []);
	const error = $derived(data.error || null);

	let composeContent = $state(`services:
  app:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
    restart: unless-stopped
`);

	const {
		form,
		errors,
		isSubmitting,
		setData,
		data: formData
	} = createForm<CreateServiceForm>({
		extend: validator({ schema: createServiceSchema }),
		initialValues: {
			name: '',
			description: '',
			type: 'compose',
			server_id: '',
			compose_file: ''
		},
		onSubmit: async (values) => {
			try {
				const response = await authPost(
					`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/projects/${page.params.projectID}/services/compose`,
					{
						...values,
						compose_file: composeContent
					}
				);

				if (!response.ok) {
					const errorData = await response.json();
					toast.error(errorData.message || 'Failed to create service');
					return;
				}

				const service = await response.json();
				toast.success('Service created successfully!');

				// Navigate to the service detail page
				goto(
					`/dashboard/${page.params.teamID}/projects/${page.params.projectID}/services/${service.id}`
				);
			} catch (err) {
				console.error('Error creating service:', err);
				toast.error('Failed to create service. Please try again.');
			}
		}
	});

	// Computed trigger content for select
	const selectedServerName = $derived(
		$formData.server_id
			? (servers.find((s) => s.id === $formData.server_id)?.name ?? 'Select a Server')
			: 'Select a Server'
	);

	function goBack() {
		goto(`/dashboard/${page.params.teamID}/projects/${page.params.projectID}/services/new`);
	}

	// Update form data when compose content changes
	$effect(() => {
		setData('compose_file', composeContent);
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
					<LucideContainer class="h-6 w-6 text-primary" />
				</div>
				<div>
					<h1 class="text-2xl font-semibold text-foreground">Create Docker Compose Service</h1>
					<p class="text-sm text-muted-foreground">
						Deploy a multi-container application to {project?.name || 'your project'}
					</p>
				</div>
			</div>
		</div>

		<!-- Form -->
		<form use:form class="space-y-6">
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
							placeholder="my-compose-app"
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
									{selectedServerName}
								</Select.Trigger>
								<Select.Content>
									<Select.Group>
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
									</Select.Group>
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
					<Card.Title>Docker Compose Configuration</Card.Title>
					<Card.Description>
						Define your multi-container application using Docker Compose YAML
					</Card.Description>
				</Card.Header>
				<Card.Content class="space-y-4">
					<div class="space-y-2">
						<Label>Compose File Content *</Label>
						<div class={$errors.compose_file ? 'border-destructive' : ''}>
							<CodeEditor
								bind:value={composeContent}
								placeholder="version: '3.8'&#10;&#10;services:&#10;  app:&#10;    image: nginx:alpine&#10;    ports:&#10;      - '80:80'"
								class="min-h-[400px]"
							/>
						</div>
						{#if $errors.compose_file}
							<p class="text-sm text-destructive">{$errors.compose_file[0]}</p>
						{/if}
						<p class="text-xs text-muted-foreground">
							Paste your docker-compose.yml content here. The editor supports YAML syntax
							highlighting and validation.
						</p>
					</div>
				</Card.Content>
			</Card.Root>

			<!-- Actions -->
			<div class="flex items-center justify-end gap-4">
				<Button variant="outline" type="button" onclick={goBack}>Cancel</Button>
				<Button type="submit" disabled={$isSubmitting} class="gap-2">
					{#if $isSubmitting}
						<LucideLoader2 class="h-4 w-4 animate-spin" />
						Creating Service...
					{:else}
						<LucideContainer class="h-4 w-4" />
						Create Service
					{/if}
				</Button>
			</div>
		</form>
	{/if}
</div>
