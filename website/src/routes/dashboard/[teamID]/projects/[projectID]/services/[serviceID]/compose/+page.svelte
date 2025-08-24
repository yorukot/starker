<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import LucideLoader2 from '~icons/lucide/loader-2';
	import LucideSave from '~icons/lucide/save';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import {
		updateServiceComposeSchema,
		type UpdateServiceComposeForm
	} from '$lib/schemas/request/service';
	import { authFetch } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { toast } from 'svelte-sonner';
	import { CodeEditor } from '$lib/components/ui/code-editor';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const composeConfig = $derived(data.composeConfig);
	const error = $derived(data.error || null);

	let composeContent = $state('');

	// Initialize compose content when data loads
	$effect(() => {
		if (composeConfig?.compose_file) {
			composeContent = composeConfig.compose_file;
		}
	});

	const { form, errors, isSubmitting, setData } = createForm<UpdateServiceComposeForm>({
		extend: validator({ schema: updateServiceComposeSchema }),
		initialValues: {
			compose_file: ''
		},
		onSubmit: async () => {
			try {
				const response = await authFetch(
					`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/projects/${page.params.projectID}/services/${page.params.serviceID}/compose`,
					{
						method: 'PATCH',
						headers: {
							'Content-Type': 'application/json'
						},
						body: JSON.stringify({
							compose_file: composeContent
						})
					}
				);

				if (!response.ok) {
					const errorData = await response.json();
					toast.error(errorData.message || 'Failed to update compose configuration');
					return;
				}

				toast.success('Compose configuration updated successfully!');
			} catch (err) {
				console.error('Error updating compose configuration:', err);
				toast.error('Failed to update compose configuration. Please try again.');
			}
		}
	});

	function goBack() {
		goto(
			`/dashboard/${page.params.teamID}/projects/${page.params.projectID}/services/${page.params.serviceID}`
		);
	}

	// Update form data when compose content changes
	$effect(() => {
		setData('compose_file', composeContent);
	});
</script>

<div class="mt-6 flex flex-col gap-6">
	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else}
		<!-- Form -->
		<form use:form class="space-y-6">
			<Card.Root>
				<Card.Header>
					<Card.Title>Docker Compose Configuration</Card.Title>
					<Card.Description>
						Edit your multi-container application configuration using Docker Compose YAML
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
							Edit your docker-compose.yml content here. The editor supports YAML syntax
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
						Updating...
					{:else}
						<LucideSave class="h-4 w-4" />
						Update Compose
					{/if}
				</Button>
			</div>
		</form>
	{/if}
</div>
