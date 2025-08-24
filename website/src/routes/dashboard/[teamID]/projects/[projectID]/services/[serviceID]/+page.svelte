<script>
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';

	onMount(() => {
		const { teamID, projectID, serviceID } = page.params;
		goto(`/dashboard/${teamID}/projects/${projectID}/services/${serviceID}/overview`);
	});
</script>
