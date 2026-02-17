import type { FieldProps } from '../types';
import { FieldWrapper } from './FieldWrapper';
import { useState, useEffect } from 'react';
import { getFileUrl } from '../../../services/upload';

export function FileField({ control, value, error, touched, onChange, onBlur }: FieldProps) {
  const isReadonly = control.options?.readonly;
  const [displayName, setDisplayName] = useState<string>('');
  const [previewUrl, setPreviewUrl] = useState<string>('');

  useEffect(() => {
    if (value instanceof File) {
      setDisplayName(value.name);
      const objectUrl = URL.createObjectURL(value);
      setPreviewUrl(objectUrl);
      return () => URL.revokeObjectURL(objectUrl);
    }

    if (typeof value === 'string' && value) {
      const parts = value.split('/');
      setDisplayName(parts[parts.length - 1] || value);
      setPreviewUrl(getFileUrl(value));
      return undefined;
    }

    setDisplayName('');
    setPreviewUrl('');
    return undefined;
  }, [value]);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Cache the file in form state. Upload happens on submit.
    onChange(file);
    setDisplayName(file.name);
  };

  const showFileInfo = Boolean(displayName);
  const hasPreview = Boolean(previewUrl);

  return (
    <FieldWrapper control={control} error={error} touched={touched}>
      <div className="space-y-3">
        <div
          className={`
            rounded-lg border p-3 transition-colors
            ${touched && error ? 'border-red-500' : 'border-gray-200'}
            bg-white
          `}
        >
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <input
                type="file"
                name={control.name}
                onChange={handleFileChange}
                onBlur={onBlur}
                disabled={isReadonly}
                accept={control.options?.format && control.options.format !== 'file' ? `.${control.options.format}` : '*/*'}
                className={`
                  w-full sm:w-auto px-3 py-2 border rounded-md shadow-sm
                  focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
                  disabled:bg-gray-100 disabled:cursor-not-allowed
                  ${touched && error ? 'border-red-500' : 'border-gray-300'}
                  file:mr-4 file:py-2 file:px-4 file:rounded-md
                  file:border-0 file:text-sm file:font-semibold
                  file:bg-blue-50 file:text-blue-700
                  hover:file:bg-blue-100
                `}
              />
            </div>

            {hasPreview && (
              <a
                href={previewUrl}
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
              <span className="text-xs text-gray-500 ml-2">(uploads on submit)</span>
            </div>
          )}
        </div>
      </div>
    </FieldWrapper>
  );
}
