<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { Toaster } from "$lib/components/ui/sonner/index.js";

	let { children } = $props();
</script>

<Toaster />

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children?.()}
