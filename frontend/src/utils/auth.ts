export const currentRole = (): string => {
  const token = localStorage.getItem('token');
  if (!token) return '';
  try {
    const payload = JSON.parse(atob(token.split('.')[1] ?? '')) as { role?: string };
    return payload.role ?? '';
  } catch {
    return '';
  }
};
export const isAdmin = () => currentRole() === 'admin';
