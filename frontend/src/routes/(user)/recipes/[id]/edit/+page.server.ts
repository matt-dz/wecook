import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { redirect } from '@sveltejs/kit';
import { getPersonalRecipe } from '$lib/recipes';
import { error } from '@sveltejs/kit';
import { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';
import { env } from '$env/dynamic/private';
import * as z from 'zod';

export const load: PageServerLoad = async ({ cookies, params }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	if (!accessToken) {
		console.log('no access token available, sending user back to login.');
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

	try {
		const recipe = await getPersonalRecipe(
			fetch,
			recipeID,
			{
				headers: {
					Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}; ${REFRESH_TOKEN_COOKIE_NAME}=${refreshToken}`
				}
			},
			env.INTERNAL_BACKEND_URL
		);
		return {
			recipe
		};
	} catch (e) {
		console.error(e);
		throw e;
	}
};
