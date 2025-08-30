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

// Service Environment Variable type (matches backend ServiceEnvironment model)
export interface ServiceEnvironment {
	id: number;
	service_id: string;
	key: string;
	value: string;
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

// Service Container type (matches backend ServiceContainer model)
export interface ServiceContainer {
	id: string;
	service_id: string;
	container_id?: string; // Docker container ID (set when container is running)
	container_name: string;
	state: string; // running, stopped, removed, exited
	created_at: string;
	updated_at: string;
}
