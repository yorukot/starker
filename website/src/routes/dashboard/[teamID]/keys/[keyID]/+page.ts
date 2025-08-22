export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { PrivateKey } from '$lib/schemas/server';

export const load: PageLoad = async ({ params, parent, depends }) => {
	const teamID = params.teamID;
	const keyID = params.keyID;
	depends(`key:${keyID}`);

	// Get parent layout data
	const parentData = await parent();

	try {
		const response = await authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/private-keys/${keyID}`);

		if (!response.ok) {
			return {
				...parentData,
				privateKey: null,
				error: `Failed to fetch key: ${response.status}`
			};
		}

		const privateKey: PrivateKey = await response.json();

		return {
			...parentData,
			privateKey
		};
	} catch (error: unknown) {
		console.error('Error fetching private key:', error);

		return {
			...parentData,
			privateKey: null,
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
