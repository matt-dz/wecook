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

export const RecipeWithoutStepsAndIngredients = z.object({
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
	id: z.int()
});

export type RecipeWithoutStepsAndIngredients = z.infer<typeof RecipeWithoutStepsAndIngredients>;

export const StepSchema = z.object({
	id: z.int(),
	image_url: z.string().optional(),
	instruction: z.string().optional(),
	recipe_id: z.int(),
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

export const UpdateIngredientSchema = z.object({
	id: z.int().optional(),
	image_url: z.string().optional(),
	name: z.string().optional(),
	quantity: z.int().optional(),
	recipe_id: z.int(),
	unit: z.string().optional()
});

export type UpdateIngredient = z.infer<typeof UpdateIngredientSchema>;

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
	servings: z.number().optional(),
	ingredients: z.array(IngredientSchema),
	steps: z.array(StepSchema)
});

export type Recipe = z.infer<typeof RecipeSchema>;

export const RecipeAndOwner = z.object({
	owner: RecipeOwner,
	recipe: RecipeSchema
});

export type RecipeAndOwnerType = z.infer<typeof RecipeAndOwner>;

export const RecipeAndOwnerWithoutStepsAndIngredients = z.object({
	owner: RecipeOwner,
	recipe: RecipeWithoutStepsAndIngredients
});

export type RecipeAndOwnerWithoutStepsAndIngredientsType = z.infer<
	typeof RecipeAndOwnerWithoutStepsAndIngredients
>;

export const GetPersonalRecipesResponse = z.object({
	recipes: z.array(RecipeAndOwnerWithoutStepsAndIngredients)
});

export async function GetPersonalRecipes(
	fetch: FetchType,
	options?: Options
): Promise<z.infer<typeof GetPersonalRecipesResponse>> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes`, options);
	return GetPersonalRecipesResponse.parse(await res.json());
}

export const GetRecipesResponse = z.object({
	recipes: z.array(RecipeAndOwnerWithoutStepsAndIngredients)
});

export async function GetRecipes(
	fetch: FetchType,
	options?: Options
): Promise<z.infer<typeof GetRecipesResponse>> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes/public`, options);
	return GetRecipesResponse.parse(await res.json());
}

export async function GetRecipe(
	fetch: FetchType,
	id: number,
	options?: Options
): Promise<RecipeAndOwnerType> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes/${id}`, options).json();
	return RecipeAndOwner.parse(res);
}

export const CreateRecipeResponse = z.object({
	recipe_id: z.int()
});

export type CreateRecipeResponseType = z.infer<typeof CreateRecipeResponse>;

export async function CreateRecipe(
	fetch: FetchType,
	options?: Options
): Promise<CreateRecipeResponseType> {
	const res = await fetch.post(`${PUBLIC_BACKEND_URL}/api/recipes`, options).json();
	return CreateRecipeResponse.parse(res);
}

export const GetPersonalRecipeResponse = RecipeAndOwner;
export type GetPersonalRecipeResponseType = z.infer<typeof GetPersonalRecipeResponse>;

export async function GetPersonalRecipe(
	fetch: FetchType,
	id: number,
	options?: Options
): Promise<GetPersonalRecipeResponseType> {
	const res = await fetch.get(`${PUBLIC_BACKEND_URL}/api/recipes/${id}`, options).json();
	return GetPersonalRecipeResponse.parse(res);
}
