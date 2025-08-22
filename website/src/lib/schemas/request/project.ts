import * as yup from 'yup';

export const createProjectSchema = yup.object({
	name: yup
		.string()
		.required('Name is required')
		.min(1, 'Name must be at least 1 character')
		.max(100, 'Name must be less than 100 characters'),
	description: yup.string().max(500, 'Description must be less than 500 characters')
});

export const updateProjectSchema = yup.object({
	name: yup
		.string()
		.required('Name is required')
		.min(1, 'Name must be at least 1 character')
		.max(100, 'Name must be less than 100 characters'),
	description: yup.string().max(500, 'Description must be less than 500 characters')
});

export type CreateProjectForm = yup.InferType<typeof createProjectSchema>;
export type UpdateProjectForm = yup.InferType<typeof updateProjectSchema>;
