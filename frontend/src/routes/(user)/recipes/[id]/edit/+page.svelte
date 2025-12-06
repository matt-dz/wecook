<script lang="ts">
	import type { PageProps } from './$types';
	import { type TimeUnitType } from '$lib/recipes';
	import Input from '$lib/components/input/Input.svelte';
	import TextArea from '$lib/components/textarea/TextArea.svelte';
	import DropdownMenu from '$lib/components/dropdown-menu/DropdownMenu.svelte';
	import Button from '$lib/components/button/Button.svelte';

	let { data }: PageProps = $props();

	type ingredientInput = {
		name?: string;
		unit?: string;
		quantity?: number;
		place: number;
		id?: number;
	};

	type stepInput = {
		instruction?: string;
		image_url?: string;
		place: number;
		id?: number;
	};

	let ingredients: ingredientInput[] = $state(
		data.recipe?.recipe.ingredients.map((i, idx) => ({
			name: i.name,
			unit: i.unit,
			quantity: i.quantity,
			place: idx,
			id: i.id
		})) ?? []
	);

	let steps: stepInput[] = $state(
		data.recipe?.recipe.steps.map((i, idx) => ({
			instruction: i.instruction,
			image_url: i.image_url,
			place: idx,
			id: i.id
		})) ?? []
	);

	let title: string | undefined = $state(data.recipe?.recipe.title);
	let description: string | undefined = $state(data.recipe?.recipe.description);
	let servings: number | undefined = $state(data.recipe?.recipe.servings);
	let cookTime: number | undefined = $state(data.recipe?.recipe.cook_time_amount);
	let cookTimeUnit: TimeUnitType | undefined = $state(data.recipe?.recipe.cook_time_unit);
	let prepTime: number | undefined = $state(data.recipe?.recipe.prep_time_amount);
	let prepTimeUnit: TimeUnitType | undefined = $state(data.recipe?.recipe.prep_time_unit);

	const newIngredient = () => {
		ingredients = [
			...ingredients,
			{
				name: undefined,
				unit: undefined,
				quantity: undefined,
				place: Math.max(...ingredients.map((i) => i.place), 1) + 1
			}
		];
	};

	const newStep = () => {
		steps = [
			...steps,
			{
				instruction: undefined,
				image_url: undefined,
				place: Math.max(...steps.map((i) => i.place), 1) + 1,
				id: undefined
			}
		];
	};

	const updatedIngredients = () =>
		// retrieve ingredients that have been updated and added
		ingredients
			.map((i) => {
				const nameEl = document.getElementById('n-' + i.place.toString());
				const quantityEl = document.getElementById('q-' + i.place.toString());
				const unitEl = document.getElementById('u-' + i.place.toString());
				if (!nameEl || !quantityEl || !unitEl) return;

				const name = (nameEl as HTMLInputElement).value?.trim();
				const quantityStr = (quantityEl as HTMLInputElement).value?.trim();
				const unit = (unitEl as HTMLInputElement).value?.trim();

				const updated = {
					...i,
					name,
					unit
				};

				const quantity = parseFloat(quantityStr);
				if (isNaN(quantity)) return updated; // reject bad input
				updated.quantity = quantity;

				if (i.name !== updated.name || i.unit !== updated.unit || i.quantity !== updated.quantity)
					return updated;
			})
			.filter((i) => i);

	const updatedSteps = () =>
		steps
			.map((s) => {
				// TODO: add image_url validation
				const instructionEl = document.getElementById('step-' + s.place.toString());
				if (!instructionEl) return;

				const instruction = (instructionEl as HTMLTextAreaElement).value;
				if (instruction !== s.instruction) {
					return {
						...s,
						instruction
					};
				}
			})
			.filter((s) => s);

	const saveRecipe = () => {
		const ingredients = updatedIngredients();
		const steps = updatedSteps();
		console.log('ingredients', ingredients);
		console.log('steps', steps);
		console.log('title', title);
		console.log('description', description);
		console.log('cook time', cookTime);
		console.log('cook time unit', cookTimeUnit);
	};

	const onlyPositiveNumbers = (e: KeyboardEvent) => {
		const invalid = ['e', 'E', '+', '-'];
		if (invalid.includes(e.key)) e.preventDefault();
	};
</script>

<div class="mt-16 mb-12 flex w-full justify-center px-6">
	<div class="flex w-full max-w-md flex-col gap-8">
		<div class="flex flex-col gap-1">
			<label for="title" class="text-lg">Title</label>
			<Input
				name="title"
				bind:value={title}
				className="font-IowanOldStyleBT"
				defaultValue={data.recipe?.recipe.title}
			/>
		</div>

		<div class="flex flex-col gap-1">
			<label for="description" class="text-lg">Description</label>
			<TextArea
				name="description"
				bind:value={description}
				className="font-IowanOldStyleBT"
				defaultValue={data.recipe?.recipe.description}
			/>
		</div>

		<div>
			<div class="flex flex-col gap-1">
				<h2 class="text-2xl">Servings &AMP; Time</h2>
				<label for="servings" class="text-lg">Servings</label>
				<Input
					name="servings"
					onkeydown={onlyPositiveNumbers}
					type="number"
					className="w-32"
					defaultValue={1}
					bind:value={servings}
				/>
			</div>
			<div class="flex gap-8">
				<div class="mt-2 flex flex-col gap-1">
					<label for="prep time" class="text-lg">Prep Time</label>
					<div class="flex gap-2">
						<Input
							name="prep time"
							bind:value={prepTime}
							onkeydown={onlyPositiveNumbers}
							type="number"
							className="w-16"
							placeholder="30"
						/>
						<DropdownMenu bind:value={prepTimeUnit} />
					</div>
				</div>
				<div class="mt-2 flex flex-col gap-1">
					<label for="prep time" class="text-lg">Cook Time</label>
					<div class="flex gap-2">
						<Input
							name="cook time"
							onkeydown={onlyPositiveNumbers}
							type="number"
							className="w-16"
							placeholder="30"
							bind:value={cookTime}
						/>
						<DropdownMenu bind:value={cookTimeUnit} />
					</div>
				</div>
			</div>
		</div>

		<div>
			<h1 class="mb-2 text-2xl">Ingredients</h1>
			<div class="flex flex-col gap-2">
				{#each ingredients as ingredient (ingredient.place)}
					<div>
						<Input
							className="w-20"
							id={'q-' + ingredient.place}
							placeholder="Quantity"
							type="number"
							onkeydown={onlyPositiveNumbers}
							defaultValue={ingredient.quantity}
						/>
						<Input
							className="w-20"
							id={'u-' + ingredient.place}
							placeholder="Unit"
							defaultValue={ingredient.unit}
						/>
						<p class="inline-block">of</p>
						<Input className="w-60" id={'n-' + ingredient.place} defaultValue={ingredient.name} />
					</div>
				{/each}
			</div>
			<Button onclick={newIngredient} className="font-medium text-sm mt-4">Add Ingredient</Button>
		</div>

		<div>
			<h1 class="mb-2 text-2xl">Steps</h1>
			<div class="flex flex-col gap-2">
				{#each steps as step, idx (step.place)}
					<div class="w-full">
						<label for="step" class="text-lg">Step {idx + 1}</label>
						<TextArea
							id={'step-' + step.place}
							name="step"
							className="block w-full"
							placeholder="Enter instructions"
						/>
					</div>
				{/each}
			</div>
			<Button onclick={newStep} className="font-medium text-sm mt-4">Add Step</Button>
		</div>

		<div class="space-x-2">
			<Button
				onclick={saveRecipe}
				className="text-sm font-medium from-blue-300 to-blue-200 w-fit hover:from-blue-200 hover:to-blue-100 mt-6"
				>Save</Button
			>
			<Button
				className="text-sm font-medium from-green-300 to-green-200 w-fit hover:from-green-200 hover:to-green-100 mt-6"
				>Publish</Button
			>
		</div>
	</div>
</div>
