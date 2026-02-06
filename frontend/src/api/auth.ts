import { api } from './client'
import type { User } from '@/types/api'

export const getMe = () => api.get('auth/me').json<User>()

export const logout = () => api.post('auth/logout')
