<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { createServerSchema, type CreateServerForm } from '$lib/schemas/request/server';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { authPost } from '$lib/api/client';
	import { toast } from 'svelte-sonner';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const privateKeys = data.privateKeys || [];
	const error = $derived(data.error || null);

	let serverError = $state('');
	const {
		form,
		errors,
		isSubmitting,
		data: formData
	} = createForm<CreateServerForm>({
		extend: validator({ schema: createServerSchema }),
		initialValues: {
			name: '',
			description: '',
			ip: '',
			port: '22',
			user: '',
			private_key_id: ''
		},
		onSubmit: async (values) => {
			serverError = '';

			const response = await authPost(
				`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/servers`,
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
			toast.success('Server created successfully');
			goto(`/dashboard/${page.params.teamID}/servers`);
		},
		onError: (error: unknown) => {
			console.error('Server creation error:', error);
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Failed to create server. Please try again.';
			}
			toast.error('Failed to create server', {
				description: serverError || 'An unexpected error occurred.'
			});
		}
	});

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
	{:else}
		<!-- Header with title -->
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-foreground">Add New Server</h1>
			<p class="text-sm text-muted-foreground">Configure a new server with SSH authentication</p>
		</div>

		<!-- Creation Error Alert -->
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
						{$formData.private_key_id
							? privateKeys.find((k) => k.id === $formData.private_key_id)?.name
							: 'Select an SSH key'}
					</Select.Trigger>
					<Select.Content>
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
					</Select.Content>
				</Select.Root>
				<input type="hidden" name="private_key_id" value={$formData.private_key_id} />
				{#if $errors.private_key_id}
					<span class="text-sm text-destructive">{$errors.private_key_id[0]}</span>
				{/if}
				{#if privateKeys.length === 0}
					<p class="text-sm text-muted-foreground">
						No SSH keys available.
						<a href="/dashboard/{page.params.teamID}/keys" class="text-primary hover:underline">
							Create an SSH key first
						</a>
					</p>
				{/if}
			</div>

			<!-- Submit Button -->
			<div class="flex justify-end">
				<Button
					type="submit"
					disabled={$isSubmitting || privateKeys.length === 0}
					class="w-full md:w-auto"
				>
					{$isSubmitting ? 'Creating...' : 'Create Server'}
				</Button>
			</div>
		</form>
	{/if}
</div>
