<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import LucideLogIn from '~icons/lucide/log-in';
	import { loginSchema, type LoginForm } from '$lib/schemas/request/user';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import { createForm } from 'felte';
	import { validator } from '@felte/validator-yup';
	import { goto } from '$app/navigation';

	let { id } = $props<{ id: () => string }>();
	let serverError = $state('');

	const { form, errors, isSubmitting } = createForm<LoginForm>({
		extend: validator({ schema: loginSchema }),
		onSubmit: async (values) => {
			// Clear any previous server errors
			serverError = '';

			// Make login request
			const response = await fetch(`${PUBLIC_API_BASE_URL}/auth/login`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				credentials: 'include', // For refresh token cookie
				body: JSON.stringify({
					email: values.email,
					password: values.password
				})
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw errorData; // This will go to onError
			}

			const data = await response.json();

			return data;
		},
		onSuccess: () => {
			goto('/dashboard');
		},
		onError: (error: unknown) => {
			console.error('Login error:', error);
			// Handle server-side errors
			if (error && typeof error === 'object' && 'message' in error) {
				serverError = (error as { message: string }).message;
			} else {
				serverError = 'Login failed. Please try again.';
			}
		}
	});
</script>

<Card.Root class="mx-auto w-full max-w-sm">
	<Card.Header>
		<Card.Title class="text-2xl">Login</Card.Title>
		<Card.Description>Enter your email below to login to your account</Card.Description>
	</Card.Header>
	<Card.Content>
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
					<Label for="email-{id()}">Email</Label>
					<Input
						id="email-{id()}"
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
					<div class="flex items-center">
						<Label for="password-{id()}">Password</Label>
						<a href="##" class="ml-auto inline-block text-sm underline"> Forgot your password? </a>
					</div>
					<Input
						id="password-{id()}"
						name="password"
						type="password"
						class={$errors.password ? 'border-destructive' : ''}
					/>
					{#if $errors.password}
						<span class="text-sm text-destructive">{$errors.password[0]}</span>
					{/if}
				</div>
				<Button type="submit" class="w-full" disabled={$isSubmitting}>
					<LucideLogIn />
					{$isSubmitting ? 'Signing in...' : 'Login'}
				</Button>
			</div>
			<div class="mt-4 text-center text-sm">
				Don't have an account?
				<a href="/auth/register" class="underline"> Sign up </a>
			</div>
		</form>
	</Card.Content>
</Card.Root>
