import type { PageServerLoad } from './$types';
import { GetRecipe } from '$lib/recipes';
import { error } from '@sveltejs/kit';
import fetch from '$lib/http';
import * as z from 'zod';

export const load: PageServerLoad = async ({ params }) => {
	const res = z.string().regex(/^\d+$/).safeParse(params.id);
	if (!res.success) {
		error(400, {
			message: 'Invalid Recipe ID'
		});
	}
	const recipeID = parseInt(res.data);
	if (recipeID == Infinity) {
		error(400, {
			message: 'Invalid Recipe ID'
		});
	}

	return {
		recipe: await GetRecipe(fetch, recipeID)
	};
};
