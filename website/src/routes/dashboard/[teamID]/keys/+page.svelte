<script lang="ts">
	import NewKey from '$lib/components/key/new-key.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import type { PageData } from './$types';
	import LucideKeyRound from '~icons/lucide/key-round';

	let { data }: { data: PageData } = $props();

	const privateKeys = $derived(data.privateKeys || []);
</script>

<div class="flex flex-col gap-4">
	<div class="flex items-center justify-between">
		<div class="flex-bet flex flex-col gap-2">
			<h1 class="text-2xl text-foreground">Keys</h1>
			<p class="text-sm text-muted-foreground">Setting your server's keys</p>
		</div>
		<NewKey />
	</div>

	<Separator />

	<div class="h-full w-full">
		{#each privateKeys as key (key.id)}
			<Card.Root>
				<Card.Header>
					<Card.Title>Card Title</Card.Title>
					<Card.Description>Card Description</Card.Description>
				</Card.Header>
				<Card.Content>
					<p>Card Content</p>
				</Card.Content>
				<Card.Footer>
					<p>Card Footer</p>
				</Card.Footer>
			</Card.Root>
		{:else}
			<div class="flex flex-col justify-center items-center w-full h-full gap-5">
				<LucideKeyRound class="w-40 h-40 text-muted-foreground/50" />
				<p>You dont have any key at this moment!</p>
			</div>
		{/each}
	</div>
</div>
