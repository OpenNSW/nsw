import { createContext, useContext, type ReactNode } from 'react';

/**
 * Contract for upload: applications provide the implementation; 
 * the renderer only calls these callbacks
 */
export interface UploadResponse {
  key: string;
  name?: string;
}

export type UploadHandler = (file: File) => Promise<UploadResponse>;
export interface DownloadUrlResult {
  url: string;
  expiresAt: number;
}

export type GetDownloadUrl = (key: string) => Promise<DownloadUrlResult>;

export interface UploadContextValue {
  onUpload?: UploadHandler;
  getDownloadUrl?: GetDownloadUrl;
}

const UploadContext = createContext<UploadContextValue | null>(null);

export function UploadProvider({
  children,
  onUpload,
  getDownloadUrl,
}: {
  children: ReactNode;
  onUpload?: UploadHandler;
  getDownloadUrl?: GetDownloadUrl;
}) {
  const value: UploadContextValue = { onUpload, getDownloadUrl };
  return (
    <UploadContext.Provider value={value}>
      {children}
    </UploadContext.Provider>
  );
}

export function useUpload(): UploadContextValue | null {
  return useContext(UploadContext);
}
