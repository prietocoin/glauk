const STORAGE_KEY = 'glauk.adminKey';

export function adminKey() {
  try { return sessionStorage.getItem(STORAGE_KEY) || ''; } catch { return ''; }
}

export function saveAdminKey(value) {
  const clean = String(value ?? '').trim();
  try {
    if (clean) sessionStorage.setItem(STORAGE_KEY, clean);
    else sessionStorage.removeItem(STORAGE_KEY);
  } catch { /* storage may be disabled */ }
  return clean;
}

export function clearAdminKey() {
  return saveAdminKey('');
}

export function hasAdminKey() {
  return adminKey().length > 0;
}

export function adminHeaders() {
  const key = adminKey();
  return key ? { 'X-Admin-Key': key } : {};
}
