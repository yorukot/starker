<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import LucideTrash2 from '~icons/lucide/trash-2';
	import type { Project } from '$lib/schemas/project';
	import { authDelete } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { page } from '$app/state';
	import { invalidate } from '$app/navigation';

	interface Props {
		project: Project | null;
		open?: boolean;
	}

	let { project, open = $bindable(false) }: Props = $props();
	let error = $state('');
	let deleting = $state(false);

	async function deleteProject() {
		if (!project) return;

		deleting = true;
		error = '';

		try {
			const response = await authDelete(
				`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/projects/${project.id}`
			);

			if (!response.ok) {
				const errorText = await response.text();
				error = `Failed to delete project: ${response.status} ${errorText}`;
				return;
			}

			// Success - close dialog and notify parent
			open = false;
			invalidate(`projects:${page.params.teamID}`);
		} catch (err: unknown) {
			console.error('Error deleting project:', err);
			error = err instanceof Error ? err.message : 'Unknown error occurred';
		} finally {
			deleting = false;
		}
	}

	function handleClose() {
		open = false;
		error = '';
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<LucideTrash2 class="h-5 w-5 text-destructive" />
				Delete Project
			</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to delete the project "{project?.name}"? This action cannot be undone
				and will also delete all associated services and configurations.
			</Dialog.Description>
		</Dialog.Header>

		{#if error}
			<Alert.Root variant="destructive">
				<Alert.Description>
					{error}
				</Alert.Description>
			</Alert.Root>
		{/if}

		<Dialog.Footer class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<Button variant="outline" onclick={handleClose} disabled={deleting}>Cancel</Button>
			<Button variant="destructive" onclick={deleteProject} disabled={deleting} class="gap-2">
				{#if deleting}
					<div
						class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
					></div>
					Deleting...
				{:else}
					<LucideTrash2 class="h-4 w-4" />
					Delete Project
				{/if}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
