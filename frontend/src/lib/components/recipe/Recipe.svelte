<script lang="ts">
	import { formatDuration } from '$lib/time';
	import { resolve } from '$app/paths';
	import type { RecipeAndOwnerType } from '$lib/recipes';
	interface Props {
		recipe: RecipeAndOwnerType;
	}

	let { recipe }: Props = $props();
</script>

<a
	class="h-fit w-fit rounded-lg border-solid border-gray-400/50 p-3 shadow-none transition-shadow duration-250 hover:shadow-[0_0_12px_rgba(0,0,0,0.5)]"
	href={resolve('/recipes/[id]', {
		id: recipe.recipe.id.toString()
	})}
>
	{#if recipe.recipe.image_url !== undefined}
		<div class="h-40 w-[260px] overflow-hidden rounded-lg border-2 border-solid border-black">
			<img
				src={recipe.recipe.image_url}
				alt="cover"
				class="h-full w-full object-cover object-center"
			/>
		</div>
	{:else}
		<div class="h-40 w-[260px] rounded-lg border border-solid border-black bg-cyan-300"></div>
	{/if}

	<h1 class="text-lg font-semibold">{recipe.recipe.title}</h1>
	<h2 class="text-sm capitalize">{recipe.owner.first_name} {recipe.owner.last_name}</h2>
	<h3 class="text-sm text-gray-400">{formatDuration(recipe.recipe.cook_time_minutes)}</h3>
</a>
