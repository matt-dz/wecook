import { type FetchType } from '$lib/http';
import type { Options } from 'ky';
import { PUBLIC_BACKEND_URL } from '$env/static/public';
import * as z from 'zod';

const RecipeOwner = z.object({
	first_name: z.string(),
	last_name: z.string(),
	id: z.int()
});

export const RecipeWithoutStepsAndIngredients = z.object({
	cook_time_minutes: z.int(),
	title: z.string(),
	published: z.boolean(),
	created_at: z.iso.datetime(),
	updated_at: z.iso.datetime(),
	description: z.string().optional(),
	image_url: z.string().optional(),
	user_id: z.int(),
	id: z.int()
});

export type RecipeWithoutStepsAndIngredients = z.infer<typeof RecipeWithoutStepsAndIngredients>;

export const Step = z.object({
	created_at: z.iso.datetime(),
	id: z.int(),
	image_url: z.string().optional(),
	instruction: z.string(),
	recipe_id: z.int(),
	step_number: z.int(),
	updated_at: z.iso.datetime()
});

export const Ingredient = z.object({
	id: z.int(),
	image_url: z.string().optional(),
	name: z.string(),
	quantity: z.int(),
	recipe_id: z.int(),
	unit: z.string().optional()
});

export const Recipe = z.object({
	cook_time_minutes: z.int(),
	title: z.string(),
	published: z.boolean(),
	created_at: z.iso.datetime(),
	updated_at: z.iso.datetime(),
	description: z.string().optional(),
	image_url: z.string().optional(),
	user_id: z.int(),
	id: z.int(),
	ingredients: z.array(Ingredient),
	steps: z.array(Step)
});

export type Recipe = z.infer<typeof Recipe>;

export const RecipeAndOwner = z.object({
	owner: RecipeOwner,
	recipe: Recipe
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
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes/personal`, options);
	return GetPersonalRecipesResponse.parse(await res.json());
}

export async function GetRecipe(
	fetch: FetchType,
	id: number,
	options?: Options
): Promise<RecipeAndOwnerType> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes/${id}`, options).json();
	return RecipeAndOwner.parse(res);
}
