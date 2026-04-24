import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import type { AuthState, LoginRequest } from '@/types';
import { authService } from '@/services/users.service';
import { tokenManager } from '@/services/api-client';

interface AuthStore extends AuthState {
  login: (payload: LoginRequest) => Promise<void>;
  logout: () => void;
  setAccessToken: (token: string) => void;
}

export const useAuthStore = create<AuthStore>()(
  devtools(
    persist(
      (set) => ({
        user: null,
        accessToken: null,
        refreshToken: null,
        isAuthenticated: false,
        isLoading: false,

        login: async (payload) => {
          set({ isLoading: true });
          try {
            const data = await authService.login(payload);
            tokenManager.setAccessToken(data.accessToken);
            tokenManager.setRefreshToken(data.refreshToken);
            set({
              user: data.user,
              accessToken: data.accessToken,
              refreshToken: data.refreshToken,
              isAuthenticated: true,
              isLoading: false,
            });
          } catch (err) {
            set({ isLoading: false });
            throw err;
          }
        },

        logout: () => {
          tokenManager.clearAll();
          set({
            user: null,
            accessToken: null,
            refreshToken: null,
            isAuthenticated: false,
          });
        },

        setAccessToken: (token) => {
          tokenManager.setAccessToken(token);
          set({ accessToken: token });
        },
      }),
      {
        name: 'msp:auth',
        // Only persist non-sensitive fields; access token stays in-memory
        partialize: (state) => ({
          user: state.user,
          refreshToken: state.refreshToken,
          isAuthenticated: state.isAuthenticated,
        }),
      },
    ),
    { name: 'AuthStore' },
  ),
);
