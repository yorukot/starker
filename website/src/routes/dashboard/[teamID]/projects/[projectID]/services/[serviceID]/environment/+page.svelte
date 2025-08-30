<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import LucideLoader2 from '~icons/lucide/loader-2';
	import LucideSave from '~icons/lucide/save';
	import LucideEye from '~icons/lucide/eye';
	import LucideEyeOff from '~icons/lucide/eye-off';
	import SettingsIcon from '~icons/lucide/settings';
	import { page } from '$app/state';
	import { invalidate } from '$app/navigation';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import {
		updateServiceEnvironmentSchema,
		type UpdateServiceEnvironmentForm
	} from '$lib/schemas/request/service';
	import { authPatch } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { toast } from 'svelte-sonner';
	import { CodeEditor } from '$lib/components/ui/code-editor';
	import type { PageData } from './$types';
	import type { ServiceEnvironment } from '$lib/schemas/service';

	let { data }: { data: PageData } = $props();

	const environments = $derived(data.environments || []);
	const error = $derived(data.error || null);

	let envContent = $state('');
	let showValues = $state(false);

	// Convert ServiceEnvironment[] to .env format
	function environmentsToEnvFormat(envs: ServiceEnvironment[]): string {
		return envs.map((env) => `${env.key}=${env.value}`).join('\n');
	}

	// Convert .env format to ServiceEnvironment[]
	function envFormatToEnvironments(content: string): { key: string; value: string; id?: number }[] {
		if (!content.trim()) return [];

		return content
			.split('\n')
			.filter((line) => line.trim() && !line.startsWith('#'))
			.map((line) => {
				const [key, ...valueParts] = line.split('=');
				const value = valueParts.join('=');

				// Find existing environment to preserve ID
				const existingEnv = environments.find((env) => env.key === key?.trim());

				return {
					key: key?.trim() || '',
					value: value || '',
					...(existingEnv && { id: existingEnv.id })
				};
			})
			.filter((env) => env.key); // Only include entries with valid keys
	}

	// Initialize environment content when data loads or changes
	$effect(() => {
		const formatted = environmentsToEnvFormat(environments);
		envContent = formatted;
	});

	// Get display content (masked or not based on showValues)
	const displayContent = $derived(() => {
		if (showValues || !envContent) {
			return envContent;
		}

		// Return masked version
		return envContent
			.split('\n')
			.map((line) => {
				if (line.trim() && !line.startsWith('#') && line.includes('=')) {
					const [key] = line.split('=', 1);
					return `${key}=${'•'.repeat(8)}`;
				}
				return line;
			})
			.join('\n');
	});

	const { form, errors, isSubmitting } = createForm<UpdateServiceEnvironmentForm>({
		extend: validator({ schema: updateServiceEnvironmentSchema }),
		initialValues: {
			environments: []
		},
		onSubmit: async () => {
			try {
				const environmentsData = envFormatToEnvironments(envContent);

				const response = await authPatch(
					`${PUBLIC_API_BASE_URL}/teams/${page.params.teamID}/projects/${page.params.projectID}/services/${page.params.serviceID}/env`,
					{
						environments: environmentsData
					}
				);

				if (!response.ok) {
					const errorData = await response.json();
					toast.error(errorData.message || 'Failed to update environment variables');
					return;
				}

				toast.success('Environment variables updated successfully!');

				// Refresh the page data to show updated environment variables
				await invalidate(() => true);
			} catch (err) {
				console.error('Error updating environment variables:', err);
				toast.error('Failed to update environment variables. Please try again.');
			}
		}
	});

	function toggleShowValues() {
		showValues = !showValues;
	}
</script>

<div class="flex h-full flex-col gap-6 p-6">
	{#if error}
		<Alert.Root variant="destructive">
			<Alert.Description>
				{error}
			</Alert.Description>
		</Alert.Root>
	{:else}
		<!-- Header -->
		<div class="flex items-center gap-3">
			<div class="rounded-lg border border-primary/20 bg-primary/10 p-3">
				<SettingsIcon class="h-6 w-6 text-primary" />
			</div>
			<div>
				<h1 class="text-2xl font-semibold text-foreground">Global Environment</h1>
				<p class="text-sm text-muted-foreground">
					Configure environment variables for your service using .env format
				</p>
			</div>
		</div>
		<!-- Form -->
		<form use:form class="space-y-6">
			<Card.Root>
				<Card.Content class="space-y-4">
					<div class="flex items-center justify-between">
						<Label>Environment Variables</Label>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onclick={toggleShowValues}
							class="gap-2"
						>
							{#if showValues}
								<LucideEyeOff class="h-4 w-4" />
								Hide Values
							{:else}
								<LucideEye class="h-4 w-4" />
								Show Values
							{/if}
						</Button>
					</div>
					<div class="space-y-2">
						<div class={$errors.environments ? 'border-destructive' : ''}>
							{#if showValues}
								<CodeEditor
									bind:value={envContent}
									language="toml"
									placeholder="DATABASE_URL=postgresql://user:password@localhost:5432/mydb&#10;API_KEY=your_api_key_here&#10;NODE_ENV=production"
									class="min-h-[400px]"
								/>
							{:else}
								<CodeEditor
									value={displayContent()}
									readonly={true}
									language="toml"
									placeholder="DATABASE_URL=postgresql://user:password@localhost:5432/mydb&#10;API_KEY=your_api_key_here&#10;NODE_ENV=production"
									class="min-h-[400px]"
								/>
							{/if}
						</div>
						{#if $errors.environments}
							<p class="text-sm text-destructive">{$errors.environments[0]}</p>
						{/if}
						<div class="space-y-1 text-xs text-muted-foreground">
							<p>• Use KEY=VALUE format, one per line</p>
							<p>• Keys must be uppercase letters, numbers, and underscores only</p>
							<p>• Comments starting with # are ignored</p>
							{#if !showValues}
								<p>• Click "Show Values" to edit variables</p>
							{:else}
								<p>• Values are visible and editable</p>
							{/if}
						</div>
					</div>
				</Card.Content>
			</Card.Root>

			<!-- Actions -->
			<div class="flex items-center justify-end gap-4">
				<Button type="submit" disabled={$isSubmitting} class="gap-2">
					{#if $isSubmitting}
						<LucideLoader2 class="h-4 w-4 animate-spin" />
						Updating...
					{:else}
						<LucideSave class="h-4 w-4" />
						Update Environment
					{/if}
				</Button>
			</div>
		</form>
	{/if}
</div>
