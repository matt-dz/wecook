import { type FetchType } from '$lib/http';
import type { Options } from 'ky';
import { PUBLIC_BACKEND_URL } from '$env/static/public';
import * as z from 'zod';

const RecipeOwner = z.object({
	first_name: z.string(),
	last_name: z.string(),
	id: z.int()
});

export const TimeUnit = z.enum(['minutes', 'hours', 'days']);
export type TimeUnitType = z.infer<typeof TimeUnit>;

export const StepSchema = z.object({
	id: z.int(),
	image_url: z.string().optional(),
	instruction: z.string().optional(),
	step_number: z.int()
});

export type Step = z.infer<typeof StepSchema>;

export const UpdateStepSchema = z.object({
	id: z.int().optional(),
	image_url: z.string().optional(),
	instruction: z.string().optional(),
	recipe_id: z.int().optional(),
	step_number: z.int()
});

export type UpdateStep = z.infer<typeof UpdateStepSchema>;

export const IngredientSchema = z.object({
	id: z.int(),
	image_url: z.string().optional(),
	name: z.string().optional(),
	quantity: z.int().optional(),
	recipe_id: z.int().optional(),
	unit: z.string().optional()
});

export type Ingredient = z.infer<typeof IngredientSchema>;

export const RecipeWithIngredientsAndStepsSchema = z.object({
	cook_time_amount: z.int().optional(),
	cook_time_unit: TimeUnit.optional(),
	prep_time_amount: z.int().optional(),
	prep_time_unit: TimeUnit.optional(),
	title: z.string(),
	published: z.boolean(),
	created_at: z.iso.datetime(),
	updated_at: z.iso.datetime(),
	description: z.string().optional(),
	image_url: z.string().optional(),
	user_id: z.int(),
	servings: z.number().optional(),
	id: z.int(),
	ingredients: z.array(IngredientSchema),
	steps: z.array(StepSchema)
});

export type RecipeWithIngredientsAndSteps = z.infer<typeof RecipeWithIngredientsAndStepsSchema>;

export const RecipeSchema = z.object({
	cook_time_amount: z.int().optional(),
	cook_time_unit: z.enum(['minutes', 'hours', 'days']).optional(),
	prep_time_amount: z.int().optional(),
	prep_time_unit: z.enum(['minutes', 'hours', 'days']).optional(),
	title: z.string(),
	published: z.boolean(),
	created_at: z.iso.datetime(),
	updated_at: z.iso.datetime(),
	description: z.string().optional(),
	image_url: z.string().optional(),
	user_id: z.int(),
	id: z.int(),
	servings: z.number().optional()
});

export type Recipe = z.infer<typeof RecipeSchema>;

export const RecipeWithStepsIngredientsAndOwnerSchema = z.object({
	owner: RecipeOwner,
	recipe: RecipeWithIngredientsAndStepsSchema
});

export type RecipeWithStepsIngredientsAndOwner = z.infer<
	typeof RecipeWithStepsIngredientsAndOwnerSchema
>;

export const RecipeAndOwnerSchema = z.object({
	owner: RecipeOwner,
	recipe: RecipeSchema
});

export type RecipeAndOwner = z.infer<typeof RecipeAndOwnerSchema>;

export const GetPersonalRecipesResponse = z.object({
	recipes: z.array(RecipeAndOwnerSchema)
});

export async function GetPersonalRecipes(
	fetch: FetchType,
	options?: Options
): Promise<z.infer<typeof GetPersonalRecipesResponse>> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes`, options);
	return GetPersonalRecipesResponse.parse(await res.json());
}

export const GetRecipesResponse = z.object({
	recipes: z.array(RecipeAndOwnerSchema)
});

export async function GetRecipes(
	fetch: FetchType,
	options?: Options
): Promise<z.infer<typeof GetRecipesResponse>> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes/public`, options);
	return GetRecipesResponse.parse(await res.json());
}

export type GetRecipeResponse = RecipeWithStepsIngredientsAndOwner;

export async function GetRecipe(
	fetch: FetchType,
	id: number,
	options?: Options
): Promise<GetRecipeResponse> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes/${id}`, options).json();
	return RecipeWithStepsIngredientsAndOwnerSchema.parse(res);
}

export const CreateRecipeResponseSchema = z.object({
	recipe_id: z.int()
});

export type CreateRecipeResponse = z.infer<typeof CreateRecipeResponseSchema>;

export async function CreateRecipe(
	fetch: FetchType,
	options?: Options
): Promise<CreateRecipeResponse> {
	const res = await fetch.post(`${PUBLIC_BACKEND_URL}/api/recipes`, options).json();
	return CreateRecipeResponseSchema.parse(res);
}

export type GetPersonalRecipeResponse = z.infer<typeof RecipeWithStepsIngredientsAndOwnerSchema>;

export async function GetPersonalRecipe(
	fetch: FetchType,
	id: number,
	options?: Options
): Promise<GetPersonalRecipeResponse> {
	const res = await fetch.get(`${PUBLIC_BACKEND_URL}/api/recipes/${id}`, options).json();
	return RecipeWithStepsIngredientsAndOwnerSchema.parse(res);
}

export type UpdateRecipeRequest = {
	recipe_id: number;
	title?: string;
	description?: string;
	published?: boolean;
	servings?: number;
	cook_time_amount?: number;
	cook_time_unit?: TimeUnitType;
	prep_time_amount?: number;
	prep_time_unit?: TimeUnitType;
};

export type UpdateRecipeResponse = Recipe;

export async function updatePersonalRecipe(
	fetch: FetchType,
	request: UpdateRecipeRequest,
	options?: Options
): Promise<UpdateRecipeResponse> {
	const res = await fetch
		.patch(`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}`, {
			...options,
			json: request
		})
		.json();
	return RecipeSchema.parse(res);
}

export type UpdateIngredientRequest = {
	recipe_id: number;
	ingredient_id: number;
	quantity?: number;
	name?: string;
	unit?: string;
};

export type UpdateIngredientResponse = Ingredient;

export async function updateIngredient(
	fetch: FetchType,
	request: UpdateIngredientRequest,
	options?: Options
): Promise<UpdateIngredientResponse> {
	const res = await fetch
		.patch(
			`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/ingredients/${request.ingredient_id}`,
			{
				...options,
				json: {
					quantity: request.quantity,
					name: request.name,
					unit: request.unit
				}
			}
		)
		.json();
	return IngredientSchema.parse(res);
}

export type CreateIngredientRequest = {
	recipe_id: number;
};
export type CreateIngredientResponse = Ingredient;

export async function createIngredient(
	fetch: FetchType,
	request: CreateIngredientRequest,
	options?: Options
): Promise<Ingredient> {
	const res = await fetch
		.post(`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/ingredients`, options)
		.json();
	return IngredientSchema.parse(res);
}

export type UpdateStepRequest = {
	recipe_id: number;
	step_id: number;
	step_number?: number;
	instruction?: string;
};

export type UpdateStepResponse = Step;

export async function updateStep(
	fetch: FetchType,
	request: UpdateStepRequest,
	options?: Options
): Promise<Step> {
	const res = await fetch
		.patch(`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/steps/${request.step_id}`, {
			...options,
			json: {
				step_number: request.step_number,
				instruction: request.instruction
			}
		})
		.json();
	return StepSchema.parse(res);
}

export type CreateStepRequest = {
	recipe_id: number;
};

export type CreateStepResponse = Step;

export async function createStep(
	fetch: FetchType,
	request: CreateStepRequest,
	options?: Options
): Promise<Step> {
	const res = await fetch
		.post(`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/steps`, {
			...options
		})
		.json();
	return StepSchema.parse(res);
}

export type DeleteIngredientRequest = {
	recipe_id: number;
	ingredient_id: number;
};

export async function deleteIngredient(
	fetch: FetchType,
	request: DeleteIngredientRequest,
	options?: Options
): Promise<void> {
	await fetch.delete(
		`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/ingredients/${request.ingredient_id}`,
		options
	);
}

export type DeleteStepRequest = {
	recipe_id: number;
	step_id: number;
};

export async function deleteStep(
	fetch: FetchType,
	request: DeleteStepRequest,
	options?: Options
): Promise<void> {
	await fetch.delete(
		`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/steps/${request.step_id}`,
		options
	);
}

export type UploadIngredientImageRequest = {
	recipe_id: number;
	ingredient_id: number;
	image: File;
};

export type UploadIngredientImageResponse = Ingredient;

export async function uploadIngredientImage(
	fetch: FetchType,
	request: UploadIngredientImageRequest,
	options?: Options
): Promise<UploadIngredientImageResponse> {
	const form = new FormData();
	form.append('image', request.image);
	const res = await fetch
		.post(
			`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/ingredients/${request.ingredient_id}/image`,
			{
				...options,
				body: form
			}
		)
		.json();
	return IngredientSchema.parse(res);
}

export type DeleteIngredientImageRequest = {
	recipe_id: number;
	ingredient_id: number;
};

export async function deleteIngredientImage(
	fetch: FetchType,
	request: DeleteIngredientImageRequest,
	options?: Options
): Promise<void> {
	await fetch.delete(
		`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/ingredients/${request.ingredient_id}/image`,
		options
	);
}

export type UploadStepImageRequest = {
	recipe_id: number;
	step_id: number;
	image: File;
};

export type UploadStepImageResponse = Step;

export async function uploadStepImage(
	fetch: FetchType,
	request: UploadStepImageRequest,
	options?: Options
): Promise<UploadStepImageResponse> {
	const form = new FormData();
	form.append('image', request.image);
	const res = await fetch
		.post(`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/steps/${request.step_id}/image`, {
			...options,
			body: form
		})
		.json();
	return StepSchema.parse(res);
}

export type DeleteStepImageRequest = {
	recipe_id: number;
	step_id: number;
};

export async function deleteStepImage(
	fetch: FetchType,
	request: DeleteStepImageRequest,
	options?: Options
): Promise<void> {
	await fetch.delete(
		`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/steps/${request.step_id}/image`,
		options
	);
}

export type UploadRecipeImageRequest = {
	recipe_id: number;
	image: File;
};

export type UploadRecipeImageResponse = Recipe;

export async function uploadRecipeImage(
	fetch: FetchType,
	request: UploadRecipeImageRequest,
	options?: Options
): Promise<UploadRecipeImageResponse> {
	const form = new FormData();
	form.append('image', request.image);
	const res = await fetch
		.post(`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/image`, {
			...options,
			body: form
		})
		.json();
	return RecipeSchema.parse(res);
}

export type DeleteRecipeImageRequest = {
	recipe_id: number;
};

export async function deleteRecipeImage(
	fetch: FetchType,
	request: DeleteRecipeImageRequest,
	options?: Options
): Promise<void> {
	await fetch.delete(`${PUBLIC_BACKEND_URL}/api/recipes/${request.recipe_id}/image`, options);
}
