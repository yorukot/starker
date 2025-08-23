export const ssr = false;

import { authGet } from '$lib/api/client';
import { PUBLIC_API_BASE_URL } from '$env/static/public';
import type { LayoutLoad } from './$types';
import type { Team } from '$lib/schemas/team';
import type { User } from '$lib/schemas/user';

export const load: LayoutLoad = async ({ params, depends }) => {
	const teamID = params.teamID;
	depends('team:current');
	depends('user:current');

	try {
		// Fetch teams and user data in parallel
		const [teamsResponse, userResponse] = await Promise.all([
			authGet(`${PUBLIC_API_BASE_URL}/teams`),
			authGet(`${PUBLIC_API_BASE_URL}/users/me`)
		]);

		// Handle teams data
		let teams: Team[] = [];
		let currentTeam: Team | null = null;
		if (teamsResponse.ok) {
			teams = await teamsResponse.json();
			currentTeam = teams.find((team) => team.id === teamID) || null;
		}

		// Handle user data
		let user: User | null = null;
		if (userResponse.ok) {
			user = await userResponse.json();
		}

		return {
			teams,
			currentTeam,
			user,
			error: !teamsResponse.ok
				? `Failed to fetch teams: ${teamsResponse.status}`
				: !userResponse.ok
					? `Failed to fetch user: ${userResponse.status}`
					: undefined
		};
	} catch (error: unknown) {
		console.error('Error fetching data:', error);

		return {
			teams: [],
			currentTeam: null,
			user: null,
			error: error instanceof Error ? error.message : 'Unknown error occurred'
		};
	}
};
