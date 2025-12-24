<script lang="ts">
	import type { PageProps } from './$types';
	import { toMinutes, formatDuration } from '$lib/time';
	import Input from '$lib/components/input/Input.svelte';

	let { data }: PageProps = $props();
	let portion: number = $state(1);

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

	const formatLocale = (n: number, decimals: number) => {
		return new Intl.NumberFormat(undefined, {
			maximumFractionDigits: decimals
		}).format(n);
	};

	const totalCookTime =
		(data.recipe.recipe?.cook_time_amount && data.recipe.recipe?.cook_time_unit
			? toMinutes(data.recipe.recipe.cook_time_amount, data.recipe.recipe.cook_time_unit)
			: 0) +
		(data.recipe.recipe?.prep_time_amount && data.recipe.recipe?.prep_time_unit
			? toMinutes(data.recipe.recipe.prep_time_amount, data.recipe.recipe.prep_time_unit)
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

			{#if data.recipe.recipe.image_url}
				<div class="h-96 w-lg">
					<img
						src={data.recipe.recipe.image_url}
						alt="cover"
						class="h-full w-full object-cover object-center"
					/>
				</div>
			{/if}
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
			<ul class="list-inside list-disc space-y-2">
				{#each ingredients as ingredient (ingredient.id)}
					<li>
						<div class="inline-block">
							{formatLocale(ingredient.quantity, 3)}{ingredient.unit && ` ${ingredient.unit} of`}
							{ingredient.name}
						</div>
						{#if ingredient.image_url}
							<div class="mt-2 ml-6">
								<img
									src={ingredient.image_url}
									alt={ingredient.name || 'Ingredient'}
									class="h-full max-h-96 object-cover"
								/>
							</div>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}

		{#if data.recipe.recipe.steps}
			<h1 class="mt-12 mb-2 text-3xl">Steps</h1>
			<ol class="list-inside list-decimal space-y-2">
				{#each data.recipe.recipe.steps as step (step.id)}
					<li>
						<div class="inline-block">{step.instruction}</div>
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
