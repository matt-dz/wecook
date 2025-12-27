<script lang="ts">
	import clsx from 'clsx';
	import { cn } from 'tailwind-variants';
	interface Props {
		error: string;
		class?: string;
	}
	const placeholderError = 'placeholder';

	let { error = $bindable(), ...rest }: Props = $props();

	let errMsg = $derived.by(() => {
		if (error.trim().length === 0) {
			return placeholderError;
		}
		return error;
	});
</script>

<p
	class={cn(clsx('text-sm text-red-600', rest.class))}
	class:invisible={errMsg === placeholderError}
	class:visible={errMsg !== placeholderError}
>
	{errMsg}
</p>
