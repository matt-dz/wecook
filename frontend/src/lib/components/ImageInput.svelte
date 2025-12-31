<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props extends HTMLInputAttributes {
		onchange?: (event: Event & { currentTarget: HTMLInputElement }) => void;
		ref?: HTMLInputElement;
	}

	let { onchange, ref = $bindable(), ...restProps }: Props = $props();

	// Supported MIME types matching backend validation
	const ACCEPTED_IMAGE_TYPES = [
		'image/jpeg',
		'image/png',
		'image/webp',
		'image/avif',
		'image/heic',
		'image/heif',
		'image/gif',
		'image/svg+xml',
		'image/bmp',
		'image/tiff'
	].join(',');
</script>

<input type="file" bind:this={ref} accept={ACCEPTED_IMAGE_TYPES} {onchange} {...restProps} />
