<script lang="ts">
	import { formatDuration } from '$lib/time';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import ShareDialog from '$lib/components/share-dialog/Dialog.svelte';
	import type { RecipeWithStepsIngredientsAndOwner } from '$lib/recipes';
	import { Share2 } from '@lucide/svelte';

	interface Props {
		recipe: RecipeWithStepsIngredientsAndOwner;
	}

	let { recipe }: Props = $props();
	let shareDialogOpen = $state(false);

	const title =
		recipe.recipe.title.trim().length > 0 ? recipe.recipe.title.trim() : 'Untitled Recipe';
</script>

<svelte:head>
	<title>{title} | WeCook</title>
</svelte:head>

<Tooltip.Provider>
	<div class="mt-16 mb-16 flex justify-center px-6">
		<div class="w-full max-w-3xl">
			<div class="mb-8">
				<div>
					<h1 class="text-3xl font-semibold">{title}</h1>
					<h2 class="text-lg capitalize">
						{recipe.owner.first_name}
						{recipe.owner.last_name}
					</h2>
					<h3 class="text-lg text-gray-500">
						{#if recipe.recipe.prep_time_amount && recipe.recipe.prep_time_unit}
							Prep Time: {formatDuration(
								recipe.recipe.prep_time_amount,
								recipe.recipe.prep_time_unit
							)}
						{/if}
					</h3>
					<h3 class="text-lg text-gray-500">
						{#if recipe.recipe.cook_time_amount && recipe.recipe.cook_time_unit}
							Cook Time: {formatDuration(
								recipe.recipe.cook_time_amount,
								recipe.recipe.cook_time_unit
							)}
						{/if}
					</h3>
				</div>

				<Tooltip.Root>
					<Tooltip.Trigger>
						<button
							class="mt-2 cursor-pointer rounded-lg bg-gray-100 p-1.5 hover:bg-gray-200"
							onclick={() => (shareDialogOpen = true)}
						>
							<Share2 size={20} strokeWidth={1.5} />
						</button>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>Share Recipe</p>
					</Tooltip.Content>
				</Tooltip.Root>

				{#if recipe.recipe.image_url}
					<div class="mt-6 flex justify-center">
						<img src={recipe.recipe.image_url} alt="cover" class="h-full max-h-96 object-cover" />
					</div>
				{/if}
			</div>

			<p class="whitespace-pre-wrap">{recipe.recipe.description}</p>

			{#if recipe.recipe.ingredients}
				<h1 class="mt-12 mb-2 text-3xl">Ingredients</h1>
				<ul class="list-inside list-disc space-y-2">
					{#each recipe.recipe.ingredients as ingredient (ingredient.id)}
						<li>
							<p class="inline-block whitespace-pre-wrap">{ingredient.description}</p>
							{#if ingredient.image_url}
								<div class="mt-2 ml-6">
									<img
										src={ingredient.image_url}
										alt={ingredient.description || 'Ingredient'}
										class="h-full max-h-96 object-cover"
									/>
								</div>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}

			{#if recipe.recipe.steps}
				<h1 class="mt-12 mb-2 text-3xl">Steps</h1>
				<ol class="steps list-inside space-y-4">
					{#each recipe.recipe.steps as step (step.id)}
						<li>
							<p class="whitespace-pre-wrap">{step.instruction}</p>
							{#if step.image_url}
								<div class="mt-2 ml-6">
									<img
										src={step.image_url}
										alt={`Step ${step.step_number}`}
										class="h-full max-h-96 object-cover"
									/>
								</div>
							{/if}
						</li>
					{/each}
				</ol>
			{/if}
		</div>
	</div>
</Tooltip.Provider>

<ShareDialog bind:open={shareDialogOpen} recipeId={recipe.recipe.id} />

<style>
	.steps {
		counter-reset: step;
	}

	.steps > li {
		counter-increment: step;
	}

	.steps > li::marker {
		content: 'Step ' counter(step) '.';
		font-weight: 600;
		font-size: 1.125rem;
	}
</style>
