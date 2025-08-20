// Server type
export interface Server {
	id: string;
	team_id: string;
	name: string;
	description?: string;
	ip: string;
	port: string;
	user: string;
	private_key_id: string;
	updated_at: string;
	created_at: string;
}

// Private Key type
export interface PrivateKey {
	id: string;
	team_id: string;
	name: string;
	description?: string;
	private_key: string;
	fingerprint?: string;
	created_at: string;
	updated_at: string;
}