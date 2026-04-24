import { Suspense, lazy, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { Toaster } from 'react-hot-toast';
import { useAuthStore } from '@/store/auth.store';
import { useUIStore } from '@/store/ui.store';
import { AppLayout } from '@/components/layout/AppLayout';
import { LoadingScreen } from '@/components/ui/LoadingScreen';

// Lazy-loaded pages for code splitting
const LoginPage     = lazy(() => import('@/components/features/auth/LoginPage'));
const DashboardPage = lazy(() => import('@/components/features/dashboard/DashboardPage'));
const UsersPage     = lazy(() => import('@/components/features/users/UsersPage'));
const ProductsPage  = lazy(() => import('@/components/features/products/ProductsPage'));
const NotFoundPage  = lazy(() => import('@/components/features/NotFoundPage'));

// ── React Query Client ────────────────────────────────────────
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error: any) => {
        // Don't retry auth errors
        if (error?.code === 401 || error?.code === 403) return false;
        return failureCount < 2;
      },
      refetchOnWindowFocus: import.meta.env.PROD,
      staleTime: 30_000,
    },
    mutations: {
      retry: false,
    },
  },
});

// ── Protected Route ───────────────────────────────────────────
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
}

// ── App Root ──────────────────────────────────────────────────
export default function App() {
  const theme = useUIStore((s) => s.theme);

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark');
  }, [theme]);

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Suspense fallback={<LoadingScreen />}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
              path="/"
              element={
                <ProtectedRoute>
                  <AppLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<DashboardPage />} />
              <Route path="users" element={<UsersPage />} />
              <Route path="products" element={<ProductsPage />} />
            </Route>
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </Suspense>
      </BrowserRouter>

      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: theme === 'dark' ? '#1e293b' : '#ffffff',
            color: theme === 'dark' ? '#f1f5f9' : '#0f172a',
            border: '1px solid',
            borderColor: theme === 'dark' ? '#334155' : '#e2e8f0',
            borderRadius: '8px',
            fontSize: '14px',
          },
        }}
      />

      {import.meta.env.VITE_ENABLE_DEVTOOLS === 'true' && (
        <ReactQueryDevtools initialIsOpen={false} />
      )}
    </QueryClientProvider>
  );
}
