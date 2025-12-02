import type { PageServerLoad } from './$types';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import fetch from '$lib/http';
import { GetPersonalRecipes } from '$lib/recipes';
import { redirect } from '@sveltejs/kit';
import { HTTPError } from 'ky';
import { refreshTokenExpired } from '$lib/errors/api';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	if (!accessToken) {
		console.log('no access token available, sending user back to login.');
		redirect(307, '/login');
	}
	try {
		const recipes = await GetPersonalRecipes(fetch, {
			headers: {
				Cookie: `${ACCESS_TOKEN_COOKIE_NAME}=${accessToken}`
			}
		});
		console.log(recipes);
		return {
			recipes
		};
	} catch (e) {
		if (e instanceof HTTPError) {
			if (await refreshTokenExpired(e.response)) {
				redirect(303, '/login');
			}
			console.error(e.message);
		} else {
			console.log(e);
		}
		// TODO: handle unexpected errors
		console.error(e);
	}
};
