<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import EditProjectDialog from './edit-project-dialog.svelte';
	import DeleteProjectDialog from './delete-project-dialog.svelte';
	import LucideMoreHorizontal from '~icons/lucide/more-horizontal';
	import LucidePencil from '~icons/lucide/pencil';
	import LucideTrash2 from '~icons/lucide/trash-2';
	import type { Project } from '$lib/schemas/project';

	let {
		project
	}: {
		project: Project;
	} = $props();

	let editDialogOpen = $state(false);
	let deleteDialogOpen = $state(false);
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		<Button variant="ghost" size="sm" class="h-8 w-8 p-0 data-[state=open]:bg-muted">
			<LucideMoreHorizontal class="h-4 w-4" />
			<span class="sr-only">Open menu</span>
		</Button>
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-[160px]">
		<DropdownMenu.Item
			class="flex cursor-pointer items-center gap-2"
			onclick={() => (editDialogOpen = true)}
		>
			<LucidePencil class="h-4 w-4" />
			Edit project
		</DropdownMenu.Item>
		<DropdownMenu.Separator />
		<DropdownMenu.Item
			variant="destructive"
			class="flex cursor-pointer items-center gap-2"
			onclick={() => (deleteDialogOpen = true)}
		>
			<LucideTrash2 class="h-4 w-4" />
			Delete
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<EditProjectDialog {project} bind:dialogOpen={editDialogOpen} />

<DeleteProjectDialog {project} bind:open={deleteDialogOpen} />
