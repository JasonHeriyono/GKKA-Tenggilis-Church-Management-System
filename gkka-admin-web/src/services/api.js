import axios from 'axios';
import { getSession, clearSession, updateSessionTokens } from './session';
const baseURL = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/gkki/api').replace(/\/$/, '');
const transport = axios.create({ baseURL, timeout: 20000 });
const api = axios.create({ baseURL, timeout: 20000 });
let refreshing = null;
const publicAuth = url => /\/auth\/(login|refresh|signup|logout)(?:\?|$)/.test(url || '');
api.interceptors.request.use(config => {
  const session = getSession();
  if (session && !publicAuth(config.url)) config.headers.set('Authorization', `Bearer ${session.accessToken}`);
  else config.headers.delete('Authorization');
  if (config.data instanceof FormData) config.headers.delete('Content-Type');
  return config;
});
api.interceptors.response.use(response => response, async error => {
  const original = error.config;
  if (!original || error.response?.status !== 401 || original._retry || publicAuth(original.url)) throw error;
  original._retry = true;
  const previous = getSession();
  if (!previous?.refreshToken) {
    clearSession();
    window.location.replace('/login');
    throw error;
  }
  try {
    if (!refreshing) {
      refreshing = (async () => {
        const response = await transport.post('/auth/refresh', { refresh_token: previous.refreshToken });
        updateSessionTokens(response.data.data, previous);
      })().finally(() => { refreshing = null; });
    }
    await refreshing;
  } catch (refreshError) {
    // Do not remove a new session created while the old refresh was in flight.
    const active = getSession();
    if (!active || active.refreshToken === previous.refreshToken) {
      const status = refreshError.response?.status;
      if (status === 401 || status === 403 || !active) {
        clearSession();
        window.location.replace('/login');
      }
    }
    throw refreshError;
  }
  return api(original);
});
export async function logout() {
  const current = getSession();
  clearSession();
  if (current?.refreshToken) {
    try { await transport.post('/auth/logout', { refresh_token: current.refreshToken }); }
    catch (error) { console.warn('Sesi lokal dihapus; pencabutan sesi server gagal.', error.message); }
  }
}
api.interceptors.response.use(response => {
  const method = response.config.method?.toLowerCase();
  const url = response.config.url || '';

  const isUserMutation =
    ['post', 'put', 'patch', 'delete'].includes(method) &&
    /\/users(?:\/|\?|$)/.test(url);

  const isRegistrationMutation =
    method === 'post' &&
    (
      /\/auth\/signup(?:\?|$)/.test(url) ||
      /\/pendaftaran\/[^/]+\/(?:setujui|tolak)(?:\?|$)/.test(url)
    );

  if (
    response.data?.success === true &&
    (isUserMutation || isRegistrationMutation)
  ) {
    window.dispatchEvent(new Event('jemaat-updated'));
  }

  return response;
});

export default api;
