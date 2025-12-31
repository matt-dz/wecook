<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';
	import { validateImageSize } from '$lib/utils';
	import { toast } from 'svelte-sonner';

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

	const handleChange = (event: Event & { currentTarget: HTMLInputElement }) => {
		const target = event.currentTarget;
		const file = target.files?.[0];

		if (file) {
			if (!validateImageSize(file, 1)) {
				toast.error('Image too large. Maximum size is 20 MB.');
				// Clear the file input
				target.value = '';
				return;
			}
		}

		// Call the original onchange handler if validation passed
		onchange?.(event);
	};
</script>

<input
	type="file"
	bind:this={ref}
	accept={ACCEPTED_IMAGE_TYPES}
	onchange={handleChange}
	{...restProps}
/>
