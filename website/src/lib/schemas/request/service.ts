import * as yup from 'yup';

export const createServiceSchema = yup.object({
	name: yup
		.string()
		.required('Service name is required')
		.min(1, 'Service name must be at least 1 character')
		.max(100, 'Service name must be less than 100 characters'),
	description: yup.string().max(500, 'Description must be less than 500 characters'),
	type: yup
		.string()
		.required('Service type is required')
		.oneOf(['compose'], 'Invalid service type'),
	server_id: yup.string().required('Server selection is required'),
	compose_file: yup
		.string()
		.required('Docker Compose file content is required')
		.min(10, 'Compose file content is too short')
});

export type CreateServiceForm = yup.InferType<typeof createServiceSchema>;

export const updateServiceComposeSchema = yup.object({
	compose_file: yup
		.string()
		.required('Docker Compose file content is required')
		.min(10, 'Compose file content is too short')
});

export type UpdateServiceComposeForm = yup.InferType<typeof updateServiceComposeSchema>;
