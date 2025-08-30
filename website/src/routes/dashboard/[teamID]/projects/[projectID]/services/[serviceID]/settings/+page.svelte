<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import LucideLoader2 from '~icons/lucide/loader-2';
	import LucideSave from '~icons/lucide/save';
	import LucideTrash2 from '~icons/lucide/trash-2';
	import LucideAlertTriangle from '~icons/lucide/alert-triangle';
	import SettingsIcon from '~icons/lucide/settings';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import {
		updateServiceBasicInfoSchema,
		type UpdateServiceBasicInfoForm
	} from '$lib/schemas/request/service';
	import { authFetch, authDelete } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { toast } from 'svelte-sonner';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const service = $derived(data.service);
	const error = $derived(data.error || null);

	let showDeleteDialog = $state(false);
	let deleteConfirmName = $state('');
	let isDeleting = $state(false);

	// Basic Information Form
	const {
		form: basicInfoForm,
		errors: basicInfoErrors,
		isSubmitting: isUpdating,
		setData
	} = createForm<UpdateServiceBasicInfoForm>({
		extend: validator({ schema: updateServiceBasicInfoSchema }),
		initialValues: {
			name: '',
			description: ''
		},
		onSubmit: async (values) => {
			try {
				const response = await authFetch(
					`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/projects/${page.params.projectID}/services/${page.params.serviceID}`,
					{
						method: 'PATCH',
						headers: {
							'Content-Type': 'application/json'
						},
						body: JSON.stringify(values)
					}
				);

				if (!response.ok) {
					const errorData = await response.json();
					toast.error(errorData.message || 'Failed to update service information');
					return;
				}

				toast.success('Service information updated successfully!');
			} catch (err) {
				console.error('Error updating service information:', err);
				toast.error('Failed to update service information. Please try again.');
			}
		}
	});

	// Format date helper
	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleString();
	}

	// Delete service function
	async function deleteService() {
		if (deleteConfirmName !== service?.name) {
			toast.error('Please type the exact service name to confirm deletion');
			return;
		}

		isDeleting = true;
		try {
			const response = await authDelete(
				`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/projects/${page.params.projectID}/services/${page.params.serviceID}`
			);

			if (!response.ok) {
				const errorData = await response.json();
				toast.error(errorData.message || 'Failed to delete service');
				return;
			}

			toast.success('Service deleted successfully!');
			// Navigate back to project page after deletion
			goto(`/dashboard/${page.params.teamID}/projects/${page.params.projectID}`);
		} catch (err) {
			console.error('Error deleting service:', err);
			toast.error('Failed to delete service. Please try again.');
		} finally {
			isDeleting = false;
			showDeleteDialog = false;
		}
	}

	// Update form data when compose content changes
	$effect(() => {
		setData('name', service?.name ?? '');
		setData('description', service?.description ?? '');
	});
</script>

<div class="flex h-full flex-col gap-6 p-6">
	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else if service}
		<!-- Header -->
		<div class="flex items-center gap-3">
			<div class="rounded-lg border border-primary/20 bg-primary/10 p-3">
				<SettingsIcon class="h-6 w-6 text-primary" />
			</div>
			<div>
				<h1 class="text-2xl font-semibold text-foreground">Service Settings</h1>
				<p class="text-sm text-muted-foreground">
					Update your service's basic information and manage service configuration
				</p>
			</div>
		</div>
		<!-- Basic Information Section -->
		<form use:basicInfoForm class="space-y-6">
			<Card.Root>
				<Card.Content class="space-y-4">
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<!-- Service Name -->
						<div class="space-y-2">
							<Label for="name">Service Name *</Label>
							<Input
								id="name"
								name="name"
								type="text"
								placeholder="Enter service name"
								class={$basicInfoErrors.name ? 'border-destructive' : ''}
							/>
							{#if $basicInfoErrors.name}
								<p class="text-sm text-destructive">{$basicInfoErrors.name[0]}</p>
							{/if}
						</div>

						<!-- Service Type (Read-only) -->
						<div class="space-y-2">
							<Label>Service Type</Label>
							<div class="flex items-center gap-2">
								<Badge variant="outline" class="capitalize">
									{service.type}
								</Badge>
							</div>
							<p class="text-xs text-muted-foreground">
								Service type cannot be changed after creation
							</p>
						</div>
					</div>

					<!-- Description -->
					<div class="space-y-2">
						<Label for="description">Description</Label>
						<Textarea
							id="description"
							name="description"
							placeholder="Enter service description (optional)"
							class={$basicInfoErrors.description ? 'border-destructive' : ''}
						/>
						{#if $basicInfoErrors.description}
							<p class="text-sm text-destructive">{$basicInfoErrors.description[0]}</p>
						{/if}
						<p class="text-xs text-muted-foreground">
							Brief description of what this service does (max 500 characters)
						</p>
					</div>

					<!-- Metadata (Read-only) -->
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div class="space-y-2">
							<Label>Created</Label>
							<p class="text-sm text-muted-foreground">{formatDate(service.created_at)}</p>
						</div>
						<div class="space-y-2">
							<Label>Last Updated</Label>
							<p class="text-sm text-muted-foreground">{formatDate(service.updated_at)}</p>
						</div>
					</div>
				</Card.Content>
				<Card.Footer>
					<Button type="submit" disabled={$isUpdating} class="gap-2">
						{#if $isUpdating}
							<LucideLoader2 class="h-4 w-4 animate-spin" />
							Updating...
						{:else}
							<LucideSave class="h-4 w-4" />
							Update Information
						{/if}
					</Button>
				</Card.Footer>
			</Card.Root>
		</form>

		<!-- Danger Zone Section -->
		<Card.Root class="border-destructive">
			<Card.Header>
				<div class="flex items-center gap-2">
					<LucideAlertTriangle class="h-5 w-5 text-destructive" />
					<Card.Title class="text-destructive">Danger Zone</Card.Title>
				</div>
				<Card.Description>Irreversible and destructive actions for this service</Card.Description>
			</Card.Header>
			<Card.Content class="space-y-4">
				<div class="rounded-lg border border-destructive/20 bg-destructive/5 p-4">
					<h4 class="mb-2 font-medium text-destructive">Delete Service</h4>
					<p class="mb-3 text-sm text-muted-foreground">
						Once you delete a service, there is no going back. This will permanently delete the
						service, stop all containers, and remove all associated data.
					</p>
					<Button variant="destructive" onclick={() => (showDeleteDialog = true)} class="gap-2">
						<LucideTrash2 class="h-4 w-4" />
						Delete Service
					</Button>
				</div>
			</Card.Content>
		</Card.Root>
	{/if}
</div>

<!-- Delete Confirmation Dialog -->
<AlertDialog.Root bind:open={showDeleteDialog}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title class="flex items-center gap-2 text-destructive">
				<LucideAlertTriangle class="h-5 w-5" />
				Delete Service
			</AlertDialog.Title>
			<AlertDialog.Description>
				This action cannot be undone. This will permanently delete the service
				<strong class="font-medium">"{service?.name}"</strong>
				and stop all associated containers.
			</AlertDialog.Description>
		</AlertDialog.Header>
		<div class="py-4">
			<Label for="confirm-name" class="text-sm font-medium">
				Type <strong>{service?.name}</strong> to confirm deletion:
			</Label>
			<Input
				id="confirm-name"
				bind:value={deleteConfirmName}
				placeholder="Enter service name"
				class="mt-2"
			/>
		</div>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action
				onclick={deleteService}
				disabled={isDeleting || deleteConfirmName !== service?.name}
				class="gap-2"
			>
				{#if isDeleting}
					<LucideLoader2 class="h-4 w-4 animate-spin" />
					Deleting...
				{:else}
					<LucideTrash2 class="h-4 w-4" />
					Delete Service
				{/if}
			</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
