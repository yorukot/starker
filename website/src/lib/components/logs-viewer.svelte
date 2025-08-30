<script lang="ts">
	import TerminalIcon from '~icons/lucide/terminal';

	interface LogMessage {
		id: string;
		timestamp: string;
		type: 'log' | 'error' | 'info' | 'status' | 'step';
		message: string;
	}

	interface Props {
		messages: LogMessage[];
		title?: string;
		description?: string;
		showEmpty?: boolean;
		class?: string;
	}

	let {
		messages,
		title = 'Logs',
		description = 'Real-time output',
		showEmpty = true,
		class: className = ''
	}: Props = $props();

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
			case 'step':
				return 'text-blue-700 bg-blue-50 border-l-blue-500 dark:text-blue-300 dark:bg-blue-950/50 dark:border-l-blue-400';
			default:
				return 'text-muted-foreground bg-muted/50 border-l-border';
		}
	}

	// Auto-scroll to bottom when new messages arrive
	$effect(() => {
		if (messages?.length > 0 && logContainer) {
			setTimeout(() => {
				logContainer.scrollTop = logContainer.scrollHeight;
			}, 10);
		}
	});
</script>

<div class="flex flex-col {className}">
	{#if title || description}
		<div class="mb-4">
			{#if title}
				<div class="mb-1 flex items-center gap-2">
					<TerminalIcon class="h-5 w-5" />
					<h3 class="text-lg font-semibold">{title}</h3>
				</div>
			{/if}
			{#if description}
				<p class="text-sm text-muted-foreground">{description}</p>
			{/if}
		</div>
	{/if}

	<div
		bind:this={logContainer}
		class="h-full space-y-1 overflow-y-auto rounded-lg border bg-background p-3"
	>
		{#each messages || [] as log (log.id)}
			<div class="rounded-r border-l-4 py-2 pl-3 transition-colors {getLogMessageClass(log.type)}">
				<div class="flex min-w-0 items-start gap-3 text-sm">
					<span class="mt-0.5 min-w-[70px] font-mono text-xs text-muted-foreground">
						{log.timestamp}
					</span>
					<span class="mt-0.5 min-w-[60px] text-xs font-medium tracking-wide uppercase">
						{log.type}
					</span>
					<span
						class="word-break overflow-wrap-anywhere flex-1 font-mono leading-5 break-words whitespace-pre-wrap"
					>
						{log.message}
					</span>
				</div>
			</div>
		{/each}

		{#if (messages?.length || 0) === 0 && showEmpty}
			<div class="flex h-full items-center justify-center text-muted-foreground">
				<div class="space-y-2 text-center">
					<TerminalIcon class="mx-auto h-12 w-12 opacity-50" />
					<p class="font-medium">No logs yet</p>
					<p class="text-sm text-muted-foreground">Logs will appear here</p>
				</div>
			</div>
		{/if}
	</div>
</div>
