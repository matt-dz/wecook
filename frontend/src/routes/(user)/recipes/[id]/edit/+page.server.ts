import type { PageServerLoad } from './$types';
import fetch from '$lib/http';
import { redirect } from '@sveltejs/kit';
import { HTTPError } from 'ky';
import { refreshTokenExpired } from '$lib/errors/api';
import { GetPersonalRecipe } from '$lib/recipes';
import { error } from '@sveltejs/kit';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import * as z from 'zod';

export const load: PageServerLoad = async ({ cookies, params }) => {
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

	try {
		const recipe = await GetPersonalRecipe(fetch, recipeID, {
			headers: {
				Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}`
			}
		});
		return {
			recipe
		};
	} catch (e) {
		if (e instanceof HTTPError) {
			if (await refreshTokenExpired(e.response)) {
				redirect(303, '/login');
			}
			console.error(e.message);
			return;
		}
		// TODO: handle unexpected errors
		console.error(e);
	}
};
