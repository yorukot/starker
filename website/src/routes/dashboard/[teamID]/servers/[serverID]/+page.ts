export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Server, PrivateKey } from '$lib/schemas/server';

export const load: PageLoad = async ({ params, parent, depends }) => {
	const teamID = params.teamID;
	const serverID = params.serverID;
	depends(`server:${serverID}`);

	// Get parent layout data
	const parentData = await parent();

	try {
		// Fetch server and private keys in parallel
		const [serverResponse, keysResponse] = await Promise.all([
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/servers/${serverID}`),
			authGet(`${PUBLIC_API_BASE_URL}/teams/${teamID}/private-keys`)
		]);

		if (!serverResponse.ok) {
			return {
				...parentData,
				server: null,
				privateKeys: [],
				error: `Failed to fetch server: ${serverResponse.status}`
			};
		}

		const server: Server = await serverResponse.json();
		let privateKeys: PrivateKey[] = [];

		if (keysResponse.ok) {
			privateKeys = await keysResponse.json();
		}

		return {
			...parentData,
			server,
			privateKeys
		};
	} catch (error: unknown) {
		console.error('Error fetching server:', error);

		return {
			...parentData,
			server: null,
			privateKeys: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};