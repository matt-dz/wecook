import { redirect } from '@sveltejs/kit';
import type { Actions } from './$types';
import { ACCESS_TOKEN_COOKIE_NAME, REFRESH_TOKEN_COOKIE_NAME } from '$lib/auth';

export const actions: Actions = {
	default: async ({ cookies }) => {
		// Delete the authentication cookies
		cookies.delete(ACCESS_TOKEN_COOKIE_NAME, { path: '/' });
		cookies.delete(REFRESH_TOKEN_COOKIE_NAME, { path: '/' });

		// Redirect to home page
		redirect(303, '/');
	}
};
