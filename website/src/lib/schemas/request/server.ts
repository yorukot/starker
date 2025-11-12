import * as yup from 'yup';

const ipv4Regex =
    /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
const ipv6Regex =
    /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:)*::([0-9a-fA-F]{1,4}:)*[0-9a-fA-F]{1,4}|::)$/;
const hostnameRegex =
    /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;

export const createServerSchema = yup.object({
    name: yup
        .string()
        .required('Name is required')
        .min(1, 'Name must be at least 1 character')
        .max(100, 'Name must be less than 100 characters'),
    description: yup.string().max(500, 'Description must be less than 500 characters'),
    host: yup
        .string()
        .required('Host / IP address is required')
        .test('valid-host', 'Please enter a valid IP address or hostname', (value) => {
            if (!value) return false;
            if (value.length > 253) return false;
            return ipv4Regex.test(value) || ipv6Regex.test(value) || hostnameRegex.test(value);
        }),
    port: yup
        .number()
        .required('Port is required')
        .integer('Port must be an integer')
        .min(1, 'Port must be at least 1')
        .max(65535, 'Port must be at most 65535'),
    user: yup
        .string()
        .required('Username is required')
        .min(1, 'Username must be at least 1 character')
        .max(50, 'Username must be less than 50 characters'),
    private_key_id: yup.string().required('SSH key is required')
});

export const updateServerSchema = yup.object({
    name: yup
        .string()
        .required('Name is required')
        .min(1, 'Name must be at least 1 character')
        .max(100, 'Name must be less than 100 characters'),
    description: yup.string().max(500, 'Description must be less than 500 characters'),
    host: yup
        .string()
        .required('Host / IP address is required')
        .test('valid-host', 'Please enter a valid IP address or hostname', (value) => {
            if (!value) return false;
            if (value.length > 253) return false;
            return ipv4Regex.test(value) || ipv6Regex.test(value) || hostnameRegex.test(value);
        }),
    port: yup
        .number()
        .required('Port is required')
        .integer('Port must be an integer')
        .min(1, 'Port must be at least 1')
        .max(65535, 'Port must be at most 65535'),
    user: yup
        .string()
        .required('Username is required')
        .min(1, 'Username must be at least 1 character')
        .max(50, 'Username must be less than 50 characters'),
    private_key_id: yup.string().required('SSH key is required')
});

export type CreateServerForm = yup.InferType<typeof createServerSchema>;
export type UpdateServerForm = yup.InferType<typeof updateServerSchema>;
