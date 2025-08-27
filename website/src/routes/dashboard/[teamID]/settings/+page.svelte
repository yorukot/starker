<script lang="ts">
	import TeamSettingsForm from '$lib/components/settings/team-settings-form.svelte';
	import DeleteTeamDialog from '$lib/components/settings/delete-team-dialog.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import type { PageData } from './$types';
	import LucideShield from '~icons/lucide/shield';
	import LucideUsers from '~icons/lucide/users';
	import { page } from '$app/state';
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
		<h1 class="text-2xl font-semibold text-foreground">Team Settings</h1>
		<p class="text-sm text-muted-foreground">
			Manage your team's configuration and access settings
		</p>
	</div>

	<Separator />

	<!-- Team Information Section -->
	<TeamSettingsForm {team} {isOwner} />

	<!-- Team Permissions Section -->
	<Card.Root>
		<Card.Header>
			<Card.Title class="flex items-center gap-2">
				<LucideShield class="h-5 w-5" />
				Team Permissions
			</Card.Title>
			<Card.Description>Your access level and permissions within this team.</Card.Description>
		</Card.Header>
		<Card.Content class="space-y-4">
			<div class="flex items-center justify-between rounded-lg border border-border/50 p-4">
				<div class="space-y-1">
					<div class="flex items-center gap-2">
						<LucideUsers class="h-4 w-4 text-muted-foreground" />
						<span class="font-medium">Your Role</span>
					</div>
					<p class="text-sm text-muted-foreground">
						{isOwner ? 'Team Owner - Full access to all settings' : 'Team Member - Limited access'}
					</p>
				</div>
				<div class="rounded-full bg-primary/10 px-3 py-1 text-sm font-medium text-primary">
					{isOwner ? 'Owner' : 'Member'}
				</div>
			</div>

			<div class="space-y-2 rounded-lg border border-border/50 p-4">
				<h4 class="font-medium">What you can do:</h4>
				<ul class="space-y-1 text-sm text-muted-foreground">
					{#if isOwner}
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-green-500"></span>
							Modify team settings and information
						</li>
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-green-500"></span>
							Manage team members and invitations
						</li>
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-green-500"></span>
							Delete the team and all data
						</li>
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-green-500"></span>
							Full access to projects, servers, and keys
						</li>
					{:else}
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-blue-500"></span>
							View team information
						</li>
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-blue-500"></span>
							Access assigned projects and services
						</li>
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-blue-500"></span>
							Manage your own SSH keys
						</li>
						<li class="flex items-center gap-2">
							<span class="h-1 w-1 rounded-full bg-orange-500"></span>
							Cannot modify team settings (owner only)
						</li>
					{/if}
				</ul>
			</div>
		</Card.Content>
	</Card.Root>

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
					class="flex items-center justify-between rounded-lg border border-destructive/20 bg-destructive/5 p-4"
				>
					<div class="space-y-1">
						<h4 class="font-medium text-destructive">Delete Team</h4>
						<p class="text-sm text-muted-foreground">
							Permanently delete this team and all associated data. This action cannot be undone.
						</p>
					</div>
					<DeleteTeamDialog {team} {isOwner} />
				</div>
			</Card.Content>
		</Card.Root>
	{/if}
</div>
