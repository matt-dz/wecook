<script lang="ts">
	import type { PageProps } from './$types';
	import Recipe from '$lib/components/recipe/Recipe.svelte';

	let { data }: PageProps = $props();
	const recipes = data.recipes;
</script>

<div class="mt-8 mb-4 flex justify-center px-6">
	<div class="flex w-full max-w-5xl justify-between border-b border-solid border-gray-300 pb-2">
		<h1 class="text-xl">Recipes</h1>
	</div>
</div>

<div class="flex justify-center">
	{#if recipes.recipes.length > 0}
		<div
			class="grid w-full max-w-5xl grid-cols-1 place-items-center items-center gap-4 min-[650px]:grid-cols-2 min-[900px]:grid-cols-3"
		>
			{#each recipes.recipes as recipe (recipe.recipe.id)}
				<Recipe
					className="h-[60vw] w-[90vw] min-[650px]:h-[25vw] min-[650px]:w-[40vw] min-[900px]:h-[20vw] min-[900px]:w-[30vw] min-[900px]:max-h-[196px] min-[900px]:max-w-[296px]"
					{recipe}
				/>
			{/each}
		</div>
	{:else}
		<h1 class="mt-4 text-gray-500">No Recipes Yet!</h1>
	{/if}
</div>
