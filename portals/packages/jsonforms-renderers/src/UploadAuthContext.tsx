import { createContext, useContext, type ReactNode } from 'react';

export type GetAuthHeaders = () => Promise<HeadersInit>;

/**
 * Value provided by the host app for secure upload/download (Option A: renderer uses
 * host callbacks only, no direct network in the renderer). The host must wrap any tree
 * that uses FileControl with UploadAuthProvider and supply getAuthHeaders and getDownloadUrl
 * so the host's upload service is the single source of truth for auth and URLs.
 */
export type UploadAuthValue = {
  getAuthHeaders: GetAuthHeaders;
  /** Host's implementation for getting a download URL; FileControl calls this only (no fetch in renderer). */
  getDownloadUrl: (key: string) => Promise<string>;
  /** When provided, FileControl uploads via API and stores key; otherwise stores data URL. */
  uploadFile?: (file: File) => Promise<{ key: string; name: string }>;
};

const UploadAuthContext = createContext<UploadAuthValue | null>(null);

export function UploadAuthProvider({
  getAuthHeaders,
  getDownloadUrl,
  uploadFile,
  children,
}: {
  getAuthHeaders: GetAuthHeaders;
  getDownloadUrl: (key: string) => Promise<string>;
  uploadFile?: (file: File) => Promise<{ key: string; name: string }>;
  children: ReactNode;
}) {
  const value: UploadAuthValue = { getAuthHeaders, getDownloadUrl, uploadFile };
  return (
    <UploadAuthContext.Provider value={value}>
      {children}
    </UploadAuthContext.Provider>
  );
}

export function useUploadAuth(): UploadAuthValue | null {
  return useContext(UploadAuthContext);
}
