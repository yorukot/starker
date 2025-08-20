<script lang="ts">
	import { onMount } from 'svelte';
	import { refreshToken, isTokenValid } from '$lib/api/auth';
	import { browser } from '$app/environment';

	let { children } = $props();

	onMount(() => {
		if (!browser) return;

		// Check and refresh token on mount
		(async () => {
			if (!isTokenValid()) {
				await refreshToken();
			}
		})();

		// Set up periodic token refresh (check every 5 minutes)
		const refreshInterval = setInterval(
			async () => {
				if (!isTokenValid()) {
					await refreshToken();
				}
			},
			5 * 60 * 1000
		); // 5 minutes

		// Cleanup interval on unmount
		return () => {
			clearInterval(refreshInterval);
		};
	});
</script>

{@render children?.()}
