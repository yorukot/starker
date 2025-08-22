<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import LucideArrowLeft from '~icons/lucide/arrow-left';
	import LucideContainer from '~icons/lucide/container';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const project = $derived(data.project);
	const error = $derived(data.error || null);

	function goBack() {
		goto(`/dashboard/${page.params.teamID}/projects/${page.params.projectID}`);
	}

	function selectCompose() {
		// For now, just navigate to a compose setup page (to be created later)
		goto(`/dashboard/${page.params.teamID}/projects/${page.params.projectID}/services/new/compose`);
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<Button variant="ghost" size="sm" onclick={goBack} class="gap-2">
			<LucideArrowLeft class="h-4 w-4" />
			Back to Project
		</Button>
	</div>

	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else}
		<!-- Header -->
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-foreground">Add New Service</h1>
			<p class="text-sm text-muted-foreground">
				Choose a service type or template to deploy in {project?.name || 'your project'}
			</p>
		</div>

		<!-- Service Type Selection -->
		<div class="space-y-4">
			<h2 class="text-lg font-medium text-foreground">Docker based</h2>
			<p class="text-sm text-muted-foreground">
				Select from available docker based service to get started
			</p>

			<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
				<!-- Docker Compose Template -->
				<Card.Root
					class="cursor-pointer border border-border/50 bg-card/50 transition-colors hover:border-border hover:bg-card"
					onclick={selectCompose}
				>
					<Card.Header>
						<div class="flex items-center gap-2">
							<div class="rounded-lg border border-primary/20 bg-primary/10 p-2">
								<LucideContainer class="h-5 w-5 text-primary" />
							</div>
							<div>
								<h3 class="font-medium">Docker Compose</h3>
								<p class="text-xs text-muted-foreground">Multi-container application</p>
							</div>
						</div>
					</Card.Header>
					<Card.Content>
						<div class="space-y-2">
							<p class="text-sm text-muted-foreground">
								Deploy applications using Docker Compose with multiple containers, networks, and
								volumes.
							</p>
						</div>
					</Card.Content>
				</Card.Root>
			</div>
		</div>
	{/if}
</div>
