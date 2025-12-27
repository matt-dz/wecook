<script lang="ts">
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	interface Props {
		status: 'saved' | 'saving' | 'failed' | 'loading';
	}
	let { status }: Props = $props();
</script>

<div
	class="flex w-fit items-center gap-2 rounded-full border border-solid bg-white px-3 py-1.5 shadow-xs"
>
	{#if status === 'saving'}
		<Spinner class="text-blue-500" />
	{:else if status === 'saved'}
		<div class="flex h-4 w-4 items-center justify-center rounded-full bg-green-500/30">
			<div class="h-2 w-2 rounded-full bg-green-500/70"></div>
		</div>
	{:else if status === 'failed'}
		<div class="flex h-4 w-4 items-center justify-center rounded-full bg-red-500/30">
			<div class="h-2 w-2 rounded-full bg-red-500/70"></div>
		</div>
	{:else if status === 'loading'}
		<Spinner class="text-gray-400" />
	{/if}
	<p class="font-inter text-sm capitalize">{status}</p>
</div>
