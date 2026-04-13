interface AuthStore {
    token: string | null;
    user: {
        id: string;
        name: string;
    } | null;
    setToken: (token: string | null) => void;
    setUser: (user: {
        id: string;
        name: string;
    } | null) => void;
    logout: () => void;
}
export declare const useAuthStore: import("zustand").UseBoundStore<import("zustand").StoreApi<AuthStore>>;
export {};
//# sourceMappingURL=auth.d.ts.map