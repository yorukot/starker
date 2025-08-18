<script lang="ts">
	let { children } = $props();
</script>


<div class="flex justify-center items-center h-screen">
  {@render children?.()}
</div>