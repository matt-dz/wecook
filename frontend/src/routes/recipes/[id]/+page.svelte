<script lang="ts">
	import type { PageProps } from './$types';
	import { toMinutes, formatDuration } from '$lib/time';
	import Input from '$lib/components/input/Input.svelte';

	let { data }: PageProps = $props();
	let portion: number = $state(1);
	const recipe = data.recipe;

	let ingredients = $derived.by(() => {
		const adjustedPortion = portion === null || portion <= 0 ? 1 : portion;
		return data.recipe.recipe.ingredients.map((i) => ({
			...i,
			quantity: (i.quantity ?? 0) * adjustedPortion
		}));
	});

	const onlyPositiveNumbers = (e: KeyboardEvent) => {
		const invalid = ['e', 'E', '+', '-'];
		if (invalid.includes(e.key)) e.preventDefault();
	};

	const totalCookTime =
		(recipe.recipe?.cook_time_amount && recipe.recipe?.cook_time_unit
			? toMinutes(recipe.recipe.cook_time_amount, recipe.recipe.cook_time_unit)
			: 0) +
		(recipe.recipe?.prep_time_amount && recipe.recipe?.prep_time_unit
			? toMinutes(recipe.recipe.prep_time_amount, recipe.recipe.prep_time_unit)
			: 0);
</script>

<svelte:head>
	<title>{data.recipe.recipe.title}</title>
</svelte:head>

<div class="mt-16 mb-16 flex justify-center px-6">
	<div class="w-full max-w-5xl">
		<div class="mb-12 flex justify-between">
			<div>
				<h1 class="text-3xl font-semibold">{data.recipe.recipe.title}</h1>
				<h2 class="text-lg capitalize">
					{data.recipe.owner.first_name}
					{data.recipe.owner.last_name}
				</h2>
				<h3 class="text-lg text-gray-500">
					Cook Time: {formatDuration(totalCookTime)}
				</h3>
			</div>

			<div class="h-96 w-lg">
				<img
					src={data.recipe.recipe.image_url}
					alt="cover"
					class="h-full w-full object-cover object-center"
				/>
			</div>
		</div>

		<p>{data.recipe.recipe.description}</p>

		{#if data.recipe.recipe.ingredients}
			<h1 class="mt-12 mb-2 text-3xl">Ingredients</h1>
			<div class="mb-4 flex flex-col">
				<label for="portion" class="font-inter"
					>Portion <span class="text-sm text-gray-500">(Default 1)</span></label
				>
				<Input
					bind:value={portion}
					name="portion"
					onkeydown={onlyPositiveNumbers}
					className="w-fit"
					type="number"
					min={1}
					defaultValue={1}
				/>
			</div>
			<ul class="list-inside list-disc">
				{#each ingredients as ingredient (ingredient.id)}
					<li>
						{ingredient.quantity}{ingredient.unit && ` ${ingredient.unit} of`}
						{ingredient.name}
					</li>
				{/each}
			</ul>
		{/if}

		{#if data.recipe.recipe.steps}
			<h1 class="mt-12 mb-2 text-3xl">Steps</h1>
			<ol class="list-inside list-decimal">
				{#each data.recipe.recipe.steps as step (step.id)}
					<li>{step.instruction}</li>
				{/each}
			</ol>
		{/if}
	</div>
</div>
