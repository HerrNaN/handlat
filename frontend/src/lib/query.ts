import { QueryClient } from "@tanstack/query-core";

export const createQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      networkMode: "always",
      staleTime: 0,
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
    },
    mutations: {
      retry: false,
      networkMode: "always",
    },
  },
});

export const queryClient = createQueryClient();
