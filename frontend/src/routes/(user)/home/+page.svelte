<script lang="ts">
	import type { PageProps } from './$types';
	import Recipe from '$lib/components/recipe/Recipe.svelte';
	import Button from '$lib/components/button/Button.svelte';

	let { data }: PageProps = $props();
	console.log(data);
</script>

<div class="mt-8 mb-4 flex justify-center">
	<div class="flex w-full max-w-5xl justify-between border-b border-solid border-gray-300 pb-2">
		<h1 class="text-xl">Recipes</h1>
		<Button className="text-sm">New Recipe</Button>
	</div>
</div>

<div class="flex justify-center">
	{#if !data.recipes?.recipes}
		<div
			class="grid w-full max-w-5xl grid-cols-1 place-items-center items-center px-6 min-[650px]:grid-cols-2 min-[900px]:grid-cols-3"
		>
			{#each data.recipes?.recipes as recipe (recipe.recipe.id)}
				<Recipe {recipe} />
			{/each}
		</div>
	{:else}
		<h1 class="mt-4 text-gray-500">No Recipes Yet!</h1>
	{/if}
</div>
