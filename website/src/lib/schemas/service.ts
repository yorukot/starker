// Service State enum
export enum ServiceState {
	RUNNING = 'running',
	STOPPED = 'stopped',
	STARTING = 'starting',
	STOPPING = 'stopping',
	RESTARTING = 'restarting'
}

// Service type
export interface Service {
	id: string;
	team_id: string;
	server_id: string;
	project_id: string;
	name: string;
	description?: string;
	type: string;
	state: ServiceState;
	container_id?: string;
	last_deployed_at?: string;
	created_at: string;
	updated_at: string;
}

// Service Environment Variable type
export interface ServiceEnvironmentVariable {
	id: string;
	service_id: string;
	key: string;
	value: string;
	is_secret: boolean;
	created_at: string;
	updated_at: string;
}

// Service Compose Config type
export interface ServiceComposeConfig {
	id: string;
	service_id: string;
	compose_file: string;
	compose_file_path?: string;
	created_at: string;
	updated_at: string;
}
