<script lang="ts">
	import NewKey from '$lib/components/key/new-key.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import type { PageData } from './$types';
	import LucideKeyRound from '~icons/lucide/key-round';
	import LucideTimer from '~icons/lucide/timer';
	import { page } from '$app/state';
	import { invalidate } from '$app/navigation';

	let { data }: { data: PageData } = $props();

	const privateKeys = $derived(data.privateKeys || []);

	async function onKeyCreated() {
		await invalidate(`keys:${page.params.teamID}`);
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<div class="flex flex-col gap-2">
			<h1 class="text-2xl font-semibold text-foreground">SSH Keys</h1>
			<p class="text-sm text-muted-foreground">
				Manage your SSH private keys for server authentication
			</p>
		</div>
		<NewKey {onKeyCreated} />
	</div>

	<Separator />

	{#if privateKeys.length > 0}
		<div class="grid grid-cols-2 gap-4 lg:grid-cols-3 2xl:grid-cols-4">
			{#each privateKeys as key (key.id)}
				<a href="/dashboard/{page.params.teamID}/keys/{key.id}" class="block">
					<Card.Root
						class="cursor-pointer border border-border/50 bg-card/50 transition-colors hover:border-border hover:bg-card"
					>
						<Card.Header>
							<div class="flex flex-col space-y-3">
								<div class="flex items-center gap-2">
									<div class="rounded-lg border border-primary/20 bg-primary/10 p-2">
										<LucideKeyRound class="h-5 w-5 text-primary" />
									</div>
									<h3>{key.name}</h3>
								</div>
							</div>
						</Card.Header>
						<Card.Content>
							<div class="space-y-1">
								<div class="space-y-1 text-xs text-muted-foreground">
									<div class="flex gap-1">
										<LucideTimer /> Created: {new Date(key.created_at).toLocaleDateString()}
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
				<LucideKeyRound class="h-16 w-16 text-muted-foreground/50" />
			</div>
			<div class="space-y-2 text-center">
				<h3 class="text-lg font-medium text-foreground">No SSH keys yet</h3>
				<p class="max-w-sm text-sm text-muted-foreground">
					Get started by creating your first SSH key to authenticate with your servers.
				</p>
			</div>
		</div>
	{/if}
</div>
