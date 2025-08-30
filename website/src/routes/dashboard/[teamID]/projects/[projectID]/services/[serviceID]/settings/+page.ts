export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { PageLoad } from './$types';
import type { Service } from '$lib/schemas/service';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	const serviceID = params.serviceID;
	depends(`service:${serviceID}:settings`);

	try {
		// Fetch service details
		const serviceResponse = await authGet(
			`${PUBLIC_API_BASE_URL}/teams/${teamID}/projects/${projectID}/services/${serviceID}`
		);

		if (!serviceResponse.ok) {
			return {
				service: null,
				error: `Failed to fetch service details: ${serviceResponse.status}`
			};
		}

		const service: Service = await serviceResponse.json();

		return {
			service
		};
	} catch (error: unknown) {
		console.error('Error fetching service settings data:', error);

		return {
			service: null,
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
