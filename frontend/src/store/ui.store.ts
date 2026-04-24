import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import type { UIState } from '@/types';

interface UIStore extends UIState {
  toggleSidebar: () => void;
  setSidebar: (open: boolean) => void;
  toggleTheme: () => void;
}

export const useUIStore = create<UIStore>()(
  devtools(
    persist(
      (set) => ({
        sidebarOpen: true,
        theme: 'dark',

        toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
        setSidebar: (open) => set({ sidebarOpen: open }),
        toggleTheme: () =>
          set((s) => ({ theme: s.theme === 'dark' ? 'light' : 'dark' })),
      }),
      { name: 'msp:ui' },
    ),
    { name: 'UIStore' },
  ),
);
