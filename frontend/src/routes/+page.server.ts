import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { GetRecipes } from '$lib/recipes';

export const load: PageServerLoad = async () => {
	return {
		recipes: await GetRecipes(fetch)
	};
};
