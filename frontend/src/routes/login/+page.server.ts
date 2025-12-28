import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/auth';
import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';

export const load: PageServerLoad = ({ cookies }) => {
	// if access token exists, the user has already been authorized
	// from the hooks, thus, they should be redirect to home page.
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	if (accessToken) {
		redirect(303, '/');
	}
};
