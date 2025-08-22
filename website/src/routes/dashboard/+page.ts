export const ssr = false;

import { goto } from '$app/navigation';
import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { Team } from '$lib/schemas/team';

export const load = async () => {
	try {
		const response = await authGet(`${PUBLIC_API_BASE_URL}/teams`);
		if (!response.ok) {
			goto('/dashboard/intro/new-team');
			return {};
		}

		const teams: Team[] = await response.json();

		if (teams.length === 0) {
			goto('/dashboard/intro/new-team');
		} else {
			goto(`/dashboard/${teams[0].id}/projects`);
		}
	} catch (error) {
		console.error('Error fetching teams:', error);
		goto('/dashboard/intro/new-team');
	}

	return {};
};
