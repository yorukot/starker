<script lang="ts">
	import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import LucideFolderPlus from '~icons/lucide/folder-plus';
	import { createProjectSchema, type CreateProjectForm } from '$lib/schemas/request/project';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { authPost } from '$lib/api/client';
	import { page } from '$app/state';
	import { invalidate } from '$app/navigation';

	const teamID = page.params.teamID;

	let serverError = $state('');
	let dialogOpen = $state(false);

	const { form, errors, isSubmitting, setFields } = createForm<CreateProjectForm>({
		extend: validator({ schema: createProjectSchema }),
		onSubmit: async (values) => {
			serverError = '';

			const response = await authPost(`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects`, {
				name: values.name,
				description: values.description || undefined
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw errorData;
			}

			return response.json();
		},
		onSuccess: async () => {
			dialogOpen = false;
			invalidate(`projects:${page.params.teamID}`);
			setFields({ name: '', description: '' });
		},
		onError: (error: unknown) => {
			console.error('Project creation error:', error);
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Failed to create project. Please try again.';
			}
		}
	});
</script>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Trigger class={buttonVariants({ variant: 'default' })}>
		<LucideFolderPlus class="h-4 w-4" />
		Create new project
	</Dialog.Trigger>
	<Dialog.Content class="">
		<Dialog.Header>
			<Dialog.Title>Create new project</Dialog.Title>
			<Dialog.Description>
				Create a new project to organize your services and deployments.
			</Dialog.Description>
		</Dialog.Header>
		<form use:form>
			{#if serverError}
				<div
					class="text-destructive-foreground mb-4 rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm"
				>
					{serverError}
				</div>
			{/if}
			<div class="grid gap-4 py-4">
				<div class="flex flex-col gap-2">
					<Label for="name" class="text-right">Name</Label>
					<Input
						id="name"
						name="name"
						placeholder="Enter project name"
						class="col-span-3 {$errors.name ? 'border-destructive' : ''}"
					/>
					{#if $errors.name}
						<span class="col-span-3 text-sm text-destructive">{$errors.name[0]}</span>
					{/if}
				</div>
				<div class="flex flex-col gap-2">
					<Label for="description" class="text-right">Description</Label>
					<Textarea
						id="description"
						name="description"
						placeholder="Enter project description (optional)"
						class="col-span-3 {$errors.description ? 'border-destructive' : ''}"
					/>
					{#if $errors.description}
						<span class="col-span-3 text-sm text-destructive">{$errors.description[0]}</span>
					{/if}
				</div>
			</div>
			<Dialog.Footer>
				<Button
					type="submit"
					disabled={$isSubmitting ||
						Object.values($errors).some((error) => error && error.length > 0)}
				>
					{$isSubmitting ? 'Creating...' : 'Create'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
