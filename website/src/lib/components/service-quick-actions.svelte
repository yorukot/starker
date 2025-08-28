<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { authPatch } from '$lib/api/client';
	import { PUBLIC_API_BASE_URL } from '$env/static/public';
	import PlayIcon from '~icons/lucide/play';
	import StopIcon from '~icons/lucide/square';
	import RestartIcon from '~icons/lucide/rotate-cw';
	import TerminalIcon from '~icons/lucide/terminal';
	import type { Service } from '$lib/schemas/service';
	import { ServiceState } from '$lib/schemas/service';

	interface Props {
		service: Service;
		teamID: string;
		projectID: string;
		serviceID: string;
		onOperationStart: (operationType: string, response: Response) => void;
		onShowLogs: () => void;
	}

	let {
		service = $bindable(),
		teamID,
		projectID,
		serviceID,
		onOperationStart,
		onShowLogs
	}: Props = $props();

	let isStarting = $state(false);
	let isStopping = $state(false);
	let isRestarting = $state(false);

	function resetLoadingStates() {
		isStarting = false;
		isStopping = false;
		isRestarting = false;
	}

	async function startService() {
		if (service.state !== ServiceState.STOPPED) return;

		isStarting = true;
		service.state = ServiceState.STARTING;

		try {
			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/state`,
				{ state: 'start' }
			);

			if (response.ok) {
				onOperationStart('start', response);
			} else {
				console.error('Failed to start service:', response.statusText);
				service.state = ServiceState.STOPPED;
				isStarting = false;
			}
		} catch (error) {
			console.error('Error starting service:', error);
			service.state = ServiceState.STOPPED;
			isStarting = false;
		}
	}

	async function stopService() {
		if (service.state !== ServiceState.RUNNING) return;

		isStopping = true;
		service.state = ServiceState.STOPPING;

		try {
			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/state`,
				{ state: 'stop' }
			);

			if (response.ok) {
				onOperationStart('stop', response);
			} else {
				console.error('Failed to stop service:', response.statusText);
				service.state = ServiceState.RUNNING;
				isStopping = false;
			}
		} catch (error) {
			console.error('Error stopping service:', error);
			service.state = ServiceState.RUNNING;
			isStopping = false;
		}
	}

	async function restartService() {
		if (service.state !== ServiceState.RUNNING) return;

		isRestarting = true;
		service.state = ServiceState.RESTARTING;

		try {
			const response = await authPatch(
				`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}/state`,
				{ state: 'restart' }
			);

			if (response.ok) {
				onOperationStart('restart', response);
			} else {
				console.error('Failed to restart service:', response.statusText);
				service.state = ServiceState.RUNNING;
				isRestarting = false;
			}
		} catch (error) {
			console.error('Error restarting service:', error);
			service.state = ServiceState.RUNNING;
			isRestarting = false;
		}
	}

	export function resetStates() {
		resetLoadingStates();
	}
</script>

<div class="flex items-center gap-2">
	{#if service.state === ServiceState.STOPPED}
		<Button onclick={startService} disabled={isStarting} size="sm" class="flex items-center gap-2">
			<PlayIcon class="h-4 w-4" />
			{isStarting ? 'Starting...' : 'Start'}
		</Button>
	{:else if service.state === ServiceState.RUNNING}
		<Button
			variant="outline"
			onclick={restartService}
			disabled={isRestarting}
			size="sm"
			class="flex items-center gap-2"
		>
			<RestartIcon class="h-4 w-4" />
			{isRestarting ? 'Restarting...' : 'Restart'}
		</Button>
		<Button
			variant="destructive"
			onclick={stopService}
			disabled={isStopping}
			size="sm"
			class="flex items-center gap-2"
		>
			<StopIcon class="h-4 w-4" />
			{isStopping ? 'Stopping...' : 'Stop'}
		</Button>
	{/if}
	<Button variant="secondary" onclick={onShowLogs} size="sm" class="flex items-center gap-2">
		<TerminalIcon class="h-4 w-4" />
		Logs
	</Button>
</div>
