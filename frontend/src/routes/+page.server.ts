import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { getRecipes } from '$lib/recipes';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async () => {
	try {
		const recipes = await getRecipes(fetch, {}, env.INTERNAL_BACKEND_URL);
		return {
			recipes
		};
	} catch (e) {
		console.error(e);
		throw e;
	}
};
