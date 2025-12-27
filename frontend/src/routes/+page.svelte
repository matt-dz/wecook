<script lang="ts">
	import type { PageProps } from './$types';
	import Recipe from '$lib/components/recipe/Recipe.svelte';
	import * as Empty from '$lib/components/ui/empty';
	import { CookingPot } from '@lucide/svelte';

	let { data }: PageProps = $props();
	const recipes = data.recipes;
</script>

<div class="mt-8 mb-4 flex justify-center px-6">
	<div class="flex w-full max-w-5xl justify-between border-b border-solid border-gray-300 pb-2">
		<h1 class="text-xl">Recipes</h1>
	</div>
</div>

<div class="mb-8 flex justify-center px-6">
	{#if recipes.recipes.length > 0}
		<div
			class="grid w-full max-w-5xl grid-cols-1 place-items-center items-center gap-2 min-[650px]:grid-cols-2 min-[1050px]:grid-cols-3"
		>
			{#each recipes.recipes as recipe (recipe.recipe.id)}
				<Recipe
					className="h-[60vw] w-[90vw] min-[650px]:h-[27vw] min-[650px]:w-[43vw] min-[1050px]:h-[20vw] min-[1050px]:w-[30vw] min-[1050px]:max-h-[196px] min-[1050px]:max-w-[296px]"
					{recipe}
				/>
			{/each}
		</div>
	{:else}
		<Empty.Root>
			<Empty.Header>
				<Empty.Media variant="icon">
					<CookingPot />
				</Empty.Media>
				<Empty.Title>No Recipes</Empty.Title>
				<Empty.Description>No recipes found. Check back later!</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{/if}
</div>
