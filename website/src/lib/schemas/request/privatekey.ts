import * as yup from 'yup';

export const createPrivateKeySchema = yup.object({
	name: yup
		.string()
		.required('Name is required')
		.min(1, 'Name must be at least 1 character')
		.max(100, 'Name must be less than 100 characters'),
	description: yup
		.string()
		.max(500, 'Description must be less than 500 characters'),
	privateKey: yup
		.string()
		.required('Private key is required')
		.test('valid-private-key', 'Please enter a valid SSH private key', (value) => {
			if (!value) return false;
			// Basic validation for SSH private key format
			return value.includes('-----BEGIN') && value.includes('-----END') && 
				   (value.includes('PRIVATE KEY') || value.includes('OPENSSH PRIVATE KEY'));
		})
});

export type CreatePrivateKeyForm = yup.InferType<typeof createPrivateKeySchema>;