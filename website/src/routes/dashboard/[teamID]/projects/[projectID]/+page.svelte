<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import type { PageData } from './$types';
	import LucideFolder from '~icons/lucide/folder';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import LucideSettings from '~icons/lucide/settings';
	import LucidePlus from '~icons/lucide/plus';
	import LucideServer from '~icons/lucide/server';
	import LucideTimer from '~icons/lucide/timer';
	import LucideActivity from '~icons/lucide/activity';
	import LucideContainer from '~icons/lucide/container';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';

	let { data }: { data: PageData } = $props();

	const project = $derived(data.project);
	const services = $derived(data.services || []);
	const error = $derived(data.error || null);

	function goBack() {
		goto(`/dashboard/${page.params.teamID}/projects`);
	}

	function getStateColor(state: string) {
		switch (state) {
			case 'running':
				return 'text-green-500';
			case 'stopped':
				return 'text-red-500';
			case 'starting':
				return 'text-yellow-500';
			case 'stopping':
				return 'text-orange-500';
			case 'restarting':
				return 'text-blue-500';
			default:
				return 'text-gray-500';
		}
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<Button variant="ghost" size="sm" onclick={goBack} class="gap-2">
			<LucideArrowLeft class="h-4 w-4" />
			Back to Projects
		</Button>
	</div>

	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else if project}
		<!-- Header with project info -->
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-4">
				<div class="rounded-lg border border-primary/20 bg-primary/10 p-3">
					<LucideFolder class="h-6 w-6 text-primary" />
				</div>
				<div>
					<h1 class="text-2xl font-semibold text-foreground">{project.name}</h1>
					{#if project.description}
						<p class="text-muted-foreground">{project.description}</p>
					{/if}
				</div>
			</div>
			<Button
				href="/dashboard/{page.params.teamID}/projects/{project.id}/services/new"
				class="gap-2"
			>
				<LucidePlus class="h-4 w-4" />
				Add Service
			</Button>
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

			{#if services.length > 0}
				<div class="grid grid-cols-1 gap-4 lg:grid-cols-2 2xl:grid-cols-3">
					{#each services as service (service.id)}
						<a
							href="/dashboard/{page.params.teamID}/projects/{project.id}/services/{service.id}"
							class="block"
						>
							<Card.Root
								class="cursor-pointer border border-border/50 bg-card/50 transition-colors hover:border-border hover:bg-card"
							>
								<Card.Header>
									<div class="flex flex-col space-y-3">
										<div class="flex items-center gap-2">
											<div class="rounded-lg border border-primary/20 bg-primary/10 p-2">
												<LucideContainer class="h-5 w-5 text-primary" />
											</div>
											<div class="flex flex-col">
												<h3 class="font-medium">{service.name}</h3>
												{#if service.description}
													<p class="text-xs text-muted-foreground">{service.description}</p>
												{/if}
											</div>
										</div>
									</div>
								</Card.Header>
								<Card.Content>
									<div class="space-y-2">
										<div class="space-y-1 text-sm text-muted-foreground">
											<div class="flex items-center gap-2">
												<LucideActivity class="h-3 w-3" />
												<span class="{getStateColor(service.state)} font-medium capitalize"
													>{service.state}</span
												>
											</div>
											<div class="flex items-center gap-2">
												<LucideServer class="h-3 w-3" />
												<span>Type: {service.type}</span>
											</div>
											{#if service.last_deployed_at}
												<div class="flex items-center gap-2">
													<LucideTimer class="h-3 w-3" />
													<span
														>Deployed: {new Date(
															service.last_deployed_at
														).toLocaleDateString()}</span
													>
												</div>
											{:else}
												<div class="flex items-center gap-2">
													<LucideTimer class="h-3 w-3" />
													<span>Created: {new Date(service.created_at).toLocaleDateString()}</span>
												</div>
											{/if}
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
						<LucideSettings class="h-16 w-16 text-muted-foreground/50" />
					</div>
					<div class="space-y-2 text-center">
						<h3 class="text-lg font-medium text-foreground">No services yet</h3>
						<p class="max-w-sm text-sm text-muted-foreground">
							Get started by adding your first service to deploy and manage in this project.
						</p>
					</div>
					<Button
						href="/dashboard/{page.params.teamID}/projects/{project.id}/services/new"
						class="gap-2"
					>
						<LucidePlus class="h-4 w-4" />
						Add Your First Service
					</Button>
				</div>
			{/if}
		</div>

		<!-- Project metadata -->
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
