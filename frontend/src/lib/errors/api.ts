import type { KyResponse } from 'ky';
import * as z from 'zod';

export const ApiError = z.object({
	code: z.string(),
	error_id: z.string().optional(),
	message: z.string(),
	status: z.number()
});

enum ApiErrorCodes {
	InternalServerError = 'internal_server_error',
	BadRequest = 'bad_request',
	UnprocessibleEntity = 'unprocessible_entity',
	InvalidAccessToken = 'invalid_access_token',
	ExpiredAccessToken = 'expired_access_token',
	InvalidRefreshToken = 'invalid_refresh_token',
	ExpiredRefreshToken = 'expired_refresh_token',
	InsufficientPermissions = 'insufficient_permissions',
	WeakPassword = 'weak_password',
	EmailConflict = 'email_conflict',
	AdminAlreadySetup = 'admin_already_setup',
	RecipeNotFound = 'recipe_not_found',
	RecipeNotOwned = 'recipe_not_owned',
	IngredientNotFound = 'ingredient_not_found',
	StepNotFound = 'step_not_found',
	InvalidCredentials = 'invalid_credentials'
}

export class RefreshTokenExpiredError extends Error {
	name = 'RefreshTokenExpiredError';

	constructor(message = 'Refresh token has expired') {
		super(message);
		Object.setPrototypeOf(this, new.target.prototype);
	}
}

export { ApiErrorCodes };

export async function parseError(response: KyResponse): Promise<z.infer<typeof ApiError>> {
	const clone = response.clone();
	return ApiError.parse(await clone.json());
}

export async function accessTokenExpired(response: KyResponse) {
	try {
		const clone = response.clone();
		const res = ApiError.safeParse(await clone.json());
		if (!res.success) {
			return false;
		}
		if (response.status !== 401) {
			return false;
		}

		return (
			ApiErrorCodes.ExpiredAccessToken === res.data.code ||
			ApiErrorCodes.InvalidAccessToken === res.data.code ||
			ApiErrorCodes.InvalidCredentials === res.data.code
		);
	} catch {
		return false;
	}
}

export async function refreshTokenExpired(response: KyResponse) {
	try {
		const clone = response.clone();
		const res = ApiError.safeParse(await clone.json());
		if (!res.success) {
			return false;
		}
		return (
			(res.data.code === ApiErrorCodes.ExpiredRefreshToken && clone.status === 401) ||
			(res.data.code === ApiErrorCodes.InvalidRefreshToken && clone.status === 401)
		);
	} catch {
		return false;
	}
}
