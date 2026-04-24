import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Eye, EyeOff, Loader2, Lock, Mail } from 'lucide-react';
import toast from 'react-hot-toast';
import { useAuthStore } from '@/store/auth.store';
import { cn } from '@/utils/cn';

const loginSchema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(1, 'Password is required'),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const [showPassword, setShowPassword] = useState(false);
  const navigate = useNavigate();
  const { login, isLoading } = useAuthStore();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (data: LoginForm) => {
    try {
      await login(data);
      navigate('/dashboard');
    } catch {
      toast.error('Invalid email or password');
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center p-4">
      {/* Background grid */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#0f172a_1px,transparent_1px),linear-gradient(to_bottom,#0f172a_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_110%)]" />

      <div className="relative w-full max-w-md">
        {/* Glow */}
        <div className="absolute -top-32 left-1/2 -translate-x-1/2 w-80 h-80 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none" />

        <div className="relative bg-slate-900 border border-slate-800 rounded-2xl p-8 shadow-2xl">
          {/* Header */}
          <div className="mb-8 text-center">
            <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-cyan-500/10 border border-cyan-500/20 mb-4">
              <Lock size={20} className="text-cyan-400" />
            </div>
            <h1 className="text-2xl font-bold text-slate-100 tracking-tight">
              Welcome back
            </h1>
            <p className="mt-1 text-sm text-slate-400">
              Sign in to Microservice Platform
            </p>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
            {/* Email */}
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1.5 uppercase tracking-wider">
                Email
              </label>
              <div className="relative">
                <Mail
                  size={16}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
                />
                <input
                  {...register('email')}
                  type="email"
                  autoComplete="email"
                  placeholder="admin@example.com"
                  className={cn(
                    'w-full bg-slate-800 border rounded-lg pl-10 pr-4 py-2.5 text-sm',
                    'text-slate-100 placeholder-slate-500',
                    'focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all',
                    errors.email
                      ? 'border-rose-500/60'
                      : 'border-slate-700 hover:border-slate-600',
                  )}
                />
              </div>
              {errors.email && (
                <p className="mt-1.5 text-xs text-rose-400">{errors.email.message}</p>
              )}
            </div>

            {/* Password */}
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1.5 uppercase tracking-wider">
                Password
              </label>
              <div className="relative">
                <Lock
                  size={16}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
                />
                <input
                  {...register('password')}
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  placeholder="••••••••"
                  className={cn(
                    'w-full bg-slate-800 border rounded-lg pl-10 pr-10 py-2.5 text-sm',
                    'text-slate-100 placeholder-slate-500',
                    'focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all',
                    errors.password
                      ? 'border-rose-500/60'
                      : 'border-slate-700 hover:border-slate-600',
                  )}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((p) => !p)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 transition-colors"
                >
                  {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              </div>
              {errors.password && (
                <p className="mt-1.5 text-xs text-rose-400">{errors.password.message}</p>
              )}
            </div>

            {/* Submit */}
            <button
              type="submit"
              disabled={isLoading}
              className={cn(
                'w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2.5',
                'bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-semibold text-sm',
                'transition-all focus:outline-none focus:ring-2 focus:ring-cyan-500/50',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              {isLoading ? (
                <>
                  <Loader2 size={16} className="animate-spin" />
                  Signing in…
                </>
              ) : (
                'Sign in'
              )}
            </button>
          </form>

          {/* Footer */}
          <p className="mt-6 text-center text-xs text-slate-500">
            Microservice Platform v{import.meta.env.VITE_APP_VERSION}
          </p>
        </div>
      </div>
    </div>
  );
}
