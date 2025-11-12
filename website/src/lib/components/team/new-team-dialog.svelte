<script lang="ts">
    import * as Dialog from '$lib/components/ui/dialog/index.js';
    import NewTeamForm from './new-team-form.svelte';
    import type { Team } from '$lib/schemas/team';

    let {
        open = $bindable(false),
        onSuccess
    }: {
        open: boolean;
        onSuccess?: (team: Team) => void;
    } = $props();
</script>

<Dialog.Root bind:open>
    <Dialog.Content class="sm:max-w-[425px]">
        <Dialog.Header>
            <Dialog.Title>Create New Team</Dialog.Title>
            <Dialog.Description>Enter a name for your new team</Dialog.Description>
        </Dialog.Header>
        <NewTeamForm {onSuccess} />
    </Dialog.Content>
</Dialog.Root>
