<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import LucidePlus from '~icons/lucide/plus';
	import { teamSchema, type TeamForm } from '$lib/schemas/request/team';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { goto } from '$app/navigation';
	import { authPost } from '$lib/api/client.js';
	import type { Team } from '$lib/schemas/team';

	let serverError = '';

	const { form, errors, isSubmitting } = createForm<TeamForm>({
		extend: validator({ schema: teamSchema }),
		onSubmit: async (values) => {
			serverError = '';

			const response = await authPost(`${PUBLIC_API_BASE_URL}/teams`, {
				name: values.team_name
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw errorData;
			}

			return response.json();
		},
		onSuccess: (data: unknown) => {
			const team = data as Team;
			goto(`/dashboard/${team.id}/projects`);
		},
		onError: (error: unknown) => {
			console.error('Team creation error:', error);
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Failed to create team. Please try again.';
			}
		}
	});
</script>

<form use:form>
	<div class="grid gap-4">
		{#if serverError}
			<div
				class="text-destructive-foreground rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm"
			>
				{serverError}
			</div>
		{/if}

		<div class="grid gap-2">
			<Label for="team_name">Team Name</Label>
			<Input
				id="team_name"
				name="team_name"
				type="text"
				placeholder="My Awesome Team"
				class={$errors.team_name ? 'border-destructive' : ''}
			/>
			{#if $errors.team_name}
				<span class="text-sm text-destructive">{$errors.team_name[0]}</span>
			{/if}
		</div>

		<Button type="submit" class="w-full" disabled={$isSubmitting}>
			<LucidePlus />
			{$isSubmitting ? 'Creating Team...' : 'Create Team'}
		</Button>
	</div>
</form>
