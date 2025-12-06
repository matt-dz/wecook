<script lang="ts">
	import Recipe from '$lib/components/recipe/Recipe.svelte';
	import Button from '$lib/components/button/Button.svelte';
	import type { PageProps } from './$types';
	import { CreateRecipe } from '$lib/recipes';
	import fetch, { isRetryable } from '$lib/http';
	import { HTTPError, TimeoutError } from 'ky';
	import { refreshTokenExpired } from '$lib/errors/api';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let { data }: PageProps = $props();

	const createNewRecipe = async () => {
		try {
			const recipe = await CreateRecipe(fetch);
			goto(resolve(`/recipes/${recipe.recipe_id.toString()}/edit`));
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error(e.response);
				if (await refreshTokenExpired(e.response)) {
					goto(resolve('/login'));
				} else if (isRetryable(e.response)) {
					alert('something went wrong. try again later.');
				} else {
					alert('uh-oh, something bad happened.');
				}
			} else if (e instanceof TimeoutError) {
				alert('request timed out. try again later.');
			} else {
				alert('uh-oh, an un-retryable error occured.');
				console.error(e);
			}
		}
	};
</script>

<div class="mt-8 mb-4 flex justify-center px-6">
	<div class="flex w-full max-w-5xl justify-between border-b border-solid border-gray-300 pb-2">
		<h1 class="text-xl">Recipes</h1>
		<Button onclick={createNewRecipe} className="text-sm">New Recipe</Button>
	</div>
</div>

<div class="flex justify-center">
	{#if data.recipes?.recipes}
		<div
			class="grid w-full max-w-5xl grid-cols-1 place-items-center items-center gap-4 min-[650px]:grid-cols-2 min-[900px]:grid-cols-3"
		>
			{#each data.recipes?.recipes as recipe (recipe.recipe.id)}
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
