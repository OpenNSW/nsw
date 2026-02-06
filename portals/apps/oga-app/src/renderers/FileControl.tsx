import { withJsonFormsControlProps } from '@jsonforms/react';
import type { ControlElement, JsonSchema } from '@jsonforms/core';
import { Card, Flex, Text, Box, IconButton } from '@radix-ui/themes';
import { UploadIcon, FileTextIcon, Cross2Icon, CheckCircledIcon, ExclamationTriangleIcon } from '@radix-ui/react-icons';
import { useState, useRef, type ChangeEvent, type DragEvent } from 'react';

interface FileControlProps {
    data: string | null;
    handleChange(path: string, value: string | null): void;
    path: string;
    label: string;
    required?: boolean;
    uischema?: ControlElement;
    schema?: JsonSchema;
    enabled?: boolean;
}


const FileControl = ({ data, handleChange, path, label, required, uischema, enabled }: FileControlProps) => {
    const [dragActive, setDragActive] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [fileName, setFileName] = useState<string | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    // Get options from UI schema (or default)
    const options = uischema?.options || {};
    const maxSizeBytes = options.maxSize ? options.maxSize : 5 * 1024 * 1024; // Default 5MB
    const accept = options.accept || 'image/*,application/pdf';

    // Extract metadata from data URI if present
    // If we have a local fileName state, use it. Otherwise fall back to generic.
    const getDisplayText = () => {
        if (fileName) return fileName;
        if (!data) return null;
        return 'Uploaded File';
    };

    const processFile = (file: File) => {
        if (file.size > maxSizeBytes) {
            const sizeMB = (maxSizeBytes / (1024 * 1024)).toFixed(0);
            setError(`File size exceeds ${sizeMB}MB limit.`);
            return;
        }

        const acceptedTypes = accept.split(',').map((t: string) => t.trim());
        const isFileTypeAccepted = acceptedTypes.some((type: string) => {
            if (type.endsWith('/*')) {
                return file.type.startsWith(type.slice(0, -1));
            }
            return file.type === type;
        });

        if (!isFileTypeAccepted) {
            setError(`Invalid file type. Accepted types: ${accept}`);
            return;
        }

        const reader = new FileReader();
        reader.onload = () => {
            const result = reader.result as string;
            handleChange(path, result);
            setFileName(file.name);
            setError(null);
        };
        reader.onerror = () => {
            setError('Failed to read file');
        };
        reader.readAsDataURL(file);
    };

    const handleDrag = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        e.stopPropagation();
        if (!enabled || data) return;

        if (e.type === 'dragenter' || e.type === 'dragover') {
            setDragActive(true);
        } else if (e.type === 'dragleave') {
            setDragActive(false);
        }
    };

    const handleDrop = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(false);
        if (!enabled || data) return;

        if (e.dataTransfer.files && e.dataTransfer.files[0]) {
            processFile(e.dataTransfer.files[0]);
        }
    };

    const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
        if (e.target.files && e.target.files[0]) {
            processFile(e.target.files[0]);
        }
    };

    const handleRemove = () => {
        handleChange(path, null);
        setFileName(null);
        setError(null);
        if (inputRef.current) {
            inputRef.current.value = '';
        }
    };

    // Accessibility handler for keyboard
    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
        if (enabled && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault();
            inputRef.current?.click();
        }
    };



    return (
        <Box mb="4">
            <Text as="label" size="2" weight="bold" mb="1" className="block">
                {label} {required && '*'}
            </Text>

            {data ? (
                <Card size="2" variant="surface" className="relative group">
                    <Flex align="center" gap="3">
                        <Box className="bg-blue-100 p-2 rounded text-blue-600">
                            <FileTextIcon width="20" height="20" />
                        </Box>
                        <Box style={{ flex: 1, overflow: 'hidden' }}>
                            <Text size="2" weight="bold" className="block truncate">
                                {getDisplayText()}
                            </Text>
                            <Text size="1" color="gray">
                                Ready to submit
                            </Text>
                        </Box>
                        <Flex align="center" gap="2">
                            <CheckCircledIcon className="text-green-600 w-5 h-5" />
                            {enabled && (
                                <IconButton
                                    variant="ghost"
                                    color="gray"
                                    onClick={handleRemove}
                                    className="hover:text-red-600 transition-colors"
                                >
                                    <Cross2Icon />
                                </IconButton>
                            )}
                        </Flex>
                    </Flex>
                </Card>
            ) : (
                <div
                    className={`
            border-2 border-dashed rounded-lg p-6 text-center transition-all duration-200 ease-in-out
            ${dragActive ? 'border-blue-500 bg-blue-50' : 'border-gray-300 hover:border-blue-400 hover:bg-gray-50'}
            ${error ? 'border-red-300 bg-red-50' : ''}
            ${!enabled ? 'opacity-60 cursor-not-allowed pointer-events-none' : 'cursor-pointer'}
          `}
                    onDragEnter={handleDrag}
                    onDragLeave={handleDrag}
                    onDragOver={handleDrag}
                    onDrop={handleDrop}
                    onClick={() => inputRef.current?.click()}
                    onKeyDown={handleKeyDown}
                    role="button"
                    tabIndex={!enabled ? -1 : 0}
                >
                    <input
                        ref={inputRef}
                        type="file"
                        className="hidden"
                        accept={accept}
                        onChange={handleInputChange}
                        disabled={!enabled}
                    />

                    <Flex direction="column" align="center" gap="2">
                        {error ? (
                            <>
                                <ExclamationTriangleIcon className="w-8 h-8 text-red-500" />
                                <Text size="2" color="red" weight="medium">
                                    {error}
                                </Text>
                                <Text size="1" color="gray">Click to try again</Text>
                            </>
                        ) : (
                            <>
                                <UploadIcon className="w-8 h-8 text-gray-400" />
                                <Text size="2" weight="medium">
                                    Click to upload or drag and drop
                                </Text>
                                <Text size="1" color="gray">
                                    Max {Math.round(maxSizeBytes / (1024 * 1024))}MB
                                </Text>
                            </>
                        )}
                    </Flex>
                </div>
            )}
        </Box>
    );
};

export default withJsonFormsControlProps(FileControl);
