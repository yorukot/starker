<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import LucideServer from '~icons/lucide/server';
	import LucideTrash2 from '~icons/lucide/trash-2';
	import { page } from '$app/state';
	import { goto, invalidate } from '$app/navigation';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { updateServerSchema, type UpdateServerForm } from '$lib/schemas/request/server';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { authPatch, authDelete } from '$lib/api/client';
	import { toast } from 'svelte-sonner';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const server = $derived(data.server);
	const privateKeys = data.privateKeys || [];
	const error = $derived(data.error || null);

	let serverError = $state('');
	let deleteConfirmation = $state('');
	let showDeleteForm = $state(false);

	const {
		form,
		errors,
		isSubmitting,
		data: formData
	} = createForm<UpdateServerForm>({
		extend: validator({ schema: updateServerSchema }),
		initialValues: {
			name: data.server?.name || '',
			description: data.server?.description || '',
			ip: data.server?.ip || '',
			port: data.server?.port || '22',
			user: data.server?.user || '',
			private_key_id: data.server?.private_key_id || ''
		},
		onSubmit: async (values) => {
			const currentServer = server;
			if (!currentServer) return;

			serverError = '';

			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/servers/${currentServer.id}`,
				{
					name: values.name.trim(),
					description: values.description?.trim() || '',
					ip: values.ip.trim(),
					port: values.port.trim(),
					user: values.user.trim(),
					private_key_id: values.private_key_id
				}
			);

			if (!response.ok) {
				const errorData = await response.json();
				throw errorData;
			}

			return response.json();
		},
		onSuccess: () => {
			invalidate(`server:${data.server?.id}`);
			toast.success('Server updated successfully');
		},
		onError: (error: unknown) => {
			console.error('Update error:', error);
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Failed to update server. Please try again.';
			}
			toast.error('Failed to update server', {
				description: serverError || 'An unexpected error occurred.'
			});
		}
	});

	// Computed trigger content for select
	const selectedKeyName = $derived(
		$formData.private_key_id
			? (privateKeys.find((k) => k.id === $formData.private_key_id)?.name ?? 'Select an SSH key')
			: 'Select an SSH key'
	);

	async function deleteServer() {
		if (!server || deleteConfirmation !== server.name) {
			toast.error('Please type the server name to confirm deletion');
			return;
		}

		try {
			const response = await authDelete(
				`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/servers/${server.id}`
			);

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.message || 'Failed to delete server');
			}

			toast.success('Server deleted successfully');
			goto(`/dashboard/${page.params.teamID}/servers`);
		} catch (error) {
			console.error('Delete error:', error);
			toast.error('Failed to delete server', {
				description: error instanceof Error ? error.message : 'An unexpected error occurred.'
			});
		}
	}

	function goBack() {
		goto(`/dashboard/${page.params.teamID}/servers`);
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<Button variant="ghost" size="sm" onclick={goBack} class="gap-2">
			<LucideArrowLeft class="h-4 w-4" />
			Back to Servers
		</Button>
	</div>

	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else if server}
		<!-- Header with title -->
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-foreground">Server Settings</h1>
			<p class="text-sm text-muted-foreground">
				Configure server connection details and SSH authentication
			</p>
		</div>

		<!-- Update Error Alert -->
		{#if serverError}
			<Alert.Root variant="destructive">
				<Alert.Description>
					{serverError}
				</Alert.Description>
			</Alert.Root>
		{/if}

		<!-- Form -->
		<form use:form class="space-y-6">
			<!-- Name and User Row -->
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
				<div class="space-y-2">
					<Label for="name">Server Name *</Label>
					<Input
						id="name"
						name="name"
						type="text"
						placeholder="My Production Server"
						required
						class={$errors.name ? 'border-destructive' : ''}
					/>
					{#if $errors.name}
						<span class="text-sm text-destructive">{$errors.name[0]}</span>
					{/if}
				</div>
				<div class="space-y-2">
					<Label for="user">Username *</Label>
					<Input
						id="user"
						name="user"
						type="text"
						placeholder="root"
						required
						class={$errors.user ? 'border-destructive' : ''}
					/>
					{#if $errors.user}
						<span class="text-sm text-destructive">{$errors.user[0]}</span>
					{/if}
				</div>
			</div>

			<!-- Description -->
			<div class="space-y-2">
				<Label for="description">Description</Label>
				<Textarea
					id="description"
					name="description"
					placeholder="Production web server (optional)"
					class="min-h-[40px] {$errors.description ? 'border-destructive' : ''}"
				/>
				{#if $errors.description}
					<span class="text-sm text-destructive">{$errors.description[0]}</span>
				{/if}
			</div>

			<!-- IP and Port Row -->
			<div class="grid grid-cols-1 gap-6 md:grid-cols-3">
				<div class="col-span-2 space-y-2">
					<Label for="ip">IP Address / Hostname *</Label>
					<Input
						id="ip"
						name="ip"
						type="text"
						placeholder="192.168.1.100 or server.example.com"
						required
						class={$errors.ip ? 'border-destructive' : ''}
					/>
					{#if $errors.ip}
						<span class="text-sm text-destructive">{$errors.ip[0]}</span>
					{/if}
				</div>
				<div class="space-y-2">
					<Label for="port">Port *</Label>
					<Input
						id="port"
						name="port"
						type="number"
						placeholder="22"
						required
						class={$errors.port ? 'border-destructive' : ''}
					/>
					{#if $errors.port}
						<span class="text-sm text-destructive">{$errors.port[0]}</span>
					{/if}
				</div>
			</div>

			<!-- SSH Key Selection -->
			<div class="space-y-2">
				<Label for="private_key_id">SSH Key *</Label>
				<Select.Root type="single" bind:value={$formData.private_key_id}>
					<Select.Trigger class="w-full {$errors.private_key_id ? 'border-destructive' : ''}">
						{selectedKeyName}
					</Select.Trigger>
					<Select.Content>
						<Select.Group>
							{#each privateKeys as key (key.id)}
								<Select.Item value={key.id} label={key.name}>
									<div class="flex flex-col">
										<div class="font-medium">{key.name}</div>
										{#if key.description}
											<div class="text-xs text-muted-foreground">{key.description}</div>
										{/if}
									</div>
								</Select.Item>
							{/each}
						</Select.Group>
					</Select.Content>
				</Select.Root>
				<input type="hidden" name="private_key_id" value={$formData.private_key_id} />
				{#if $errors.private_key_id}
					<span class="text-sm text-destructive">{$errors.private_key_id[0]}</span>
				{/if}
			</div>

			<!-- Save Button -->
			<div class="flex justify-end">
				<Button type="submit" disabled={$isSubmitting} class="w-full md:w-auto">
					{$isSubmitting ? 'Saving...' : 'Save Changes'}
				</Button>
			</div>
		</form>

		<!-- Metadata -->
		<div class="grid grid-cols-1 gap-6 border-t pt-4 md:grid-cols-2">
			<div class="text-sm">
				<span class="font-medium text-muted-foreground">Created:</span>
				<span class="ml-2">{new Date(server.created_at).toLocaleString()}</span>
			</div>
			{#if server.updated_at !== server.created_at}
				<div class="text-sm">
					<span class="font-medium text-muted-foreground">Last Updated:</span>
					<span class="ml-2">{new Date(server.updated_at).toLocaleString()}</span>
				</div>
			{/if}
		</div>

		<!-- Danger Zone -->
		<div class="space-y-4 rounded-lg border border-destructive/20 bg-destructive/5 p-6">
			<div>
				<h3 class="text-lg font-medium text-destructive">Danger Zone</h3>
				<p class="text-sm text-muted-foreground">
					Once you delete this server, there is no going back. Please be certain.
				</p>
			</div>

			{#if !showDeleteForm}
				<Button variant="destructive" onclick={() => (showDeleteForm = true)} class="gap-2">
					<LucideTrash2 class="h-4 w-4" />
					Delete Server
				</Button>
			{:else}
				<div class="space-y-4">
					<div class="space-y-2">
						<Label for="delete-confirmation">
							Type <code class="rounded bg-muted px-1 py-0.5 text-sm">{server.name}</code> to confirm
							deletion:
						</Label>
						<Input
							id="delete-confirmation"
							type="text"
							bind:value={deleteConfirmation}
							placeholder="Type server name to confirm"
							class="max-w-md"
						/>
					</div>
					<div class="flex gap-2">
						<Button
							variant="destructive"
							onclick={deleteServer}
							disabled={deleteConfirmation !== server.name}
							class="gap-2"
						>
							<LucideTrash2 class="h-4 w-4" />
							Delete Server
						</Button>
						<Button
							variant="outline"
							onclick={() => {
								showDeleteForm = false;
								deleteConfirmation = '';
							}}
						>
							Cancel
						</Button>
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<div class="flex w-full flex-col items-center justify-center gap-6 py-20">
			<div class="rounded-full border border-muted bg-muted/30 p-6">
				<LucideServer class="h-16 w-16 text-muted-foreground/50" />
			</div>
			<div class="space-y-2 text-center">
				<h3 class="text-lg font-medium text-foreground">Loading server...</h3>
			</div>
		</div>
	{/if}
</div>
