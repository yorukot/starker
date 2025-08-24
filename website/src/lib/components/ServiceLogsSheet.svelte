<script lang="ts">
	import * as Sheet from '$lib/components/ui/sheet';
	import TerminalIcon from '~icons/lucide/terminal';

	interface LogMessage {
		id: string;
		timestamp: string;
		type: 'log' | 'error' | 'info' | 'status';
		message: string;
	}

	interface Props {
		open: boolean;
		messages: LogMessage[];
		onOpenChange?: (open: boolean) => void;
	}

	let { open = $bindable(), messages, onOpenChange }: Props = $props();
	let logContainer: HTMLDivElement;

	// Get CSS classes for different log message types using shadcn design tokens
	function getLogMessageClass(type: string) {
		switch (type) {
			case 'log':
				return 'text-foreground bg-foreground/10 border-l-foreground/50';
			case 'error':
				return 'text-foreground/80 bg-destructive/10 border-l-destructive';
			case 'info':
				return 'text-secondary-foreground bg-secondary/10 border-l-secondary';
			case 'status':
				return 'text-foreground bg-primary/10 border-l-primary';
			default:
				return 'text-muted-foreground bg-muted/50 border-l-border';
		}
	}

	// Auto-scroll to bottom when new messages arrive
	$effect(() => {
		if (messages.length > 0 && logContainer) {
			setTimeout(() => {
				logContainer.scrollTop = logContainer.scrollHeight;
			}, 10);
		}
	});

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
			<div
				bind:this={logContainer}
				class="h-[calc(100vh-200px)] space-y-1 overflow-y-auto rounded-lg border bg-background p-3"
			>
				{#each messages as log (log.id)}
					<div
						class="rounded-r border-l-4 py-2 pl-3 transition-colors {getLogMessageClass(log.type)}"
					>
						<div class="flex items-start gap-3 text-sm">
							<span class="mt-0.5 min-w-[70px] font-mono text-xs text-muted-foreground">
								{log.timestamp}
							</span>
							<span class="mt-0.5 min-w-[60px] text-xs font-medium tracking-wide uppercase">
								{log.type}
							</span>
							<span class="flex-1 font-mono leading-5 break-words">
								{log.message}
							</span>
						</div>
					</div>
				{/each}

				{#if messages.length === 0}
					<div class="flex h-full items-center justify-center text-muted-foreground">
						<div class="space-y-2 text-center">
							<TerminalIcon class="mx-auto h-12 w-12 opacity-50" />
							<p class="font-medium">No logs yet</p>
							<p class="text-sm text-muted-foreground">Service operation logs will appear here</p>
						</div>
					</div>
				{/if}
			</div>
		</div>
	</Sheet.Content>
</Sheet.Root>
