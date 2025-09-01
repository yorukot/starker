export const ssr = false;

import { fetchServiceContainers } from '$lib/api/client';
import type { PageLoad } from './$types';
import type { ServiceContainer } from '$lib/schemas/service';

export const load: PageLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	const projectID = params.projectID;
	const serviceID = params.serviceID;
	depends(`project:${projectID}`);

	try {
		// Fetch service containers
		const containersResponse = await fetchServiceContainers(teamID!, projectID!, serviceID!);

		if (!containersResponse.ok) {
			return {
				containers: [],
				error: `Failed to fetch containers: ${containersResponse.status}`
			};
		}

		const containersData = await containersResponse.json();
		// Handle both wrapped (containersData.data) and direct array responses
		const containers: ServiceContainer[] = Array.isArray(containersData)
			? containersData
			: containersData?.data || [];

		return {
			containers
		};
	} catch (error: unknown) {
		console.error('Error fetching containers:', error);

		return {
			containers: [],
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};