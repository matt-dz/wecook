<script lang="ts">
	import type { PageProps } from './$types';
	import {
		updatePersonalRecipe,
		updateIngredient,
		createIngredient,
		updateStep,
		type TimeUnitType,
		type UpdateRecipeRequest,
		type UpdateIngredientRequest,
		type UpdateStepRequest,
		createStep,
		deleteIngredient,
		deleteStep,
		uploadIngredientImage,
		deleteIngredientImage
	} from '$lib/recipes';
	import fetch from '$lib/http';
	import Input from '$lib/components/input/Input.svelte';
	import StepInput from '$lib/components/step/Input.svelte';
	import IngredientInput from '$lib/components/ingredient/Input.svelte';
	import TextArea from '$lib/components/textarea/TextArea.svelte';
	import TimeunitMenu from '$lib/components/timeunit-menu/TimeunitMenu.svelte';
	import Button from '$lib/components/button/Button.svelte';
	import { debounce } from '$lib/debounce';
	import { HTTPError } from 'ky';
	import clsx from 'clsx';

	let { data }: PageProps = $props();

	let title: string | undefined = $state(data.recipe.recipe.title);
	let description: string | undefined = $state(data.recipe.recipe.description);
	let servings: number | undefined = $state(data.recipe.recipe.servings);
	let cookTime: number | undefined = $state(data.recipe.recipe.cook_time_amount);
	let cookTimeUnit: TimeUnitType | undefined = $state(data.recipe.recipe.cook_time_unit);
	let prepTime: number | undefined = $state(data.recipe.recipe.prep_time_amount);
	let prepTimeUnit: TimeUnitType | undefined = $state(data.recipe.recipe.prep_time_unit);
	let ingredients = $state(data.recipe.recipe.ingredients);
	let steps = $state(data.recipe.recipe.steps);
	let published = $state(data.recipe.recipe.published);

	const debounceDelay = 200;

	const updateRecipeField = debounce(
		async (field: keyof UpdateRecipeRequest, value: UpdateRecipeRequest[typeof field]) => {
			await updatePersonalRecipe(fetch, {
				[field]: value,
				recipe_id: data.recipe.recipe.id
			});
		},
		debounceDelay
	);

	const updateIngredientField = debounce(
		async (
			ingredientID: number,
			field: keyof UpdateIngredientRequest,
			value: UpdateIngredientRequest[typeof field]
		) => {
			await updateIngredient(fetch, {
				[field]: value,
				recipe_id: data.recipe.recipe.id,
				ingredient_id: ingredientID
			});
		},
		debounceDelay
	);

	const updateStepField = debounce(
		async (
			stepID: number,
			field: keyof UpdateStepRequest,
			value: UpdateStepRequest[typeof field]
		) => {
			await updateStep(fetch, {
				[field]: value,
				recipe_id: data.recipe.recipe.id,
				step_id: stepID
			});
		},
		debounceDelay
	);

	const onTitleChange = () => title !== undefined && updateRecipeField('title', title);
	const onDescriptionChange = () =>
		description !== undefined && updateRecipeField('description', description);
	const onServingsChange = () => servings !== undefined && updateRecipeField('servings', servings);
	const onPrepTimeChange = () =>
		prepTime !== undefined && updateRecipeField('prep_time_amount', prepTime);
	const onPrepTimeUnitChange = () =>
		prepTimeUnit !== undefined && updateRecipeField('prep_time_unit', prepTimeUnit);
	const onCookTimeChange = () =>
		cookTime !== undefined && updateRecipeField('cook_time_amount', cookTime);
	const onCookTimeUnitChange = () =>
		cookTimeUnit !== undefined && updateRecipeField('cook_time_unit', cookTimeUnit);

	const onIngredientQuantityChange = (ingredientID: number) => {
		const ingredient = ingredients.find((i) => i.id === ingredientID);
		if (!ingredient) return;
		updateIngredientField(ingredientID, 'quantity', ingredient.quantity);
	};
	const onIngredientUnitChange = (ingredientID: number) => {
		const ingredient = ingredients.find((i) => i.id === ingredientID);
		if (!ingredient) return;
		updateIngredientField(ingredientID, 'unit', ingredient.unit);
	};
	const onIngredientNameChange = (ingredientID: number) => {
		const ingredient = ingredients.find((i) => i.id === ingredientID);
		if (!ingredient) return;
		updateIngredientField(ingredientID, 'name', ingredient.name);
	};

	const onStepInstructionChange = (stepID: number) => {
		const step = steps.find((s) => s.id === stepID);
		if (!step) return;
		updateStepField(stepID, 'instruction', step.instruction);
	};

	const onIngredientImageUpload = async (ingredientID: number, image: File) => {
		try {
			const res = await uploadIngredientImage(fetch, {
				recipe_id: data.recipe.recipe.id,
				ingredient_id: ingredientID,
				image
			});
			const idx = ingredients.findIndex((i) => i.id === ingredientID);
			ingredients = [...ingredients.slice(0, idx), res, ...ingredients.slice(idx + 1)];
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to upload image', e.message);
			} else {
				console.error(e);
			}
			alert('failed to upload image. try again later.');
		}
	};

	const onIngredientImageDeletion = async (ingredientID: number) => {
		try {
			await deleteIngredientImage(fetch, {
				recipe_id: data.recipe.recipe.id,
				ingredient_id: ingredientID
			});
			ingredients = ingredients.map((i) =>
				i.id !== ingredientID ? i : { ...i, image_url: undefined }
			);
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to delete image', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete image. try again later.');
		}
	};

	const togglePublish = async () => {
		try {
			await updatePersonalRecipe(fetch, {
				recipe_id: data.recipe.recipe.id,
				published: !published
			});
			published = !published;
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to publish recipe', e.message);
			} else {
				console.error(e);
			}
			alert('failed to publish recipe. try again later.');
		}
	};

	const handleCreateIngredient = async () => {
		try {
			const newIngredient = await createIngredient(fetch, { recipe_id: data.recipe.recipe.id });
			ingredients = [...ingredients, newIngredient];
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to create ingredient', e.message);
			} else {
				console.error(e);
			}
			alert('failed to create ingredient. try again later.');
		}
	};

	const handleCreateStep = async () => {
		try {
			const newStep = await createStep(fetch, { recipe_id: data.recipe.recipe.id });
			steps = [...steps, newStep];
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to create step', e.message);
			} else {
				console.error(e);
			}
			alert('failed to create step. try again later.');
		}
	};

	const handleDeleteIngredient = async (ingredientID: number) => {
		try {
			await deleteIngredient(fetch, {
				recipe_id: data.recipe.recipe.id,
				ingredient_id: ingredientID
			});
			ingredients = ingredients.filter((i) => i.id !== ingredientID);
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to delete ingredient', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete ingredient. try again later.');
		}
	};

	const handleDeleteStep = async (stepID: number) => {
		try {
			await deleteStep(fetch, {
				recipe_id: data.recipe.recipe.id,
				step_id: stepID
			});
			const idx = steps.findIndex((s) => s.id === stepID);
			if (idx !== -1) {
				steps = [
					...steps.slice(0, idx),
					...steps.slice(idx + 1).map((s) => ({ ...s, step_number: s.step_number - 1 }))
				];
			}
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to delete step', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete step. try again later.');
		}
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
				oninput={onTitleChange}
			/>
		</div>

		<div class="flex flex-col gap-1">
			<label for="description" class="text-lg">Description</label>
			<TextArea
				name="description"
				bind:value={description}
				className="font-IowanOldStyleBT"
				oninput={onDescriptionChange}
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
					oninput={onServingsChange}
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
							oninput={onPrepTimeChange}
						/>
						<TimeunitMenu bind:value={prepTimeUnit} onValueChange={onPrepTimeUnitChange} />
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
							oninput={onCookTimeChange}
						/>
						<TimeunitMenu bind:value={cookTimeUnit} onValueChange={onCookTimeUnitChange} />
					</div>
				</div>
			</div>
		</div>

		<div>
			<h1 class="mb-2 text-2xl">Ingredients</h1>
			<div class="flex flex-col gap-2">
				{#each ingredients as ingredient, idx (ingredient.id)}
					<IngredientInput
						bind:ingredient={ingredients[idx]}
						onQuantityChange={onIngredientQuantityChange}
						onUnitChange={onIngredientUnitChange}
						onNameChange={onIngredientNameChange}
						onDelete={handleDeleteIngredient}
						onImageUpload={onIngredientImageUpload}
						onImageDeletion={onIngredientImageDeletion}
					/>
				{/each}
			</div>
			<Button onclick={handleCreateIngredient} className="font-medium text-sm mt-4"
				>Add Ingredient</Button
			>
		</div>

		<div>
			<h1 class="mb-2 text-2xl">Steps</h1>
			<div class="flex flex-col gap-4">
				{#each steps as step, i (step.id)}
					<div class="w-full">
						<StepInput
							bind:step={steps[i]}
							onInstructionChange={() => onStepInstructionChange(step.id)}
							onDelete={() => handleDeleteStep(step.id)}
						/>
					</div>
				{/each}
			</div>
			<Button onclick={handleCreateStep} className="font-medium text-sm mt-4">Add Step</Button>
		</div>

		<Button
			onclick={togglePublish}
			className={clsx(
				'text-sm font-medium w-fit mt-6',
				!published && 'from-green-300 to-green-200 hover:from-green-200 hover:to-green-100',
				published && 'from-red-300 to-red-200 hover:from-red-200 hover:to-red-100'
			)}>{published ? 'Unpublish' : 'Publish'}</Button
		>
	</div>
</div>
