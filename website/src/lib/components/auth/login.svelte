<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import LucideLogIn from '~icons/lucide/log-in';

	const id = $props.id();
</script>

<Card.Root class="mx-auto w-full max-w-sm">
	<Card.Header>
		<Card.Title class="text-2xl">Login</Card.Title>
		<Card.Description>Enter your email below to login to your account</Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="grid gap-4">
			<div class="grid gap-2">
				<Label for="email-{id}">Email</Label>
				<Input id="email-{id}" type="email" placeholder="m@example.com" required />
			</div>
			<div class="grid gap-2">
				<div class="flex items-center">
					<Label for="password-{id}">Password</Label>
					<a href="##" class="ml-auto inline-block text-sm underline"> Forgot your password? </a>
				</div>
				<Input id="password-{id}" type="password" required />
			</div>
			<Button type="submit" class="w-full">
				<LucideLogIn />
				Login
			</Button>
		</div>
		<div class="mt-4 text-center text-sm">
			Don't have an account?
			<a href="/auth/register" class="underline"> Sign up </a>
		</div>
	</Card.Content>
</Card.Root>
