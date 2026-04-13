import { create } from 'zustand';
export const useAuthStore = create((set) => ({
    token: localStorage.getItem('auth_token'),
    user: null,
    setToken: (token) => {
        if (token) {
            localStorage.setItem('auth_token', token);
        }
        else {
            localStorage.removeItem('auth_token');
        }
        set({ token });
    },
    setUser: (user) => set({ user }),
    logout: () => {
        localStorage.removeItem('auth_token');
        set({ token: null, user: null });
    },
}));
//# sourceMappingURL=auth.js.map