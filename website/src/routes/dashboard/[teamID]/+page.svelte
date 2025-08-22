<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	onMount(() => {
		const teamID = $page.params.teamID;
		goto(`/dashboard/${teamID}/projects`);
	});
</script>
