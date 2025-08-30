<script lang="ts">
	import * as Sheet from '$lib/components/ui/sheet';
	import TerminalIcon from '~icons/lucide/terminal';
	import LogsViewer from './logs-viewer.svelte';

	interface LogMessage {
		id: string;
		timestamp: string;
		type: 'log' | 'error' | 'info' | 'status' | 'step';
		message: string;
	}

	interface Props {
		open: boolean;
		messages: LogMessage[];
		onOpenChange?: (open: boolean) => void;
	}

	let { open = $bindable(), messages, onOpenChange }: Props = $props();

	function handleOpenChange(newOpen: boolean) {
		open = newOpen;
		onOpenChange?.(newOpen);
	}
</script>

<Sheet.Root {open} onOpenChange={handleOpenChange}>
	<Sheet.Content side="right" class="w-full sm:w-[600px] sm:max-w-[600px]">
		<Sheet.Header>
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-2">
					<TerminalIcon class="h-5 w-5" />
					<Sheet.Title>Service Operation Logs</Sheet.Title>
				</div>
				<div class="flex items-center gap-2"></div>
			</div>
			<Sheet.Description>Real-time output from service operations</Sheet.Description>
		</Sheet.Header>

		<div class="mt-4 flex-1 overflow-hidden">
			<LogsViewer
				{messages}
				description="Service operation logs will appear here"
				class="h-[calc(100vh-200px)]"
			/>
		</div>
	</Sheet.Content>
</Sheet.Root>
