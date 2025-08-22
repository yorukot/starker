<script lang="ts">
	import NewProject from '$lib/components/project/new-project.svelte';
	import ProjectActionsDropdown from '$lib/components/project/project-actions-dropdown.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import type { PageData } from './$types';
	import LucideFolder from '~icons/lucide/folder';
	import LucideTimer from '~icons/lucide/timer';
	import { page } from '$app/state';
	import { invalidate } from '$app/navigation';

	let { data }: { data: PageData } = $props();

	const projects = $derived(data.projects || []);

	async function onProjectCreated() {
		await invalidate(`projects:${page.params.teamID}`);
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<div class="flex flex-col gap-2">
			<h1 class="text-2xl font-semibold text-foreground">Projects</h1>
			<p class="text-sm text-muted-foreground">
				Organize your services and deployments into projects
			</p>
		</div>
		<NewProject {onProjectCreated} />
	</div>

	<Separator />

	{#if projects.length > 0}
		<div class="grid grid-cols-2 gap-4 lg:grid-cols-3 2xl:grid-cols-4">
			{#each projects as project (project.id)}
				<a href="/dashboard/{page.params.teamID}/projects/{project.id}" class="block h-full">
					<Card.Root class="border border-border/50 hover:bg-card bg-card/50 transition-colors hover:border-border cursor-pointer group h-full">
						<Card.Header>
							<div class="flex flex-col space-y-3">
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-2">
										<div class="rounded-lg border border-primary/20 bg-primary/10 p-2">
											<LucideFolder class="h-5 w-5 text-primary" />
										</div>
										<h3 class="font-medium">{project.name}</h3>
									</div>
									<div 
										class="opacity-0 group-hover:opacity-100 transition-opacity" 
										onclick={(e) => e.preventDefault()}
										onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') e.preventDefault(); }}
										role="button"
										tabindex="-1"
									>
										<ProjectActionsDropdown {project} />
									</div>
								</div>
							</div>
						</Card.Header>
						<Card.Content>
							<div class="space-y-2">
								{#if project.description}
									<p class="text-sm text-muted-foreground line-clamp-2">
										{project.description}
									</p>
								{/if}
								<div class="space-y-1 text-xs text-muted-foreground">
									<div class="flex gap-1 items-center">
										<LucideTimer class="h-3 w-3" /> 
										Created: {new Date(project.created_at).toLocaleDateString()}
									</div>
								</div>
							</div>
						</Card.Content>
					</Card.Root>
				</a>
			{/each}
		</div>
	{:else}
		<div class="flex w-full flex-col items-center justify-center gap-6 py-20">
			<div class="rounded-full border border-muted bg-muted/30 p-6">
				<LucideFolder class="h-16 w-16 text-muted-foreground/50" />
			</div>
			<div class="space-y-2 text-center">
				<h3 class="text-lg font-medium text-foreground">No projects yet</h3>
				<p class="max-w-sm text-sm text-muted-foreground">
					Get started by creating your first project to organize your services and deployments.
				</p>
			</div>
		</div>
	{/if}
</div>