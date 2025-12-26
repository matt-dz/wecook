import type { PageServerLoad } from './$types';
import { GetRecipe } from '$lib/recipes';
import { error } from '@sveltejs/kit';
import fetch from '$lib/http';
import * as z from 'zod';
import { HTTPError } from 'ky';
import { ApiErrorCodes, parseError } from '$lib/errors/api';

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

	try {
		return {
			recipe: await GetRecipe(fetch, recipeID)
		};
	} catch (e) {
		if (e instanceof HTTPError) {
			const err = await parseError(e.response);
			if (err.success && err.data.code === ApiErrorCodes.RecipeNotFound) {
				return error(404, {
					message: 'Recipe not found.',
					status: err.data.status,
					errorId: err.data.error_id,
					code: err.data.code
				});
			}
		}
		throw e;
	}
};
