import type { KyResponse } from 'ky';
import * as z from 'zod';

export const ApiError = z.object({
	code: z.string(),
	error_id: z.string().optional(),
	message: z.string(),
	status: z.number()
});

enum ApiErrorCodes {
	MissingCredentials = 'missing_credentials',
	InternalServerError = 'internal_server_error',
	BadRequest = 'bad_request',
	UnprocessibleEntity = 'unprocessible_entity',
	InvalidToken = 'invalid_token',
	ExpiredToken = 'expired_token',
	InvalidCredentials = 'invalid_credentials',
	InsufficientPermissions = 'insufficient_permissions',
	WeakPassword = 'weak_password',
	EmailConflict = 'email_conflict',
	AdminAlreadySetup = 'admin_already_setup',
	RecipeNotFound = 'recipe_not_found',
	RecipeNotOwned = 'recipe_not_owned',
	IngredientNotFound = 'ingredient_not_found',
	StepNotFound = 'step_not_found'
}

export { ApiErrorCodes };

export async function parseError(response: KyResponse): Promise<z.infer<typeof ApiError>> {
	const clone = response.clone();
	return ApiError.parse(await clone.json());
}
