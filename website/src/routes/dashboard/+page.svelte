<script lang="ts">
    import NewTeamDialog from '$lib/components/team/new-team-dialog.svelte';
    import { goto } from '$app/navigation';
    import type { PageData } from './$types';

    let { data }: { data: PageData } = $props();

    let showNewTeamDialog = $state(true);

    function handleNewTeamSuccess(team: import('$lib/schemas/team').Team) {
        showNewTeamDialog = false;
        // Navigate to the new team's projects page
        goto(`/dashboard/${team.id}/projects`);
    }

    // Show dialog if no teams exist
    $effect(() => {
        if (data.teams && data.teams.length === 0) {
            showNewTeamDialog = true;
        }
    });
</script>

{#if data.teams && data.teams.length === 0}
    <NewTeamDialog bind:open={showNewTeamDialog} onSuccess={handleNewTeamSuccess} />
{:else}
    <!-- This should not be reached due to redirects in +page.ts, but just in case -->
    <div class="flex min-h-screen items-center justify-center">
        <div class="text-center">
            <h1 class="text-2xl font-bold">Loading...</h1>
        </div>
    </div>
{/if}
