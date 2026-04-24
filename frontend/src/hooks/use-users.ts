import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { usersService } from '@/services/users.service';
import type { CreateUserRequest, UpdateUserRequest, ListFilter } from '@/types';

// ── Query Keys (typed, hierarchical) ─────────────────────────
export const userKeys = {
  all: ['users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (filter: ListFilter) => [...userKeys.lists(), filter] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};

// ── Queries ───────────────────────────────────────────────────

export function useUsers(filter: ListFilter = {}) {
  return useQuery({
    queryKey: userKeys.list(filter),
    queryFn: () => usersService.list(filter),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () => usersService.getById(id),
    enabled: Boolean(id),
    staleTime: 60_000,
  });
}

// ── Mutations ─────────────────────────────────────────────────

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateUserRequest) => usersService.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: userKeys.lists() });
      toast.success('User created successfully');
    },
    onError: () => {
      toast.error('Failed to create user');
    },
  });
}

export function useUpdateUser(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateUserRequest) => usersService.update(id, payload),
    onSuccess: (updatedUser) => {
      qc.setQueryData(userKeys.detail(id), updatedUser);
      qc.invalidateQueries({ queryKey: userKeys.lists() });
      toast.success('User updated');
    },
    onError: () => toast.error('Failed to update user'),
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => usersService.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: userKeys.lists() });
      toast.success('User deleted');
    },
    onError: () => toast.error('Failed to delete user'),
  });
}
