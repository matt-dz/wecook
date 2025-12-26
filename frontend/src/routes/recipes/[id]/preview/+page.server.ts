import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { redirect } from '@sveltejs/kit';
import { GetPersonalRecipe } from '$lib/recipes';
import { error } from '@sveltejs/kit';
import { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';
import * as z from 'zod';

export const load: PageServerLoad = async ({ cookies, params }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	if (!accessToken) {
		redirect(303, '/login');
	}
	const refreshToken = cookies.get(REFRESH_TOKEN_COOKIE_NAME);

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

	const recipe = await GetPersonalRecipe(fetch, recipeID, {
		headers: {
			Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}; ${REFRESH_TOKEN_COOKIE_NAME}=${refreshToken}`
		}
	});
	return {
		recipe
	};
};
