<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import LucideSave from '~icons/lucide/save';
	import { teamSettingsSchema, type TeamSettingsForm } from '$lib/schemas/request/team-settings';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import type { Team } from '$lib/schemas/team';
	import { toast } from 'svelte-sonner';
	import { authPatch } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { page } from '$app/state';
	import { invalidate } from '$app/navigation';

	let { team, isOwner }: { team: Team; isOwner: boolean } = $props();

	let serverError = $state('');

	const {
		form: updateForm,
		errors: updateErrors,
		isSubmitting: isUpdating,
		setFields
	} = createForm<TeamSettingsForm>({
		extend: validator({ schema: teamSettingsSchema }),
		initialValues: {
			name: team.name
		},
		onSubmit: async (values) => {
			// Clear any previous server errors
			serverError = '';

			try {
				const response = await authPatch(`${PUBLIC_API_BASE_URL}/teams/${team.id}`, {
					name: values.name
				});

				if (!response.ok) {
					const errorData = await response.json();
					throw errorData;
				}

				// Success - invalidate team data and show success message
				await invalidate(`team:${team.id}`);
				await invalidate('team:current');
				toast.success('Team settings updated successfully!');

				// Update form fields to new values
				setFields({
					name: values.name
				});
			} catch (error) {
				console.error('Update error:', error);
				if (error && typeof error === 'object' && 'message' in error) {
					serverError = (error as { message: string }).message;
				} else {
					serverError = 'Failed to update team settings. Please try again.';
				}
				toast.error('Failed to update team settings');
			}
		}
	});
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Team Information</Card.Title>
		<Card.Description>Manage your team's basic information and settings.</Card.Description>
	</Card.Header>
	<Card.Content class="space-y-6">
		<form use:updateForm class="space-y-4">
			{#if serverError}
				<div
					class="text-destructive-foreground rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm"
				>
					{serverError}
				</div>
			{/if}

			<div class="grid gap-2">
				<Label for="name">Team Name</Label>
				<Input
					id="name"
					name="name"
					type="text"
					disabled={!isOwner}
					class={$updateErrors.name ? 'border-destructive' : ''}
					placeholder="Enter team name"
				/>
				{#if $updateErrors.name}
					<span class="text-sm text-destructive">{$updateErrors.name[0]}</span>
				{/if}
			</div>

			{#if isOwner}
				<div class="flex justify-end">
					<Button type="submit" disabled={$isUpdating}>
						<LucideSave class="h-4 w-4" />
						{$isUpdating ? 'Updating...' : 'Update Team'}
					</Button>
				</div>
			{:else}
				<div class="rounded-md bg-muted/30 p-3 text-sm text-muted-foreground">
					Only team owners can modify team settings.
				</div>
			{/if}
		</form>

		<div class="grid gap-4 pt-4">
			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-1">
					<Label class="text-sm font-medium">Team ID</Label>
					<p class="font-mono text-sm text-muted-foreground">{team.id}</p>
				</div>
				<div class="space-y-1">
					<Label class="text-sm font-medium">Owner ID</Label>
					<p class="font-mono text-sm text-muted-foreground">{team.owner_id}</p>
				</div>
			</div>
			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-1">
					<Label class="text-sm font-medium">Created</Label>
					<p class="text-sm text-muted-foreground">
						{new Date(team.created_at).toLocaleDateString('en-US', {
							year: 'numeric',
							month: 'long',
							day: 'numeric',
							hour: '2-digit',
							minute: '2-digit'
						})}
					</p>
				</div>
				<div class="space-y-1">
					<Label class="text-sm font-medium">Last Updated</Label>
					<p class="text-sm text-muted-foreground">
						{new Date(team.updated_at).toLocaleDateString('en-US', {
							year: 'numeric',
							month: 'long',
							day: 'numeric',
							hour: '2-digit',
							minute: '2-digit'
						})}
					</p>
				</div>
			</div>
		</div>
	</Card.Content>
</Card.Root>
