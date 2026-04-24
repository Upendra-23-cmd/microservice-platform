import { api } from './api-client';
import type {
  Product,
  CreateProductRequest,
  UpdateProductRequest,
  PaginatedResponse,
  ProductListFilter,
} from '@/types';

export const productsService = {
  list: (filter?: ProductListFilter): Promise<PaginatedResponse<Product>> => {
    const params = new URLSearchParams();
    if (filter?.page) params.set('page', String(filter.page));
    if (filter?.pageSize) params.set('page_size', String(filter.pageSize));
    if (filter?.sortBy) params.set('sort_by', filter.sortBy);
    if (filter?.order) params.set('order', filter.order);
    if (filter?.search) params.set('search', filter.search);
    if (filter?.category) params.set('category', filter.category);
    if (filter?.minPrice !== undefined) params.set('min_price', String(filter.minPrice));
    if (filter?.maxPrice !== undefined) params.set('max_price', String(filter.maxPrice));
    return api.get<PaginatedResponse<Product>>(`/products?${params}`);
  },

  getById: (id: string): Promise<Product> =>
    api.get<{ product: Product }>(`/products/${id}`).then((r) => r.product),

  create: (payload: CreateProductRequest): Promise<Product> =>
    api.post<{ product: Product }>('/products', payload).then((r) => r.product),

  update: (id: string, payload: UpdateProductRequest): Promise<Product> =>
    api.put<{ product: Product }>(`/products/${id}`, payload).then((r) => r.product),

  delete: (id: string): Promise<void> =>
    api.delete(`/products/${id}`),

  updateStock: (
    id: string,
    quantity: number,
    op: 'increment' | 'decrement' | 'set',
  ): Promise<{ id: string; stock: number }> =>
    api.patch(`/products/${id}/stock`, { quantity, op }),
};
