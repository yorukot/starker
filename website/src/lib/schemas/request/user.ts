import * as yup from 'yup';

export const registerSchema = yup.object({
	displayName: yup.string().required('Display name is required'),
	email: yup.string().required('Email is required').email('Please enter a valid email address.'),
	password: yup
		.string()
		.required('Password is required')
		.min(8, 'Password must be at least 8 characters')
		.matches(
			/^(?=.*[A-Z])(?=.*[a-z])(?=.*[0-9]).+$/,
			'Password must contain at least one uppercase letter, one lowercase letter, and one number'
		),
	confirmPassword: yup
		.string()
		.required('Please confirm your password')
		.oneOf([yup.ref('password')], 'Passwords do not match')
});

export type RegisterForm = yup.InferType<typeof registerSchema>;

export const loginSchema = yup.object({
	email: yup.string().required('Email is required').email('Please enter a valid email address.'),
	password: yup.string().required('Password is required')
});

export type LoginForm = yup.InferType<typeof loginSchema>;
