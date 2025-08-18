<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { superForm } from 'sveltekit-superforms';
	import LucideLogIn from '~icons/lucide/log-in';
	import { registerSchema } from '$lib/schemas/auth.js';
	import { valibotClient } from 'sveltekit-superforms/adapters';
	import {
	  PUBLIC_API_BASE_URL
	} from '$env/static/public';
	
	const { id = 'register' } = $props();
	
	const { form, errors, enhance, submitting } = superForm(
		{
			displayName: '',
			email: '',
			password: '',
			confirmPassword: ''
		},
		{
			validators: valibotClient(registerSchema),
			onSubmit: async ({ formData, cancel }) => {
				cancel();
				
				try {
					const response = await fetch(`${PUBLIC_API_BASE_URL}/auth/register`, {
						method: 'POST',
						headers: {
							'Content-Type': 'application/json'
						},
						body: JSON.stringify({
							display_name: formData.get('displayName'),
							email: formData.get('email'),
							password: formData.get('password')
						})
					});

					if (response.ok) {
						console.log('Registration successful');
						// Handle successful registration (e.g., redirect to login)
					} else {
						const error = await response.json();
						console.error('Registration failed:', error);
						// Handle registration error
					}
				} catch (error) {
					console.error('Network error:', error);
					// Handle network error
				}
			}
		}
	);
</script>

<Card.Root class="mx-auto w-full max-w-sm">
	<Card.Header>
		<Card.Title class="text-2xl">Register</Card.Title>
		<Card.Description>Enter your email below to register your account</Card.Description>
	</Card.Header>
	<Card.Content>
		<form method="POST" use:enhance>
			<div class="grid gap-4">
				<div class="grid gap-2">
					<div class="flex items-center">
						<Label for="display-name-{id}">Display name</Label>
					</div>
					<Input
						id="display-name-{id}"
						name="displayName"
						type="text"
						placeholder="John Wick"
						bind:value={$form.displayName}
						class={$errors.displayName ? 'border-destructive' : ''}
					/>
					{#if $errors.displayName}
						<span class="text-sm text-destructive">{$errors.displayName}</span>
					{/if}
				</div>

				<div class="grid gap-2">
					<Label for="email-{id}">Email</Label>
					<Input
						id="email-{id}"
						name="email"
						type="email"
						placeholder="m@example.com"
						bind:value={$form.email}
						class={$errors.email ? 'border-destructive' : ''}
					/>
					{#if $errors.email}
						<span class="text-sm text-destructive">{$errors.email}</span>
					{/if}
				</div>

				<div class="grid gap-2">
					<div class="flex items-center">
						<Label for="password-{id}">Password</Label>
					</div>
					<Input
						id="password-{id}"
						name="password"
						type="password"
						bind:value={$form.password}
						class={$errors.password ? 'border-destructive' : ''}
					/>
					{#if $errors.password}
						<span class="text-sm text-destructive">{$errors.password}</span>
					{/if}
				</div>

				<div class="grid gap-2">
					<div class="flex items-center">
						<Label for="confirm-password-{id}">Confirm Password</Label>
					</div>
					<Input
						id="confirm-password-{id}"
						name="confirmPassword"
						type="password"
						bind:value={$form.confirmPassword}
						class={$errors.confirmPassword ? 'border-destructive' : ''}
					/>
					{#if $errors.confirmPassword}
						<span class="text-sm text-destructive">{$errors.confirmPassword}</span>
					{/if}
				</div>

				<Button type="submit" class="w-full" disabled={$submitting}>
					<LucideLogIn />
					{$submitting ? 'Signing up...' : 'Sign up'}
				</Button>
			</div>
		</form>
		<div class="mt-4 text-center text-sm">
			Already have an account?
			<a href="login" class="underline"> Log in </a>
		</div>
	</Card.Content>
</Card.Root>
