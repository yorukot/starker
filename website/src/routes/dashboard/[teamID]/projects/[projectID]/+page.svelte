<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import DeleteProjectDialog from '$lib/components/project/delete-project-dialog.svelte';
	import EditProjectDialog from '$lib/components/project/edit-project-dialog.svelte';
	import type { PageData } from './$types';
	import LucideFolder from '~icons/lucide/folder';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import LucideSettings from '~icons/lucide/settings';
	import { page } from '$app/state';
	import { goto, invalidate } from '$app/navigation';

	let { data }: { data: PageData } = $props();

	const project = $derived(data.project);
	const error = $derived(data.error || null);

	function goBack() {
		goto(`/dashboard/${page.params.teamID}/projects`);
	}

	async function onProjectUpdated() {
		await invalidate(`project:${page.params.projectID}`);
	}

	function onProjectDeleted() {
		goto(`/dashboard/${page.params.teamID}/projects`);
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<Button variant="ghost" size="sm" onclick={goBack} class="gap-2">
			<LucideArrowLeft class="h-4 w-4" />
			Back to Projects
		</Button>
		{#if project}
			<div class="flex gap-2">
				<EditProjectDialog {project} {onProjectUpdated} />
				<DeleteProjectDialog {project} onDeleted={onProjectDeleted} />
			</div>
		{/if}
	</div>

	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else if project}
		<!-- Header with project info -->
		<div class="flex items-center gap-4">
			<div class="rounded-lg border border-primary/20 bg-primary/10 p-3">
				<LucideFolder class="h-8 w-8 text-primary" />
			</div>
			<div>
				<h1 class="text-2xl font-semibold text-foreground">{project.name}</h1>
				{#if project.description}
					<p class="text-muted-foreground">{project.description}</p>
				{/if}
			</div>
		</div>

		<Separator />

		<!-- Services Section -->
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<div class="flex flex-col gap-2">
					<h2 class="text-xl font-semibold text-foreground">Services</h2>
					<p class="text-sm text-muted-foreground">
						Manage your project's services and deployments
					</p>
				</div>
				<!-- Future: Add service creation button here -->
			</div>

			<!-- Empty state for services -->
			<div class="flex w-full flex-col items-center justify-center gap-6 py-20">
				<div class="rounded-full border border-muted bg-muted/30 p-6">
					<LucideSettings class="h-16 w-16 text-muted-foreground/50" />
				</div>
				<div class="space-y-2 text-center">
					<h3 class="text-lg font-medium text-foreground">No services yet</h3>
					<p class="max-w-sm text-sm text-muted-foreground">
						Services will be displayed here once they are created for this project.
					</p>
				</div>
			</div>
		</div>

		<!-- Project metadata -->
		<div class="grid grid-cols-1 gap-6 border-t pt-4 md:grid-cols-2">
			<div class="text-sm">
				<span class="font-medium text-muted-foreground">Created:</span>
				<span class="ml-2">{new Date(project.created_at).toLocaleString()}</span>
			</div>
			{#if project.updated_at !== project.created_at}
				<div class="text-sm">
					<span class="font-medium text-muted-foreground">Last Updated:</span>
					<span class="ml-2">{new Date(project.updated_at).toLocaleString()}</span>
				</div>
			{/if}
		</div>
	{:else}
		<div class="flex w-full flex-col items-center justify-center gap-6 py-20">
			<div class="rounded-full border border-muted bg-muted/30 p-6">
				<LucideFolder class="h-16 w-16 text-muted-foreground/50" />
			</div>
			<div class="space-y-2 text-center">
				<h3 class="text-lg font-medium text-foreground">Loading project...</h3>
			</div>
		</div>
	{/if}
</div>