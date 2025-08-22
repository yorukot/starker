<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import type { PageData } from './$types';
	import LucideServer from '~icons/lucide/server';
	import LucidePlus from '~icons/lucide/plus';
	import LucideTimer from '~icons/lucide/timer';
	import LucideUser from '~icons/lucide/user';
	import LucideGlobe from '~icons/lucide/globe';
	import { page } from '$app/state';

	let { data }: { data: PageData } = $props();

	const servers = $derived((data).servers || []);
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<div class="flex flex-col gap-2">
			<h1 class="text-2xl font-semibold text-foreground">Servers</h1>
			<p class="text-sm text-muted-foreground">Manage your SSH-enabled servers and connections</p>
		</div>
		<Button href="/dashboard/{page.params.teamID}/servers/new" class="gap-2">
			<LucidePlus class="h-4 w-4" />
			Add Server
		</Button>
	</div>

	<Separator />

	{#if servers.length > 0}
		<div class="grid grid-cols-1 gap-4 lg:grid-cols-2 2xl:grid-cols-3">
			{#each servers as server (server.id)}
				<a href="/dashboard/{page.params.teamID}/servers/{server.id}" class="block">
					<Card.Root
						class="cursor-pointer border border-border/50 bg-card/50 transition-colors hover:border-border hover:bg-card"
					>
						<Card.Header>
							<div class="flex flex-col space-y-3">
								<div class="flex items-center gap-2">
									<div class="rounded-lg border border-primary/20 bg-primary/10 p-2">
										<LucideServer class="h-5 w-5 text-primary" />
									</div>
									<div class="flex flex-col">
										<h3 class="font-medium">{server.name}</h3>
										{#if server.description}
											<p class="text-xs text-muted-foreground">{server.description}</p>
										{/if}
									</div>
								</div>
							</div>
						</Card.Header>
						<Card.Content>
							<div class="space-y-2">
								<div class="space-y-1 text-sm text-muted-foreground">
									<div class="flex items-center gap-2">
										<LucideGlobe class="h-3 w-3" />
										<span>{server.ip}:{server.port}</span>
									</div>
									<div class="flex items-center gap-2">
										<LucideUser class="h-3 w-3" />
										<span>{server.user}</span>
									</div>
									<div class="flex items-center gap-2">
										<LucideTimer class="h-3 w-3" />
										<span>Created: {new Date(server.created_at).toLocaleDateString()}</span>
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
				<LucideServer class="h-16 w-16 text-muted-foreground/50" />
			</div>
			<div class="space-y-2 text-center">
				<h3 class="text-lg font-medium text-foreground">No servers yet</h3>
				<p class="max-w-sm text-sm text-muted-foreground">
					Get started by adding your first server to manage with SSH connections.
				</p>
			</div>
			<Button href="/dashboard/{page.params.teamID}/servers/new" class="gap-2">
				<LucidePlus class="h-4 w-4" />
				Add Your First Server
			</Button>
		</div>
	{/if}
</div>
