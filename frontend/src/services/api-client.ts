import axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios';
import type { ApiError, RefreshTokenResponse } from '@/types';

// ============================================================
// Constants
// ============================================================

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string;

// Storage keys — never store access tokens in localStorage in production.
// Use httpOnly cookies or in-memory storage.
const ACCESS_TOKEN_KEY = 'msp:access_token';
const REFRESH_TOKEN_KEY = 'msp:refresh_token';

// ============================================================
// Token Manager (in-memory for access token, localStorage for refresh)
// ============================================================

let inMemoryAccessToken: string | null = null;

export const tokenManager = {
  getAccessToken: (): string | null => inMemoryAccessToken,
  setAccessToken: (token: string): void => {
    inMemoryAccessToken = token;
  },
  getRefreshToken: (): string | null => localStorage.getItem(REFRESH_TOKEN_KEY),
  setRefreshToken: (token: string): void => {
    localStorage.setItem(REFRESH_TOKEN_KEY, token);
  },
  clearAll: (): void => {
    inMemoryAccessToken = null;
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(ACCESS_TOKEN_KEY);
  },
};

// ============================================================
// Axios Instance
// ============================================================

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15_000,
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
});

// ── Request Interceptor: attach Bearer token ──────────────────
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = tokenManager.getAccessToken();
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    // Attach a unique request ID for tracing
    config.headers['X-Request-ID'] = crypto.randomUUID();
    return config;
  },
  (error) => Promise.reject(error),
);

// ── Response Interceptor: handle 401 and refresh ─────────────
let isRefreshing = false;
let refreshSubscribers: Array<(token: string) => void> = [];

const subscribeTokenRefresh = (cb: (token: string) => void) => {
  refreshSubscribers.push(cb);
};

const onTokenRefreshed = (token: string) => {
  refreshSubscribers.forEach((cb) => cb(token));
  refreshSubscribers = [];
};

apiClient.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean };

    // If 401 and we haven't already retried
    if (error.response?.status === 401 && !originalRequest._retry) {
      const refreshToken = tokenManager.getRefreshToken();

      if (!refreshToken) {
        tokenManager.clearAll();
        window.location.href = '/login';
        return Promise.reject(error);
      }

      if (isRefreshing) {
        // Queue requests while refresh is in flight
        return new Promise((resolve) => {
          subscribeTokenRefresh((newToken) => {
            if (originalRequest.headers) {
              originalRequest.headers['Authorization'] = `Bearer ${newToken}`;
            }
            resolve(apiClient(originalRequest));
          });
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const { data } = await axios.post<RefreshTokenResponse>(
          `${API_BASE_URL}/auth/refresh`,
          { refreshToken },
        );

        tokenManager.setAccessToken(data.accessToken);
        onTokenRefreshed(data.accessToken);

        if (originalRequest.headers) {
          originalRequest.headers['Authorization'] = `Bearer ${data.accessToken}`;
        }

        return apiClient(originalRequest);
      } catch (refreshError) {
        tokenManager.clearAll();
        window.location.href = '/login';
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    // Normalize error shape
    const apiError: ApiError = {
      code: error.response?.status ?? 0,
      message: error.response?.data?.message ?? error.message ?? 'Unknown error',
      details: error.response?.data?.details,
    };

    return Promise.reject(apiError);
  },
);

// ============================================================
// Typed API helpers
// ============================================================

export const api = {
  get: <T>(url: string, config?: AxiosRequestConfig) =>
    apiClient.get<T>(url, config).then((r) => r.data),

  post: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) =>
    apiClient.post<T>(url, data, config).then((r) => r.data),

  put: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) =>
    apiClient.put<T>(url, data, config).then((r) => r.data),

  patch: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) =>
    apiClient.patch<T>(url, data, config).then((r) => r.data),

  delete: <T>(url: string, config?: AxiosRequestConfig) =>
    apiClient.delete<T>(url, config).then((r) => r.data),
};
