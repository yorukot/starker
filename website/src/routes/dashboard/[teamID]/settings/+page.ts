import type { PageLoad } from './$types';
import type { Team } from '$lib/schemas/team';

export const load: PageLoad = async ({ params, parent }) => {
  const { currentTeam, user } = await parent();

  // Ensure we have team data
  if (!currentTeam) {
    throw new Error('Team not found');
  }

  // Check if current user is team owner
  const isOwner = user && currentTeam.owner_id === user.id;

  return {
    team: currentTeam as Team,
    isOwner: isOwner || false,
    user
  };
};