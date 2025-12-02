<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { twMerge } from 'tailwind-merge';
	import clsx from 'clsx';

	interface Props extends Omit<HTMLButtonAttributes, 'class'> {
		className?: string;
		children?: Snippet;
	}

	let { className, children, ...others }: Props = $props();
</script>

<button
	class={twMerge(
		clsx(
			'cursor-pointer rounded-full border border-solid border-gray-500 bg-linear-to-t from-gray-300 to-gray-200 px-4 py-0.5 font-inter transition-all hover:from-gray-200 hover:to-gray-100 disabled:cursor-not-allowed disabled:border-gray-500 disabled:bg-gray-400 disabled:bg-none disabled:text-gray-600',
			className
		)
	)}
	{...others}
>
	{@render children?.()}
</button>
