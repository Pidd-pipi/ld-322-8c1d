import axios from 'axios';
const client = axios.create({ baseURL: '/api/v1' });
client.interceptors.request.use((config) => { const token = localStorage.getItem('token'); if (token) config.headers.Authorization = `Bearer ${token}`; return config; });
client.interceptors.response.use(
  (response) => response,
  (error) => {
    const reason = error?.response?.data?.message;
    if (typeof reason === 'string' && reason) error.message = reason;
    return Promise.reject(error);
  },
);
export default client;
