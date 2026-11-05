import { create } from 'zustand';

import { changePassword, login as loginRequest, logout as logoutRequest, me } from '@/features/auth/api';
import { clearAccessToken, markSessionHint, refreshSession, setAccessToken, setUnauthorizedHandler } from '@/lib/auth';
import { useOutletStore } from '@/stores/outlet-store';
import type { AuthUser } from '@/types/auth';

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated';

interface AuthState {
  user: AuthUser | null;
  status: AuthStatus;
  initialized: boolean;
  bootstrap: () => Promise<void>;
  login: (email: string, password: string) => Promise<AuthUser>;
  logout: () => Promise<void>;
  refreshMe: () => Promise<void>;
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>;
  clearSession: () => void;
  setUser: (user: AuthUser) => void;
}

let bootstrapping = false;

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  status: 'loading',
  initialized: false,

  bootstrap: async () => {
    if (get().initialized || bootstrapping) return;
    bootstrapping = true;
    try {
      const session = await refreshSession();
      if (session) {
        useOutletStore.getState().syncWithUser(session.user);
        set({ user: session.user, status: 'authenticated', initialized: true });
      } else {
        clearAccessToken();
        useOutletStore.getState().syncWithUser(null);
        set({ user: null, status: 'unauthenticated', initialized: true });
      }
    } finally {
      bootstrapping = false;
    }
  },

  login: async (email, password) => {
    const session = await loginRequest(email, password);
    setAccessToken(session.access_token);
    markSessionHint(true);
    useOutletStore.getState().syncWithUser(session.user);
    set({ user: session.user, status: 'authenticated', initialized: true });
    return session.user;
  },

  logout: async () => {
    try {
      await logoutRequest();
    } catch {
      // A revoked or expired session is already logged out locally.
    } finally {
      clearAccessToken();
      markSessionHint(false);
      useOutletStore.getState().syncWithUser(null);
      set({ user: null, status: 'unauthenticated', initialized: true });
    }
  },

  refreshMe: async () => {
    const user = await me();
    useOutletStore.getState().syncWithUser(user);
    set({ user, status: 'authenticated' });
  },

  changePassword: async (currentPassword, newPassword) => {
    await changePassword(currentPassword, newPassword);
  },

  clearSession: () => {
    clearAccessToken();
    markSessionHint(false);
    useOutletStore.getState().syncWithUser(null);
    set({ user: null, status: 'unauthenticated', initialized: true });
  },

  setUser: (user) => {
    useOutletStore.getState().syncWithUser(user);
    set({ user, status: 'authenticated' });
  },
}));

setUnauthorizedHandler(() => {
  useAuthStore.getState().clearSession();
});
