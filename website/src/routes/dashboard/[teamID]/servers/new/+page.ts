export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { PrivateKey } from '$lib/schemas/server';

export const load: PageLoad = async ({ params }) => {
	const teamID = params.teamID;

	try {
		const response = await authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/private-keys`);

		if (!response.ok) {
			return {
				privateKeys: [],
				error: `Failed to fetch SSH keys: ${response.status}`
			};
		}

		const privateKeys: PrivateKey[] = await response.json();

		return {
			privateKeys
		};
	} catch (error: unknown) {
		console.error('Error fetching private keys:', error);

		return {
			privateKeys: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};