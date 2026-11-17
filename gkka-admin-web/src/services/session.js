const keys = ['access_token', 'refresh_token', 'user', 'token', 'role'];
export function clearSession() {
  for (const storage of [localStorage, sessionStorage]) {
    keys.forEach(key => storage.removeItem(key));
  }
}
export function getSession() {
  for (const storage of [sessionStorage, localStorage]) {
    try {
      const accessToken = storage.getItem('access_token');
      const user = JSON.parse(storage.getItem('user') || 'null');
      if (accessToken && user) return { accessToken, refreshToken: storage.getItem('refresh_token'), user, storage };
    } catch { /* Invalid stored session is ignored. */ }
  }
  return null;
}
export function saveSession(data, remember = false) {
  if (!data?.access_token || !data?.refresh_token || !data?.user?.role) throw new Error('Respons sesi tidak lengkap');
  clearSession();
  const storage = remember ? localStorage : sessionStorage;
  storage.setItem('access_token', data.access_token);
  storage.setItem('refresh_token', data.refresh_token);
  storage.setItem('user', JSON.stringify(data.user));
}
export function updateSessionTokens(data, previous) {
  const active = getSession();
  if (!active || active.refreshToken !== previous.refreshToken) throw new Error('Sesi telah berubah');
  if (!data?.access_token || !data?.refresh_token) throw new Error('Respons refresh tidak lengkap');
  active.storage.setItem('access_token', data.access_token);
  active.storage.setItem('refresh_token', data.refresh_token);
}
