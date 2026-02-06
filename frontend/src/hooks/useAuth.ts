import { useQuery, useQueryClient } from '@tanstack/react-query'
import { getMe, logout as apiLogout } from '@/api/auth'
import type { User } from '@/types/api'

export function useAuth() {
  const queryClient = useQueryClient()

  const {
    data: user,
    isLoading,
    error,
    refetch,
  } = useQuery<User>({
    queryKey: ['auth', 'me'],
    queryFn: getMe,
    retry: false,
    staleTime: 1000 * 60 * 5, // 5 minutes
  })

  const isAuthenticated = !!user && !error

  const logout = async () => {
    try {
      await apiLogout()
    } finally {
      queryClient.clear()
      window.location.href = '/login'
    }
  }

  return {
    user,
    isLoading,
    isAuthenticated,
    error,
    logout,
    refetch,
  }
}
