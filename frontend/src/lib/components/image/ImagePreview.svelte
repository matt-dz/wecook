<script lang="ts">
	import { X } from '@lucide/svelte';

	interface Props {
		src: string;
		alt?: string;
		onRemove?: () => void;
	}

	let { src, alt = 'Preview', onRemove }: Props = $props();
</script>

<div class="relative inline-block">
	<img {src} {alt} class="h-full max-h-96 object-cover" />
	{#if onRemove}
		<button
			onclick={onRemove}
			class="absolute -top-2 -right-2 cursor-pointer rounded-full bg-red-400 p-1 text-white hover:bg-red-600"
			type="button"
		>
			<X size={16} />
		</button>
	{/if}
</div>
