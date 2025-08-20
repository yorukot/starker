<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import DeleteKeyDialog from '$lib/components/key/delete-key-dialog.svelte';
	import type { PageData } from './$types';
	import LucideKeyRound from '~icons/lucide/key-round';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import {
		updatePrivateKeySchema,
		type UpdatePrivateKeyForm
	} from '$lib/schemas/request/privatekey';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { authPatch } from '$lib/api/client';
	import { toast } from "svelte-sonner";

	let { data }: { data: PageData } = $props();

	const privateKey = data.privateKey;
	const error = $derived(data.error || null);

	let serverError = $state('');

	const { form, errors, isSubmitting } = createForm<UpdatePrivateKeyForm>({
		extend: validator({ schema: updatePrivateKeySchema }),
		initialValues: {
			name: privateKey?.name || '',
			description: privateKey?.description || '',
			privateKey: privateKey?.private_key || ''
		},
		onSubmit: async (values) => {
			const currentKey = privateKey;
			if (!currentKey) return;

			serverError = '';

			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/private-keys/${currentKey.id}`,
				{
					name: values.name.trim(),
					description: values.description?.trim() || '',
					private_key: values.privateKey?.trim() || ''
				}
			);

			if (!response.ok) {
				const errorData = await response.json();
				throw errorData;
			}

			return response.json();
		},
		onSuccess: () => {
			location.reload();
			toast.success('Successful update key');
		},
		onError: (error: unknown) => {
			console.error('Update error:', error);
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Failed to update key. Please try again.';
			}
			toast.error('Failed to update keys', {
				description: serverError || 'An unexpected error occurred.'
			});
		}
	});

	function goBack() {
		goto(`/dashboard/${page.params.teamID}/keys`);
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<Button variant="ghost" size="sm" onclick={goBack} class="gap-2">
			<LucideArrowLeft class="h-4 w-4" />
			Back to Keys
		</Button>
		<DeleteKeyDialog {privateKey} />
	</div>

	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else if privateKey}
		<!-- Header with title -->
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-foreground">Private Key</h1>
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
			<!-- Name and Description Row -->
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
				<div class="space-y-2">
					<Label for="name">Name *</Label>
					<Input
						id="name"
						name="name"
						type="text"
						placeholder="Enter key name"
						required
						class={$errors.name ? 'border-destructive' : ''}
					/>
					{#if $errors.name}
						<span class="text-sm text-destructive">{$errors.name[0]}</span>
					{/if}
				</div>
				<div class="space-y-2">
					<Label for="description">Description</Label>
					<Textarea
						id="description"
						name="description"
						placeholder="Enter key description (optional)"
						class="min-h-[40px] {$errors.description ? 'border-destructive' : ''}"
					/>
					{#if $errors.description}
						<span class="text-sm text-destructive">{$errors.description[0]}</span>
					{/if}
				</div>
			</div>

			<!-- Fingerprint Section -->
			<div class="space-y-2">
				<label class="text-sm font-medium text-foreground" for="public-key">Fingerprint</label>
				<div class="relative">
					<div
						id="public-key"
						class="rounded-md border border-input bg-background px-3 py-2 font-mono text-sm break-all text-muted-foreground"
					>
						{privateKey.fingerprint || 'No public key available'}
					</div>
				</div>
			</div>

			<!-- Private Key Section -->
			<div class="space-y-2">
				<div class="flex items-center justify-between">
					<label class="text-sm font-medium text-foreground" for="private-key">Private Key *</label>
				</div>
				<div class="relative">
					<Textarea
						id="privateKey"
						name="privateKey"
						placeholder="Paste your private key here"
						class="min-h-[120px] font-mono text-xs {$errors.privateKey ? 'border-destructive' : ''}"
						required
					/>
					{#if $errors.privateKey}
						<span class="text-sm text-destructive">{$errors.privateKey[0]}</span>
					{/if}
				</div>
				{#if $errors.privateKey}
					<span class="text-sm text-destructive">{$errors.privateKey[0]}</span>
				{/if}

				<!-- Save Button -->
				<div class="flex justify-end">
					<Button type="submit" disabled={$isSubmitting} class="w-full md:w-auto">
						{$isSubmitting ? 'Saving...' : 'Save Changes'}
					</Button>
				</div>
			</div>
		</form>

		<!-- Metadata -->
		<div class="grid grid-cols-1 gap-6 border-t pt-4 md:grid-cols-2">
			<div class="text-sm">
				<span class="font-medium text-muted-foreground">Created:</span>
				<span class="ml-2">{new Date(privateKey.created_at).toLocaleString()}</span>
			</div>
			{#if privateKey.updated_at !== privateKey.created_at}
				<div class="text-sm">
					<span class="font-medium text-muted-foreground">Last Updated:</span>
					<span class="ml-2">{new Date(privateKey.updated_at).toLocaleString()}</span>
				</div>
			{/if}
		</div>
	{:else}
		<div class="flex w-full flex-col items-center justify-center gap-6 py-20">
			<div class="rounded-full border border-muted bg-muted/30 p-6">
				<LucideKeyRound class="h-16 w-16 text-muted-foreground/50" />
			</div>
			<div class="space-y-2 text-center">
				<h3 class="text-lg font-medium text-foreground">Loading key...</h3>
			</div>
		</div>
	{/if}
</div>
