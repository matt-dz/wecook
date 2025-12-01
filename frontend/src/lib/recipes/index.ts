import { type FetchType } from '$lib/http';
import type { Options } from 'ky';
import { PUBLIC_BACKEND_URL } from '$env/static/public';
import * as z from 'zod';

const RecipeOwner = z.object({
	first_name: z.string(),
	last_name: z.string(),
	id: z.int()
});

const Recipe = z.object({
	cook_time_minutes: z.int(),
	title: z.string(),
	published: z.boolean(),
	created_at: z.iso.datetime(),
	updated_at: z.iso.datetime(),
	description: z.string().optional(),
	image_url: z.string().optional(),
	user_id: z.int()
});

const RecipeAndOwner = z.object({
	owner: RecipeOwner,
	recipe: Recipe
});

const GetPersonalRecipesResponse = z.object({
	recipes: z.array(RecipeAndOwner)
});

export async function GetPersonalRecipes(
	fetch: FetchType,
	options?: Options
): Promise<z.infer<typeof GetPersonalRecipesResponse>> {
	const res = await fetch(`${PUBLIC_BACKEND_URL}/api/recipes/personal`, options);
	return GetPersonalRecipesResponse.parse(await res.json());
}
