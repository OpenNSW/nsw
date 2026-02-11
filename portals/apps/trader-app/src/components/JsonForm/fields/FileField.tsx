import type { FieldProps } from '../types';
import { FieldWrapper } from './FieldWrapper';
import { useState, useEffect } from 'react';
import { uploadFile, getFileUrl } from '../../../services/upload';

export function FileField({ control, value, error, touched, onChange, onBlur }: FieldProps) {
  const isReadonly = control.options?.readonly;
  const [displayName, setDisplayName] = useState<string>('');
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string>('');

  useEffect(() => {
    // value is now the file key (string) returned from the server
    if (typeof value === 'string' && value) {
      // Extract filename from key or use the key itself
      const parts = value.split('/');
      setDisplayName(parts[parts.length - 1]);
    } else {
      setDisplayName('');
    }
  }, [value]);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    setUploadError('');
    
    try {
      // Upload file to server
      const metadata = await uploadFile(file);
      
      // Store the file key (reference) as the field value
      onChange(metadata.key);
      setDisplayName(file.name);
    } catch (err) {
      console.error('File upload failed:', err);
      setUploadError('Failed to upload file. Please try again.');
      // Clear the input
      e.target.value = '';
    } finally {
      setUploading(false);
    }
  };

  const fileKey = typeof value === 'string' ? value : '';
  const showFileInfo = displayName && !uploading && fileKey;

  return (
    <FieldWrapper control={control} error={error || uploadError} touched={touched}>
      <div className="space-y-3">
        <div
          className={`
            rounded-lg border p-3 transition-colors
            ${touched && (error || uploadError) ? 'border-red-500' : 'border-gray-200'}
            ${uploading ? 'bg-blue-50' : 'bg-white'}
          `}
        >
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <input
                type="file"
                name={control.name}
                onChange={handleFileChange}
                onBlur={onBlur}
                disabled={isReadonly || uploading}
                accept={control.options?.format && control.options.format !== 'file' ? `.${control.options.format}` : '*/*'}
                className={`
                  w-full sm:w-auto px-3 py-2 border rounded-md shadow-sm
                  focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
                  disabled:bg-gray-100 disabled:cursor-not-allowed
                  ${touched && (error || uploadError) ? 'border-red-500' : 'border-gray-300'}
                  file:mr-4 file:py-2 file:px-4 file:rounded-md
                  file:border-0 file:text-sm file:font-semibold
                  file:bg-blue-50 file:text-blue-700
                  hover:file:bg-blue-100
                `}
              />
              {uploading && (
                <span className="text-xs font-medium text-blue-700 bg-blue-100 px-2 py-1 rounded-full">
                  Uploading...
                </span>
              )}
            </div>

            {showFileInfo && (
              <a
                href={getFileUrl(fileKey)}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center text-sm font-medium text-blue-700 bg-blue-50 px-3 py-1.5 rounded-md border border-blue-200 hover:bg-blue-100"
              >
                View
              </a>
            )}
          </div>

          {showFileInfo && (
            <div className="mt-3 text-sm text-gray-700">
              Selected file: <span className="font-medium">{displayName}</span>
            </div>
          )}
        </div>
      </div>
    </FieldWrapper>
  );
}
