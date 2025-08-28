<script lang="ts">
	import TeamSettingsForm from '$lib/components/settings/team-settings-form.svelte';
	import DeleteTeamDialog from '$lib/components/settings/delete-team-dialog.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import type { PageData } from './$types';
	import { invalidate } from '$app/navigation';

	let { data }: { data: PageData } = $props();

	const team = $derived(data.team);
	const isOwner = $derived(data.isOwner);
	const user = $derived(data.user);

	// Function to refresh team data after updates
	async function refreshTeamData() {
		await invalidate(`team:${team.id}`);
		await invalidate('team:current');
	}
</script>

<svelte:head>
	<title>Settings - {team.name} | Starker</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-2">
		<h1 class="text-foreground text-2xl font-semibold">Team Settings</h1>
		<p class="text-muted-foreground text-sm">
			Manage your team's configuration and access settings
		</p>
	</div>

	<Separator />

	<!-- Team Information Section -->
	<TeamSettingsForm {team} {isOwner} />

	<!-- Dangerous Actions Section (Owner Only) -->
	{#if isOwner}
		<Card.Root class="border-destructive/20">
			<Card.Header>
				<Card.Title class="text-destructive">Dangerous Actions</Card.Title>
				<Card.Description>
					These actions are irreversible and will permanently affect your team.
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<div
					class="border-destructive/20 bg-destructive/5 flex items-center justify-between rounded-lg border p-4"
				>
					<div class="space-y-1">
						<h4 class="text-destructive font-medium">Delete Team</h4>
						<p class="text-muted-foreground text-sm">
							Permanently delete this team and all associated data. This action cannot be undone.
						</p>
					</div>
					<DeleteTeamDialog {team} {isOwner} />
				</div>
			</Card.Content>
		</Card.Root>
	{/if}
</div>
