import { createContext, useContext, type ReactNode } from 'react';

export type GetAuthHeaders = () => Promise<HeadersInit>;

const UploadAuthContext = createContext<GetAuthHeaders | null>(null);

export function UploadAuthProvider({
  getAuthHeaders,
  children,
}: {
  getAuthHeaders: GetAuthHeaders;
  children: ReactNode;
}) {
  return (
    <UploadAuthContext.Provider value={getAuthHeaders}>
      {children}
    </UploadAuthContext.Provider>
  );
}

export function useUploadAuth(): GetAuthHeaders | null {
  return useContext(UploadAuthContext);
}
