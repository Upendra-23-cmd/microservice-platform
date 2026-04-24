// ============================================================
// Domain Types — mirrors the gRPC/protobuf schema
// ============================================================

export type UserRole = 'admin' | 'manager' | 'member' | 'guest';

export interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  role: UserRole;
  isActive: boolean;
  createdAt: string; // ISO 8601
  updatedAt: string;
}

export interface Product {
  id: string;
  name: string;
  description: string;
  price: number;
  stock: number;
  category: string;
  isActive: boolean;
  tags?: string[];
  metadata?: Record<string, string>;
  createdAt: string;
  updatedAt: string;
}

export interface Order {
  id: string;
  userId: string;
  status: OrderStatus;
  total: number;
  currency: string;
  notes?: string;
  items: OrderItem[];
  createdAt: string;
  updatedAt: string;
}

export type OrderStatus =
  | 'pending'
  | 'confirmed'
  | 'processing'
  | 'shipped'
  | 'delivered'
  | 'cancelled'
  | 'refunded';

export interface OrderItem {
  id: string;
  orderId: string;
  productId: string;
  quantity: number;
  unitPrice: number;
  subtotal: number;
}

// ============================================================
// API Types
// ============================================================

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pages: number;
}

export interface ApiError {
  code: number;
  message: string;
  details?: string[];
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: User;
}

export interface RefreshTokenResponse {
  accessToken: string;
  expiresIn: number;
}

export interface CreateUserRequest {
  email: string;
  password: string;
  firstName: string;
  lastName: string;
}

export interface UpdateUserRequest {
  firstName?: string;
  lastName?: string;
}

export interface CreateProductRequest {
  name: string;
  description?: string;
  price: number;
  stock: number;
  category: string;
  tags?: string[];
  metadata?: Record<string, string>;
}

export interface UpdateProductRequest extends Partial<CreateProductRequest> {}

export interface ListFilter {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  order?: 'asc' | 'desc';
  search?: string;
}

export interface ProductListFilter extends ListFilter {
  category?: string;
  minPrice?: number;
  maxPrice?: number;
}

// ============================================================
// Store Types
// ============================================================

export interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

export interface UIState {
  sidebarOpen: boolean;
  theme: 'light' | 'dark';
}
