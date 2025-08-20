<script>
	import RegisterForm from '$lib/components/auth/register-form.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
</script>

<Card.Root class="mx-auto w-full max-w-sm">
	<Card.Header>
		<Card.Title class="text-2xl">Register</Card.Title>
		<Card.Description>Enter your email below to register your account</Card.Description>
	</Card.Header>
	<Card.Content>
		<RegisterForm />
		<div class="mt-4 text-center text-sm">
			Already have an account?
			<a href="login" class="underline"> Log in </a>
		</div>
	</Card.Content>
</Card.Root>
