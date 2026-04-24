import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Search, Plus, Trash2, ChevronLeft, ChevronRight,
  Loader2, Package, X, Filter, ArrowUpDown,
} from 'lucide-react';
import { useProducts, useCreateProduct, useDeleteProduct, useUpdateStock } from '@/hooks/use-products';
import type { Product } from '@/types';
import { cn } from '@/utils/cn';
import { format } from 'date-fns';

// ── Schema ────────────────────────────────────────────────────
const createSchema = z.object({
  name:        z.string().min(1, 'Required'),
  description: z.string().optional(),
  price:       z.coerce.number().positive('Must be > 0'),
  stock:       z.coerce.number().int().min(0, 'Cannot be negative'),
  category:    z.string().min(1, 'Required'),
  tags:        z.string().optional(), // comma-separated
});
type CreateForm = z.infer<typeof createSchema>;

// ── Shared classes ────────────────────────────────────────────
const inputClass =
  'w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm ' +
  'text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 ' +
  'focus:ring-cyan-500/40 transition-all';
const primaryBtnClass =
  'flex items-center justify-center gap-2 rounded-lg px-4 py-2 bg-cyan-500 ' +
  'hover:bg-cyan-400 text-slate-950 font-semibold text-sm transition-colors disabled:opacity-50';
const secondaryBtnClass =
  'rounded-lg px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm transition-colors';
const paginationBtn =
  'p-1.5 rounded text-slate-400 hover:bg-slate-800 hover:text-slate-100 ' +
  'disabled:opacity-30 disabled:cursor-not-allowed transition-colors';

// ── Stock Badge ───────────────────────────────────────────────
function StockBadge({ stock }: { stock: number }) {
  if (stock === 0)
    return <span className="text-rose-400 font-medium text-xs">Out of stock</span>;
  if (stock < 10)
    return <span className="text-amber-400 font-medium text-xs">{stock} low</span>;
  return <span className="text-emerald-400 font-medium text-xs">{stock} in stock</span>;
}

// ── Stock Control ─────────────────────────────────────────────
function StockControl({ product }: { product: Product }) {
  const { mutate: updateStock, isPending } = useUpdateStock();
  return (
    <div className="flex items-center gap-1">
      <button
        onClick={() => updateStock({ id: product.id, quantity: 1, op: 'decrement' })}
        disabled={isPending || product.stock === 0}
        className="w-6 h-6 flex items-center justify-center rounded bg-slate-700 hover:bg-slate-600 text-slate-300 text-xs disabled:opacity-30 transition-colors"
      >
        −
      </button>
      <span className="w-10 text-center text-sm font-mono text-slate-200">
        {isPending ? <Loader2 size={12} className="animate-spin mx-auto" /> : product.stock}
      </span>
      <button
        onClick={() => updateStock({ id: product.id, quantity: 1, op: 'increment' })}
        disabled={isPending}
        className="w-6 h-6 flex items-center justify-center rounded bg-slate-700 hover:bg-slate-600 text-slate-300 text-xs disabled:opacity-30 transition-colors"
      >
        +
      </button>
    </div>
  );
}

// ── Create Product Modal ──────────────────────────────────────
function CreateProductModal({ onClose }: { onClose: () => void }) {
  const { mutateAsync, isPending } = useCreateProduct();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<CreateForm>({ resolver: zodResolver(createSchema) });

  const onSubmit = async (data: CreateForm) => {
    await mutateAsync({
      ...data,
      tags: data.tags ? data.tags.split(',').map((t) => t.trim()).filter(Boolean) : [],
    });
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-lg bg-slate-900 border border-slate-700 rounded-2xl p-6 shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-base font-semibold text-slate-100">Create Product</h2>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-100 transition-colors">
            <X size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          {/* Name */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Product Name</label>
            <input {...register('name')} className={cn(inputClass, errors.name && 'border-rose-500/60')} placeholder="Wireless Headphones" />
            {errors.name && <p className="mt-1 text-xs text-rose-400">{errors.name.message}</p>}
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Description</label>
            <textarea
              {...register('description')}
              rows={2}
              className={cn(inputClass, 'resize-none')}
              placeholder="Product description…"
            />
          </div>

          {/* Price + Stock */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs text-slate-400 mb-1">Price ($)</label>
              <input
                {...register('price')}
                type="number"
                step="0.01"
                className={cn(inputClass, errors.price && 'border-rose-500/60')}
                placeholder="29.99"
              />
              {errors.price && <p className="mt-1 text-xs text-rose-400">{errors.price.message}</p>}
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Initial Stock</label>
              <input
                {...register('stock')}
                type="number"
                className={cn(inputClass, errors.stock && 'border-rose-500/60')}
                placeholder="100"
              />
              {errors.stock && <p className="mt-1 text-xs text-rose-400">{errors.stock.message}</p>}
            </div>
          </div>

          {/* Category */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Category</label>
            <select {...register('category')} className={cn(inputClass, errors.category && 'border-rose-500/60')}>
              <option value="">Select category…</option>
              {['electronics', 'clothing', 'books', 'food', 'sports', 'home', 'other'].map((c) => (
                <option key={c} value={c} className="capitalize">{c.charAt(0).toUpperCase() + c.slice(1)}</option>
              ))}
            </select>
            {errors.category && <p className="mt-1 text-xs text-rose-400">{errors.category.message}</p>}
          </div>

          {/* Tags */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">
              Tags <span className="text-slate-500">(comma-separated)</span>
            </label>
            <input {...register('tags')} className={inputClass} placeholder="wireless, premium, audio" />
          </div>

          {/* Buttons */}
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={onClose} className={secondaryBtnClass}>Cancel</button>
            <button type="submit" disabled={isPending} className={cn(primaryBtnClass, 'flex-1')}>
              {isPending ? <Loader2 size={15} className="animate-spin mx-auto" /> : 'Create Product'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ── Products Page ─────────────────────────────────────────────
const CATEGORIES = ['', 'electronics', 'clothing', 'books', 'food', 'sports', 'home', 'other'];

export default function ProductsPage() {
  const [page, setPage]         = useState(1);
  const [search, setSearch]     = useState('');
  const [category, setCategory] = useState('');
  const [sortBy, setSortBy]     = useState('created_at');
  const [order, setOrder]       = useState<'asc' | 'desc'>('desc');
  const [showCreate, setShowCreate] = useState(false);

  const { data, isLoading } = useProducts({
    page,
    pageSize: 10,
    search: search || undefined,
    category: category || undefined,
    sortBy,
    order,
  });

  const { mutate: deleteProduct } = useDeleteProduct();

  const products: Product[] = data?.data ?? [];
  const totalPages = data?.pages ?? 1;

  const toggleSort = (col: string) => {
    if (sortBy === col) setOrder((o) => (o === 'asc' ? 'desc' : 'asc'));
    else { setSortBy(col); setOrder('desc'); }
  };

  const SortBtn = ({ col, label }: { col: string; label: string }) => (
    <button
      onClick={() => toggleSort(col)}
      className="flex items-center gap-1 group"
    >
      {label}
      <ArrowUpDown
        size={12}
        className={cn(
          'transition-colors',
          sortBy === col ? 'text-cyan-400' : 'text-slate-600 group-hover:text-slate-400',
        )}
      />
    </button>
  );

  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-slate-100">Products</h1>
          <p className="text-sm text-slate-400 mt-0.5">
            {data?.total ?? '—'} total products
          </p>
        </div>
        <button onClick={() => setShowCreate(true)} className={primaryBtnClass}>
          <Plus size={16} /> New Product
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
          <input
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            placeholder="Search products…"
            className={cn(inputClass, 'pl-9')}
          />
        </div>

        <div className="relative">
          <Filter size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
          <select
            value={category}
            onChange={(e) => { setCategory(e.target.value); setPage(1); }}
            className={cn(inputClass, 'pl-8 pr-8 w-auto')}
          >
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {c ? c.charAt(0).toUpperCase() + c.slice(1) : 'All Categories'}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-800/40">
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  <SortBtn col="name" label="Product" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Category</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  <SortBtn col="price" label="Price" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Stock</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  <SortBtn col="created_at" label="Added" />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center">
                    <Loader2 size={24} className="animate-spin text-cyan-400 mx-auto" />
                  </td>
                </tr>
              ) : products.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center">
                    <Package size={32} className="text-slate-600 mx-auto mb-3" />
                    <p className="text-slate-500 text-sm">No products found</p>
                  </td>
                </tr>
              ) : (
                products.map((p) => (
                  <tr key={p.id} className="hover:bg-slate-800/40 transition-colors group">
                    {/* Product */}
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-lg bg-violet-500/10 border border-violet-500/20 flex items-center justify-center shrink-0">
                          <Package size={14} className="text-violet-400" />
                        </div>
                        <div>
                          <p className="font-medium text-slate-100 line-clamp-1 max-w-[180px]">{p.name}</p>
                          {p.description && (
                            <p className="text-xs text-slate-500 line-clamp-1 max-w-[180px]">{p.description}</p>
                          )}
                        </div>
                      </div>
                    </td>

                    {/* Category */}
                    <td className="px-4 py-3">
                      <span className="text-xs text-slate-300 bg-slate-800 rounded px-2 py-0.5 capitalize">
                        {p.category}
                      </span>
                    </td>

                    {/* Price */}
                    <td className="px-4 py-3 font-mono text-slate-200 text-sm">
                      ${p.price.toFixed(2)}
                    </td>

                    {/* Stock — inline control */}
                    <td className="px-4 py-3">
                      <div className="flex flex-col gap-1">
                        <StockControl product={p} />
                        <StockBadge stock={p.stock} />
                      </div>
                    </td>

                    {/* Status */}
                    <td className="px-4 py-3">
                      <span className={cn(
                        'inline-flex items-center gap-1.5 text-xs font-medium',
                        p.isActive ? 'text-emerald-400' : 'text-slate-500',
                      )}>
                        <span className={cn('w-1.5 h-1.5 rounded-full', p.isActive ? 'bg-emerald-400' : 'bg-slate-500')} />
                        {p.isActive ? 'Active' : 'Inactive'}
                      </span>
                    </td>

                    {/* Date */}
                    <td className="px-4 py-3 text-slate-400 text-xs">
                      {format(new Date(p.createdAt), 'MMM d, yyyy')}
                    </td>

                    {/* Actions */}
                    <td className="px-4 py-3">
                      <button
                        onClick={() => deleteProduct(p.id)}
                        className="opacity-0 group-hover:opacity-100 p-1.5 rounded text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 transition-all"
                      >
                        <Trash2 size={14} />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-slate-800">
            <span className="text-xs text-slate-400">
              Page {page} of {totalPages} · {data?.total} products
            </span>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className={paginationBtn}
              >
                <ChevronLeft size={15} />
              </button>
              {/* Page numbers */}
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                const pg = i + 1;
                return (
                  <button
                    key={pg}
                    onClick={() => setPage(pg)}
                    className={cn(
                      'w-7 h-7 rounded text-xs font-medium transition-colors',
                      pg === page
                        ? 'bg-cyan-500 text-slate-950'
                        : 'text-slate-400 hover:bg-slate-800 hover:text-slate-100',
                    )}
                  >
                    {pg}
                  </button>
                );
              })}
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className={paginationBtn}
              >
                <ChevronRight size={15} />
              </button>
            </div>
          </div>
        )}
      </div>

      {showCreate && <CreateProductModal onClose={() => setShowCreate(false)} />}
    </div>
  );
}
