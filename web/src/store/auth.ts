import { create } from 'zustand'

interface AuthStore {
  token: string | null
  user: { id: string; name: string } | null
  setToken: (token: string | null) => void
  setUser: (user: { id: string; name: string } | null) => void
  logout: () => void
}

export const useAuthStore = create<AuthStore>((set) => ({
  token: localStorage.getItem('auth_token'),
  user: null,
  setToken: (token) => {
    if (token) {
      localStorage.setItem('auth_token', token)
    } else {
      localStorage.removeItem('auth_token')
    }
    set({ token })
  },
  setUser: (user) => set({ user }),
  logout: () => {
    localStorage.removeItem('auth_token')
    set({ token: null, user: null })
  },
}))
