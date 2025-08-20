<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import LucideLogIn from '~icons/lucide/log-in';
	import { registerSchema, type RegisterForm } from '$lib/schemas/request/user';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { goto } from '$app/navigation';

	let serverError = '';

	const { form, errors, isSubmitting } = createForm<RegisterForm>({
		extend: validator({ schema: registerSchema }),
		onSubmit: async (values) => {
			// Clear any previous server errors
			serverError = '';

			// Make registration request
			const response = await fetch(`${PUBLIC_API_BASE_URL}/auth/register`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				credentials: 'include', // For refresh token cookie
				body: JSON.stringify({
					display_name: values.displayName,
					email: values.email,
					password: values.password
				})
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw errorData; // This will go to onError
			}

			return response.json(); // This will go to onSuccess
		},
		onSuccess: () => {
			goto('/dashboard');
		},
		onError: (error: unknown) => {
			console.error('Registration error:', error);
			// Handle server-side errors
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Registration failed. Please try again.';
			}
		}
	});
</script>

<form use:form>
	<div class="grid gap-4">
		{#if serverError}
			<div
				class="text-destructive-foreground rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm"
			>
				{serverError}
			</div>
		{/if}
		<div class="grid gap-2">
			<Label for="displayName">Display name</Label>
			<Input
				id="displayName"
				name="displayName"
				type="text"
				placeholder="John Wick"
				class={$errors.displayName ? 'border-destructive' : ''}
			/>
			{#if $errors.displayName}
				<span class="text-sm text-destructive">{$errors.displayName[0]}</span>
			{/if}
		</div>

		<div class="grid gap-2">
			<Label for="email">Email</Label>
			<Input
				id="email"
				name="email"
				type="email"
				placeholder="m@example.com"
				class={$errors.email ? 'border-destructive' : ''}
			/>
			{#if $errors.email}
				<span class="text-sm text-destructive">{$errors.email[0]}</span>
			{/if}
		</div>

		<div class="grid gap-2">
			<Label for="password">Password</Label>
			<Input
				id="password"
				name="password"
				type="password"
				class={$errors.password ? 'border-destructive' : ''}
			/>
			{#if $errors.password}
				<span class="text-sm text-destructive">{$errors.password[0]}</span>
			{/if}
		</div>

		<div class="grid gap-2">
			<Label for="confirmPassword">Confirm Password</Label>
			<Input
				id="confirmPassword"
				name="confirmPassword"
				type="password"
				class={$errors.confirmPassword ? 'border-destructive' : ''}
			/>
			{#if $errors.confirmPassword}
				<span class="text-sm text-destructive">{$errors.confirmPassword[0]}</span>
			{/if}
		</div>

		<Button type="submit" class="w-full" disabled={$isSubmitting}>
			<LucideLogIn />
			{$isSubmitting ? 'Signing up...' : 'Sign up'}
		</Button>
	</div>
</form>
