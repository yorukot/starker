<script lang="ts">
	let { msg }: { msg: string } = $props();
</script>

<div class="flex h-screen items-center justify-center">
	<div class="text-center">
		<div
			class="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-4 border-card border-t-primary"
		></div>
		<p class="text-foreground">{msg}</p>
	</div>
</div>
