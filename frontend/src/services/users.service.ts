import { api } from './api-client';
import type {
  LoginRequest,
  LoginResponse,
  User,
  CreateUserRequest,
  UpdateUserRequest,
  PaginatedResponse,
  ListFilter,
} from '@/types';

// ============================================================
// Auth Service
// ============================================================

export const authService = {
  login: (payload: LoginRequest): Promise<LoginResponse> =>
    api.post<LoginResponse>('/auth/login', payload),

  refresh: (refreshToken: string): Promise<{ accessToken: string; expiresIn: number }> =>
    api.post('/auth/refresh', { refreshToken }),

  logout: (): Promise<void> =>
    api.post('/auth/logout'),
};

// ============================================================
// Users Service
// ============================================================

export const usersService = {
  list: (filter?: ListFilter): Promise<PaginatedResponse<User>> => {
    const params = new URLSearchParams();
    if (filter?.page) params.set('page', String(filter.page));
    if (filter?.pageSize) params.set('page_size', String(filter.pageSize));
    if (filter?.sortBy) params.set('sort_by', filter.sortBy);
    if (filter?.order) params.set('order', filter.order);
    if (filter?.search) params.set('search', filter.search);
    return api.get<PaginatedResponse<User>>(`/users?${params}`);
  },

  getById: (id: string): Promise<User> =>
    api.get<{ user: User }>(`/users/${id}`).then((r) => r.user),

  create: (payload: CreateUserRequest): Promise<User> =>
    api.post<{ user: User }>('/users', payload).then((r) => r.user),

  update: (id: string, payload: UpdateUserRequest): Promise<User> =>
    api.put<{ user: User }>(`/users/${id}`, payload).then((r) => r.user),

  delete: (id: string): Promise<void> =>
    api.delete(`/users/${id}`),
};
