<script>
	import NewTeamForm from '$lib/components/team/new-team-form.svelte';
	import * as Card from '$lib/components/ui/card/index.js';
</script>

<div class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="mx-auto w-full max-w-md">
		<Card.Header>
			<Card.Title class="text-2xl">Create New Team</Card.Title>
			<Card.Description>Enter a name for your new team</Card.Description>
		</Card.Header>
		<Card.Content>
			<NewTeamForm />
		</Card.Content>
	</Card.Root>
</div>
