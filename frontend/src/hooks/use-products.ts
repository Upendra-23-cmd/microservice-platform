import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { productsService } from '@/services/products.service';
import type {
  CreateProductRequest,
  UpdateProductRequest,
  ProductListFilter,
} from '@/types';

export const productKeys = {
  all: ['products'] as const,
  lists: () => [...productKeys.all, 'list'] as const,
  list: (filter: ProductListFilter) => [...productKeys.lists(), filter] as const,
  details: () => [...productKeys.all, 'detail'] as const,
  detail: (id: string) => [...productKeys.details(), id] as const,
};

export function useProducts(filter: ProductListFilter = {}) {
  return useQuery({
    queryKey: productKeys.list(filter),
    queryFn: () => productsService.list(filter),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
}

export function useProduct(id: string) {
  return useQuery({
    queryKey: productKeys.detail(id),
    queryFn: () => productsService.getById(id),
    enabled: Boolean(id),
    staleTime: 60_000,
  });
}

export function useCreateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateProductRequest) => productsService.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: productKeys.lists() });
      toast.success('Product created');
    },
    onError: () => toast.error('Failed to create product'),
  });
}

export function useUpdateProduct(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateProductRequest) => productsService.update(id, payload),
    onSuccess: (updated) => {
      qc.setQueryData(productKeys.detail(id), updated);
      qc.invalidateQueries({ queryKey: productKeys.lists() });
      toast.success('Product updated');
    },
    onError: () => toast.error('Failed to update product'),
  });
}

export function useDeleteProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => productsService.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: productKeys.lists() });
      toast.success('Product deleted');
    },
    onError: () => toast.error('Failed to delete product'),
  });
}

export function useUpdateStock() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      quantity,
      op,
    }: {
      id: string;
      quantity: number;
      op: 'increment' | 'decrement' | 'set';
    }) => productsService.updateStock(id, quantity, op),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: productKeys.detail(id) });
      qc.invalidateQueries({ queryKey: productKeys.lists() });
      toast.success('Stock updated');
    },
    onError: () => toast.error('Failed to update stock'),
  });
}
