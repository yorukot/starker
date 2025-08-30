<script lang="ts">
	import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import LucideTrash2 from '~icons/lucide/trash-2';
	import LucideAlertTriangle from '~icons/lucide/alert-triangle';
	import { deleteTeamSchema, type DeleteTeamForm } from '$lib/schemas/request/team-settings';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import type { Team } from '$lib/schemas/team';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { authDelete } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';

	let { team, isOwner }: { team: Team; isOwner: boolean } = $props();

	let dialogOpen = $state(false);
	let serverError = $state('');

	const {
		form: deleteForm,
		errors: deleteErrors,
		isSubmitting: isDeleting,
		reset
	} = createForm<DeleteTeamForm>({
		extend: validator({
			schema: deleteTeamSchema,
			castValues: false
		}),
		validate: (values) => {
			// Add team name to context for validation
			return deleteTeamSchema.validateSync(values, {
				context: { teamName: team.name }
			});
		},
		onSubmit: async (values) => {
			serverError = '';

			try {
				const response = await authDelete(`${PUBLIC_API_BASE_URL}/teams/${team.id}`);

				if (!response.ok) {
					const errorData = await response.json();
					throw errorData;
				}

				toast.success('Team deleted successfully');

				// Redirect to dashboard after successful deletion
				await goto('/dashboard');
			} catch (error) {
				console.error('Delete error:', error);
				if (error && typeof error === 'object' && 'message' in error) {
					serverError = (error as { message: string }).message;
				} else {
					serverError = 'Failed to delete team. Please try again.';
				}
				toast.error('Failed to delete team');
			}
		}
	});

	function openDialog() {
		if (!isOwner) return;
		dialogOpen = true;
		reset();
		serverError = '';
	}

	function closeDialog() {
		dialogOpen = false;
		reset();
		serverError = '';
	}
</script>

{#if isOwner}
	<Dialog.Root bind:open={dialogOpen}>
		<Dialog.Trigger class={buttonVariants({ variant: 'destructive' })} onclick={openDialog}>
			<LucideTrash2 class="h-4 w-4" />
			Delete Team
		</Dialog.Trigger>
		<Dialog.Content class="max-w-md">
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2 text-destructive">
					<LucideAlertTriangle class="h-5 w-5" />
					Delete Team
				</Dialog.Title>
				<Dialog.Description>
					This action cannot be undone. This will permanently delete the team and all associated
					data.
				</Dialog.Description>
			</Dialog.Header>

			<div class="space-y-4">
				<Alert.Root class="border-destructive/50">
					<LucideAlertTriangle class="h-4 w-4 text-destructive" />
					<Alert.Title class="text-destructive">Warning</Alert.Title>
					<Alert.Description>
						Deleting this team will:
						<ul class="mt-2 list-inside list-disc space-y-1 text-sm">
							<li>Remove all team members</li>
							<li>Delete all projects and services</li>
							<li>Remove all servers and SSH keys</li>
							<li>This action is irreversible</li>
						</ul>
					</Alert.Description>
				</Alert.Root>

				<form use:deleteForm class="space-y-4">
					{#if serverError}
						<div
							class="text-destructive-foreground rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm"
						>
							{serverError}
						</div>
					{/if}

					<div class="space-y-2">
						<Label for="confirmText" class="text-sm font-medium">
							To confirm, type <span class="font-mono font-semibold">"{team.name}"</span> below:
						</Label>
						<Input
							id="confirmText"
							name="confirmText"
							type="text"
							placeholder="Type team name here"
							class={$deleteErrors.confirmText ? 'border-destructive' : ''}
							disabled={$isDeleting}
						/>
						{#if $deleteErrors.confirmText}
							<span class="text-sm text-destructive">{$deleteErrors.confirmText[0]}</span>
						{/if}
					</div>

					<div class="flex justify-end gap-2">
						<Dialog.Close
							class={buttonVariants({ variant: 'outline' })}
							onclick={closeDialog}
							disabled={$isDeleting}
						>
							Cancel
						</Dialog.Close>
						<Button type="submit" variant="destructive" disabled={$isDeleting}>
							{$isDeleting ? 'Deleting...' : 'Delete Team'}
						</Button>
					</div>
				</form>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}
