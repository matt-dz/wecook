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
		deleteIngredientImage,
		uploadStepImage,
		deleteStepImage,
		uploadRecipeImage,
		deleteRecipeImage,
		deleteRecipe
	} from '$lib/recipes';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { EllipsisVertical } from '@lucide/svelte';
	import Status from '$lib/components/status/Status.svelte';
	import { toast } from 'svelte-sonner';
	import fetch from '$lib/http';
	import UnpublishDialog from '$lib/components/unpublish-diaglog/Dialog.svelte';
	import PublishDialog from '$lib/components/publish-dialog/Dialog.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import StepInput from '$lib/components/step/Input.svelte';
	import IngredientInput from '$lib/components/ingredient/Input.svelte';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import TimeunitMenu from '$lib/components/timeunit-menu/TimeunitMenu.svelte';
	import Button from '$lib/components/button/Button.svelte';
	import ImagePreview from '$lib/components/image/ImagePreview.svelte';
	import DeleteDialog from '$lib/components/delete-recipe-dialog/Dialog.svelte';
	import { debounce } from '$lib/debounce';
	import { HTTPError } from 'ky';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

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
	let recipeImageUrl = $state<string | undefined>(data.recipe.recipe.image_url);

	let recipeFileInput: HTMLInputElement;
	let saveState: 'saved' | 'saving' | 'failed' | 'loading' = $state('saved');
	let deleteDialogOpen = $state(false);
	let publishDialogOpen = $state(false);
	let unpublishDialogOpen = $state(false);

	const debounceDelay = 200;

	const updateRecipeField = debounce(
		async (field: keyof UpdateRecipeRequest, value: UpdateRecipeRequest[typeof field]) => {
			try {
				saveState = 'saving';
				await updatePersonalRecipe(fetch, {
					[field]: value,
					recipe_id: data.recipe.recipe.id
				});
				saveState = 'saved';
			} catch (e) {
				console.error('failed to update recipe field', e);
				saveState = 'failed';
			}
		},
		debounceDelay
	);

	const updateIngredientField = debounce(
		async (
			ingredientID: number,
			field: keyof UpdateIngredientRequest,
			value: UpdateIngredientRequest[typeof field]
		) => {
			try {
				saveState = 'saving';
				await updateIngredient(fetch, {
					[field]: value,
					recipe_id: data.recipe.recipe.id,
					ingredient_id: ingredientID
				});
				saveState = 'saved';
			} catch (e) {
				console.error('failed to update ingredient', e);
				saveState = 'failed';
			}
		},
		debounceDelay
	);

	const updateStepField = debounce(
		async (
			stepID: number,
			field: keyof UpdateStepRequest,
			value: UpdateStepRequest[typeof field]
		) => {
			try {
				saveState = 'saving';
				await updateStep(fetch, {
					[field]: value,
					recipe_id: data.recipe.recipe.id,
					step_id: stepID
				});
				saveState = 'saved';
			} catch (e) {
				console.error('failed to update step field', e);
				saveState = 'failed';
			}
		},
		debounceDelay
	);

	const onTitleChange = () => title !== undefined && updateRecipeField('title', title);
	const onDescriptionChange = () => updateRecipeField('description', description ?? null);
	const onServingsChange = () => updateRecipeField('servings', servings ?? null);
	const onPrepTimeChange = () => updateRecipeField('prep_time_amount', prepTime ?? null);
	const onPrepTimeUnitChange = () => updateRecipeField('prep_time_unit', prepTimeUnit ?? null);
	const onCookTimeChange = () => updateRecipeField('cook_time_amount', cookTime ?? null);
	const onCookTimeUnitChange = () => updateRecipeField('cook_time_unit', cookTimeUnit ?? null);

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
			saveState = 'saving';
			const res = await uploadIngredientImage(fetch, {
				recipe_id: data.recipe.recipe.id,
				ingredient_id: ingredientID,
				image
			});
			saveState = 'saved';
			ingredients = ingredients.map((i) => (i.id !== ingredientID ? i : res));
		} catch (e) {
			saveState = 'failed';
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
			saveState = 'saved';
			await deleteIngredientImage(fetch, {
				recipe_id: data.recipe.recipe.id,
				ingredient_id: ingredientID
			});
			saveState = 'saved';
			ingredients = ingredients.map((i) =>
				i.id !== ingredientID ? i : { ...i, image_url: undefined }
			);
		} catch (e) {
			saveState = 'failed';
			if (e instanceof HTTPError) {
				console.error('failed to delete image', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete image. try again later.');
		}
	};

	const onStepImageUpload = async (stepID: number, image: File) => {
		try {
			saveState = 'saving';
			const res = await uploadStepImage(fetch, {
				recipe_id: data.recipe.recipe.id,
				step_id: stepID,
				image
			});
			saveState = 'saved';
			steps = steps.map((s) => (s.id !== stepID ? s : res));
		} catch (e) {
			saveState = 'failed';
			if (e instanceof HTTPError) {
				console.error('failed to upload image', e.message);
			} else {
				console.error(e);
			}
			alert('failed to upload image. try again later.');
		}
	};

	const onStepImageDeletion = async (stepID: number) => {
		try {
			saveState = 'saving';
			await deleteStepImage(fetch, {
				recipe_id: data.recipe.recipe.id,
				step_id: stepID
			});
			saveState = 'saved';
			steps = steps.map((s) => (s.id !== stepID ? s : { ...s, image_url: undefined }));
		} catch (e) {
			saveState = 'failed';
			if (e instanceof HTTPError) {
				console.error('failed to delete image', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete image. try again later.');
		}
	};

	const onRecipeImageUpload = async (image: File) => {
		try {
			saveState = 'saving';
			const res = await uploadRecipeImage(fetch, {
				recipe_id: data.recipe.recipe.id,
				image
			});
			saveState = 'saved';
			recipeImageUrl = res.image_url;
		} catch (e) {
			saveState = 'failed';
			if (e instanceof HTTPError) {
				console.error('failed to upload image', e.message);
			} else {
				console.error(e);
			}
			alert('failed to upload image. try again later.');
		}
	};

	const onRecipeImageDeletion = async () => {
		try {
			saveState = 'saving';
			await deleteRecipeImage(fetch, {
				recipe_id: data.recipe.recipe.id
			});
			saveState = 'saved';
			recipeImageUrl = undefined;
		} catch (e) {
			saveState = 'failed';
			if (e instanceof HTTPError) {
				console.error('failed to delete image', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete image. try again later.');
		}
	};

	const openRecipeFilePicker = () => {
		recipeFileInput?.click();
	};

	const handleRecipeFileSelect = (e: Event) => {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (file) {
			onRecipeImageUpload(file);
		}
	};

	const togglePublish = async () => {
		try {
			saveState = 'saving';
			await updatePersonalRecipe(fetch, {
				recipe_id: data.recipe.recipe.id,
				published: !published
			});
			saveState = 'saved';
			published = !published;
			toast.success(`Recipe ${published ? '' : 'un'}published successfully.`);
		} catch (e) {
			saveState = 'failed';
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
			saveState = 'saving';
			const newIngredient = await createIngredient(fetch, { recipe_id: data.recipe.recipe.id });
			saveState = 'saved';
			ingredients = [...ingredients, newIngredient];
		} catch (e) {
			saveState = 'failed';
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
			saveState = 'saving';
			const newStep = await createStep(fetch, { recipe_id: data.recipe.recipe.id });
			saveState = 'saved';
			steps = [...steps, newStep];
		} catch (e) {
			saveState = 'failed';
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
			saveState = 'saving';
			await deleteIngredient(fetch, {
				recipe_id: data.recipe.recipe.id,
				ingredient_id: ingredientID
			});
			saveState = 'saved';
			ingredients = ingredients.filter((i) => i.id !== ingredientID);
		} catch (e) {
			saveState = 'failed';
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
			saveState = 'saving';
			await deleteStep(fetch, {
				recipe_id: data.recipe.recipe.id,
				step_id: stepID
			});
			saveState = 'saved';
			const idx = steps.findIndex((s) => s.id === stepID);
			if (idx !== -1) {
				steps = [
					...steps.slice(0, idx),
					...steps.slice(idx + 1).map((s) => ({ ...s, step_number: s.step_number - 1 }))
				];
			}
		} catch (e) {
			saveState = 'failed';
			if (e instanceof HTTPError) {
				console.error('failed to delete step', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete step. try again later.');
		}
	};

	const handleDeleteRecipe = async () => {
		try {
			await deleteRecipe(fetch, { recipe_id: data.recipe.recipe.id });
			toast.success('Recipe has been deleted');
			goto(resolve('/home'));
		} catch (e) {
			if (e instanceof HTTPError) {
				console.error('failed to delete recipe', e.message);
			} else {
				console.error(e);
			}
			alert('failed to delete recipe. try again later.');
		}
	};

	const onlyPositiveNumbers = (e: KeyboardEvent) => {
		const invalid = ['e', 'E', '+', '-'];
		if (invalid.includes(e.key)) e.preventDefault();
	};
</script>

<svelte:head>
	<title>Edit Recipe</title>
</svelte:head>

<div class="relative mt-12 mb-12 flex w-full flex-col items-center px-6">
	<div class="sticky top-0 z-50 flex w-full justify-center bg-white">
		<div class="mt-4 flex w-full max-w-md items-center justify-between pb-4">
			<Status status={saveState} />

			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					<div class="rounded-lg p-1 hover:bg-gray-200">
						<EllipsisVertical />
					</div>
				</DropdownMenu.Trigger>
				<DropdownMenu.Content>
					<DropdownMenu.Item class="p-0"
						><a
							class="w-full px-2 py-1.5"
							href={resolve(`/recipes/${data.recipe.recipe.id}/preview`)}>Preview</a
						></DropdownMenu.Item
					>
					<DropdownMenu.Separator />
					{#if published}
						<DropdownMenu.Item
							onSelect={(e) => {
								e.preventDefault();
								unpublishDialogOpen = true;
							}}>Unpublish</DropdownMenu.Item
						>
					{:else}
						<DropdownMenu.Item
							onSelect={(e) => {
								e.preventDefault();
								publishDialogOpen = true;
							}}>Publish</DropdownMenu.Item
						>
					{/if}
					<DropdownMenu.Item
						class="text-red-500 data-highlighted:bg-red-100 data-highlighted:text-red-500"
						onSelect={(e) => {
							e.preventDefault();
							deleteDialogOpen = true;
						}}>Delete</DropdownMenu.Item
					>
				</DropdownMenu.Content>
			</DropdownMenu.Root>

			<DeleteDialog bind:open={deleteDialogOpen} onConfirmation={handleDeleteRecipe} />
			<PublishDialog bind:open={publishDialogOpen} onConfirmation={togglePublish} />
			<UnpublishDialog bind:open={unpublishDialogOpen} onConfirmation={togglePublish} />
		</div>
	</div>
	<div class="flex w-full max-w-md flex-col gap-8">
		<div class="flex flex-col gap-1">
			<label for="title" class="text-lg">Title</label>
			<Input
				name="title"
				bind:value={title}
				class="font-IowanOldStyleBT"
				oninput={onTitleChange}
				placeholder="Untitled Recipe"
			/>
		</div>

		<div class="flex flex-col gap-1">
			<label for="description" class="text-lg">Description</label>
			<Textarea
				name="description"
				bind:value={description}
				class="font-IowanOldStyleBT"
				oninput={onDescriptionChange}
				placeholder="Write a description."
			/>
		</div>

		<div class="flex flex-col gap-1">
			<label for="recipe-image" class="text-lg">Recipe Cover Image</label>
			{#if recipeImageUrl}
				<ImagePreview
					src={recipeImageUrl}
					alt={title || 'Recipe'}
					onRemove={onRecipeImageDeletion}
				/>
			{:else}
				<Button onclick={openRecipeFilePicker} className="w-fit text-sm font-medium">
					Upload Image
				</Button>
			{/if}
			<input
				type="file"
				accept="image/*"
				bind:this={recipeFileInput}
				onchange={handleRecipeFileSelect}
				class="hidden"
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
					class="w-32"
					bind:value={servings}
					oninput={onServingsChange}
					placeholder="1"
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
							class="w-16"
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
							class="w-16"
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
			<Button
				onclick={handleCreateIngredient}
				className="font-medium text-sm mt-4 from-blue-300 to-blue-200 hover:to-blue-100 hover:from-blue-200"
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
							onInstructionChange={onStepInstructionChange}
							onDelete={handleDeleteStep}
							onImageUpload={onStepImageUpload}
							onImageDeletion={onStepImageDeletion}
						/>
					</div>
				{/each}
			</div>
			<Button
				onclick={handleCreateStep}
				className="font-medium text-sm mt-4 from-blue-300 to-blue-200 hover:to-blue-100 hover:from-blue-200"
				>Add Step</Button
			>
		</div>
	</div>
</div>
