const STORAGE_KEY = 'fixphone.active_outlet';

function readStored(): string | null {
  try {
    return window.localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

let activeOutletId: string | null = readStored();

/**
 * The outlet used for the X-Outlet-Id header. Kept in sync by the outlet store;
 * tokens never touch browser storage.
 */
export function getActiveOutletId(): string | null {
  return activeOutletId;
}

export function persistActiveOutletId(outletId: string | null): void {
  activeOutletId = outletId;
  try {
    if (outletId) {
      window.localStorage.setItem(STORAGE_KEY, outletId);
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Storage can be unavailable in private modes; the in-memory value still works.
  }
}
