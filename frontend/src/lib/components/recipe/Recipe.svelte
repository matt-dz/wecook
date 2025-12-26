<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { formatDuration, toMinutes } from '$lib/time';
	import { resolve } from '$app/paths';
	import { twMerge } from 'tailwind-merge';
	import { EllipsisVertical } from '@lucide/svelte';
	import clsx from 'clsx';
	import type { RecipeAndOwner } from '$lib/recipes';
	interface Props {
		recipe: RecipeAndOwner;
		className?: string;
		editable?: boolean;
		personal?: boolean;
	}

	let { recipe, className, editable = false, personal = false }: Props = $props();

	const totalCookTime =
		(recipe.recipe?.cook_time_amount && recipe.recipe?.cook_time_unit
			? toMinutes(recipe.recipe.cook_time_amount, recipe.recipe.cook_time_unit)
			: 0) +
		(recipe.recipe?.prep_time_amount && recipe.recipe?.prep_time_unit
			? toMinutes(recipe.recipe.prep_time_amount, recipe.recipe.prep_time_unit)
			: 0);
</script>

<a
	class="h-fit w-fit rounded-3xl border-solid border-gray-400/50 p-3 shadow-none transition-shadow duration-250 hover:shadow-[0_0_12px_rgba(0,0,0,0.5)]"
	href={resolve(
		personal ? `/recipes/${recipe.recipe.id}/personal` : `/recipes/${recipe.recipe.id}`
	)}
>
	{#if recipe.recipe.image_url !== undefined}
		<div
			class={twMerge(
				clsx(
					'h-40 w-[260px] overflow-hidden rounded-xl border-2 border-solid border-black',
					className
				)
			)}
		>
			<img
				src={recipe.recipe.image_url}
				alt="cover"
				class="h-full w-full object-cover object-center"
			/>
		</div>
	{:else}
		<div
			class={twMerge(
				clsx('h-40 w-[260px] rounded-xl border border-solid border-black bg-blue-200', className)
			)}
		></div>
	{/if}

	<div class="mt-1 flex items-start">
		<div class="flex-1">
			<h1 class="text-lg font-semibold">{recipe.recipe.title}</h1>
			<h2 class="text-sm capitalize">{recipe.owner.first_name} {recipe.owner.last_name}</h2>
			<h3 class="text-sm text-gray-400">{formatDuration(totalCookTime, 'minutes')}</h3>
		</div>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<button {...props} class="-mr-2 rounded-full p-1 hover:bg-gray-200">
						<EllipsisVertical />
					</button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content>
				<DropdownMenu.Group>
					{#if editable}
						<DropdownMenu.Item>
							<a class="w-full" href={resolve(`/recipes/${recipe.recipe.id}/edit`)}> Edit </a>
						</DropdownMenu.Item>
						<DropdownMenu.Item>Publish</DropdownMenu.Item>
					{/if}
					<DropdownMenu.Item>Share</DropdownMenu.Item>
				</DropdownMenu.Group>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>
</a>
