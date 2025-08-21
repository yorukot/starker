import * as yup from 'yup';

export const createServerSchema = yup.object({
	name: yup
		.string()
		.required('Name is required')
		.min(1, 'Name must be at least 1 character')
		.max(100, 'Name must be less than 100 characters'),
	description: yup
		.string()
		.max(500, 'Description must be less than 500 characters'),
	ip: yup
		.string()
		.required('IP address is required')
		.test('valid-ip', 'Please enter a valid IP address or hostname', (value) => {
			if (!value) return false;
			// Basic validation for IP or hostname
			const ipRegex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
			const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
			return ipRegex.test(value) || hostnameRegex.test(value);
		}),
	port: yup
		.string()
		.required('Port is required')
		.test('valid-port', 'Port must be between 1 and 65535', (value) => {
			if (!value) return false;
			const port = parseInt(value, 10);
			return !isNaN(port) && port >= 1 && port <= 65535;
		}),
	user: yup
		.string()
		.required('Username is required')
		.min(1, 'Username must be at least 1 character')
		.max(50, 'Username must be less than 50 characters'),
	private_key_id: yup
		.string()
		.required('SSH key is required')
});

export const updateServerSchema = yup.object({
	name: yup
		.string()
		.required('Name is required')
		.min(1, 'Name must be at least 1 character')
		.max(100, 'Name must be less than 100 characters'),
	description: yup
		.string()
		.max(500, 'Description must be less than 500 characters'),
	ip: yup
		.string()
		.required('IP address is required')
		.test('valid-ip', 'Please enter a valid IP address or hostname', (value) => {
			if (!value) return false;
			// Basic validation for IP or hostname
			const ipRegex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
			const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
			return ipRegex.test(value) || hostnameRegex.test(value);
		}),
	port: yup
		.string()
		.required('Port is required')
		.test('valid-port', 'Port must be between 1 and 65535', (value) => {
			if (!value) return false;
			const port = parseInt(value, 10);
			return !isNaN(port) && port >= 1 && port <= 65535;
		}),
	user: yup
		.string()
		.required('Username is required')
		.min(1, 'Username must be at least 1 character')
		.max(50, 'Username must be less than 50 characters'),
	private_key_id: yup
		.string()
		.required('SSH key is required')
});

export type CreateServerForm = yup.InferType<typeof createServerSchema>;
export type UpdateServerForm = yup.InferType<typeof updateServerSchema>;