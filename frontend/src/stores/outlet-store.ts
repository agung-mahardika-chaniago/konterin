import { create } from 'zustand';

import { getActiveOutletId, persistActiveOutletId } from '@/lib/outlet';
import type { AuthUser } from '@/types/auth';

interface OutletState {
  outletId: string | null;
  setOutletId: (outletId: string | null) => void;
  syncWithUser: (user: AuthUser | null) => void;
}

/**
 * Resolves the persisted outlet against the outlets the signed-in user can
 * actually access, so a stale or foreign id is never sent to the API.
 */
export const useOutletStore = create<OutletState>((set) => ({
  outletId: getActiveOutletId(),

  setOutletId: (outletId) => {
    persistActiveOutletId(outletId);
    set({ outletId });
  },

  syncWithUser: (user) => {
    if (!user) {
      persistActiveOutletId(null);
      set({ outletId: null });
      return;
    }

    const available = user.outlets.map((outlet) => outlet.id);
    const stored = getActiveOutletId();
    const candidates = [stored, user.active_outlet_id, available[0] ?? null];
    const next = candidates.find((id): id is string => Boolean(id) && available.includes(id as string)) ?? null;

    persistActiveOutletId(next);
    set({ outletId: next });
  },
}));
