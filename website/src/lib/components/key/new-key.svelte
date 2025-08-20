<script lang="ts">
	import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import LucideKeyRound from '~icons/lucide/key-round';
	import {
		createPrivateKeySchema,
		type CreatePrivateKeyForm
	} from '$lib/schemas/request/privatekey';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { authPost } from '$lib/api/client';
	import { generateSSHKey, type SSHKeyType } from '$lib/utils/ssh-key';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import LucideNotebookPen from '~icons/lucide/notebook-pen';
	import { page } from '$app/state';

	const teamID = page.params.teamID;

	let serverError = '';
	let dialogOpen = false;
	let generatedPublicKey = '';

	function generateKey(keyType: SSHKeyType) {
		try {
			const keyPair = generateSSHKey(keyType, {
				comment: `starker-generated-${keyType}-key`,
				...(keyType === 'rsa' && { keySize: 2048 })
			});
			setFields((fields) => ({ ...fields, privateKey: keyPair.privateKey }));
			generatedPublicKey = keyPair.publicKey;
		} catch (error) {
			console.error('Key generation error:', error);
			serverError = error instanceof Error ? error.message : 'Failed to generate key';
		}
	}

	const { form, errors, isSubmitting, setFields } = createForm<CreatePrivateKeyForm>({
		extend: validator({ schema: createPrivateKeySchema }),
		onSubmit: async (values) => {
			serverError = '';

			const response = await authPost(`${PUBLIC_API_BASE_URL}/teams/${teamID}/private-keys`, {
				name: values.name,
				description: values.description || undefined,
				private_key: values.privateKey
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw errorData;
			}

			return response.json();
		},
		onSuccess: () => {
			dialogOpen = false;
			setFields({ name: '', description: '', privateKey: '' });
		},
		onError: (error: unknown) => {
			console.error('Private key creation error:', error);
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Failed to create private key. Please try again.';
			}
		}
	});
</script>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Trigger class={buttonVariants({ variant: 'default' })}>
		<LucideKeyRound class="mr-2 h-4 w-4" />
		Create new key
	</Dialog.Trigger>
	<Dialog.Content class="">
		<Dialog.Header>
			<Dialog.Title>Create new key</Dialog.Title>
			<Dialog.Description
				>Create new key for your to access your server.<br /><strong
					>Do not use passphrase protected keys.</strong
				></Dialog.Description
			>
		</Dialog.Header>
		<div class="flex flex-wrap justify-between gap-2">
			<Button
				class="w-full"
				variant="secondary"
				type="button"
				onclick={() => generateKey('ed25519')}
			>
				Generate new ED25519 SSH Key
			</Button>
		</div>
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
						class="col-span-3 {$errors.name ? 'border-destructive' : ''}"
					/>
					{#if $errors.name}
						<span class="col-span-3 text-sm text-destructive">{$errors.name[0]}</span>
					{/if}
				</div>
				<div class="flex flex-col gap-2">
					<Label for="description" class="text-right">Description</Label>
					<Input
						id="description"
						name="description"
						class="col-span-3 {$errors.description ? 'border-destructive' : ''}"
					/>
					{#if $errors.description}
						<span class="col-span-3 text-sm text-destructive">{$errors.description[0]}</span>
					{/if}
				</div>
				<div class="flex flex-col gap-2">
					<Label for="privateKey" class="pt-2 text-right">Private Key</Label>
					<Textarea
						id="privateKey"
						name="privateKey"
						class="max-h-30 w-full resize-none overflow-y-auto font-mono break-all
           {$errors.privateKey ? 'border-destructive' : ''}"
						placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
					/>
					{#if $errors.privateKey}
						<span class="col-span-3 text-sm text-destructive">{$errors.privateKey[0]}</span>
					{/if}
				</div>
				{#if generatedPublicKey}
					<Alert.Root>
						<LucideNotebookPen />
						<Alert.Title>Run this command on your server to authorize the key:</Alert.Title>
						<Alert.Description>
							<code class="block rounded bg-muted p-2 font-mono text-xs break-all">
								echo "{generatedPublicKey}" >> ~/.ssh/authorized_keys
							</code>
						</Alert.Description>
					</Alert.Root>
				{:else}
					<Alert.Root>
						<LucideNotebookPen />
						<Alert.Title>Add your public key to server</Alert.Title>
						<Alert.Description>
							You need to add your public key to the server's ~/.ssh/authorized_keys file
						</Alert.Description>
					</Alert.Root>
				{/if}
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
