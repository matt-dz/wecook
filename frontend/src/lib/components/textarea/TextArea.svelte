<script lang="ts">
	import type { HTMLTextareaAttributes } from 'svelte/elements';
	import { twMerge } from 'tailwind-merge';
	import clsx from 'clsx';

	interface Props extends Omit<HTMLTextareaAttributes, 'class'> {
		className?: string;
	}

	let { className, value = $bindable(), ...others }: Props = $props();
</script>

<textarea
	bind:value
	class={twMerge(
		clsx(
			'rounded-md border border-solid border-gray-400 bg-gray-200/50 py-[3px] pl-2 font-inter placeholder:text-sm focus:ring focus:outline-none disabled:cursor-not-allowed disabled:text-gray-500',
			className
		)
	)}
	{...others}
>
</textarea>
