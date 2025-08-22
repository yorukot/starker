// Team type
export interface Team {
	id: string;
	owner_id: string;
	name: string;
	updated_at: string;
	created_at: string;
}

// Team User type
export interface TeamUser {
	id: string;
	team_id: string;
	user_id: string;
	updated_at: string;
	created_at: string;
}

// Team Invite type
export interface TeamInvite {
	id: string;
	team_id: string;
	invited_by: string;
	invited_to: string;
	updated_at: string;
	created_at: string;
}
