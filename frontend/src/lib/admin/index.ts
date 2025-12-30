import type { FetchType } from '$lib/http';
import type { Options } from 'ky';
import * as z from 'zod';

export const PreferencesSchema = z.object({
	allow_public_signup: z.boolean()
});

export type Preferences = z.infer<typeof PreferencesSchema>;

export async function getPreferences(
	fetch: FetchType,
	options?: Options,
	apiUrl?: string
): Promise<Preferences> {
	const json = await fetch.get(`${apiUrl ?? ''}/api/preferences`, options).json();
	return PreferencesSchema.parse(json);
}

export const UpdatePreferencesRequestSchema = z.object({
	allow_public_signups: z.boolean().optional()
});

export type UpdateReferencesRequest = z.infer<typeof UpdatePreferencesRequestSchema>;

export async function updatePreferences(
	fetch: FetchType,
	request: UpdateReferencesRequest,
	options?: Options,
	apiUrl?: string
): Promise<Preferences> {
	const json = await fetch
		.patch(`${apiUrl ?? ''}/api/preferences`, {
			...options,
			json: request
		})
		.json();
	return PreferencesSchema.parse(json);
}
