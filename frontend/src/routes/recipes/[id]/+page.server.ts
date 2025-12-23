import type { PageServerLoad } from './$types';
import { GetRecipe } from '$lib/recipes';
import { error } from '@sveltejs/kit';
import fetch from '$lib/http';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import { redirect } from '@sveltejs/kit';
import * as z from 'zod';

export const load: PageServerLoad = async ({ params, cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	if (!accessToken) {
		console.log('no access token available, sending user back to login.');
		redirect(307, '/login');
	}

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
		recipe: await GetRecipe(fetch, recipeID, {
			headers: {
				Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}`
			}
		})
	};
};
