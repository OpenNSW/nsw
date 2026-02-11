import { JsonForms } from '@jsonforms/react';
import type { JsonSchema, UISchemaElement } from '@jsonforms/core';
import { radixRenderers, radixCells } from '../../renderers';
import { useState, useEffect, useMemo } from 'react';
import { schemaToZod } from './schemaToZod';
import { Button, Callout, Spinner } from '@radix-ui/themes';
import { z } from 'zod';
import { ExclamationTriangleIcon } from '@radix-ui/react-icons';

const i18n = {
    translate: (id: string, defaultMessage: string | undefined, _values: unknown): string => {
        if (id === 'error.required') {
            return 'is required';
        }
        return defaultMessage ?? '';
    },
};

export interface JsonFormProps {
    schema: JsonSchema;
    uiSchema?: UISchemaElement;
    data: Record<string, unknown>;
    onSubmit: (data: Record<string, unknown>) => void | Promise<void>;
    submitLabel?: string;
    showAutoFillButton?: boolean;
    autoFillLabel?: string;
    readonly?: boolean;
    submitting?: boolean;
    onChange?: (data: Record<string, unknown>) => void | Promise<void>;
}

export const JsonForm = ({
    schema,
    uiSchema,
    data: initialData,
    onSubmit,
    submitLabel = 'Submit',
    showAutoFillButton = false,
    autoFillLabel = 'Auto-Fill',
    readonly = false,
    submitting = false,
    onChange,
}: JsonFormProps) => {
    const [data, setData] = useState<Record<string, unknown>>(initialData || {});
    const [errors, setErrors] = useState<unknown[]>([]);

    // State for inline validation error message
    const [validationError, setValidationError] = useState<string | null>(null);

    useEffect(() => {
        setData(initialData || {});
    }, [initialData]);

    // Clear validation error when user interacts if it's currently showing
    useEffect(() => {
        if (validationError && data !== initialData) {
            setValidationError(null);
        }
    }, [data, validationError, initialData]);

    const zodSchema = useMemo(() => {
        if (!schema) return null;
        try {
            return schemaToZod(schema);
        } catch (e) {
            console.error('Failed to generate Zod schema:', e);
            return null;
        }
    }, [schema]);

    const handleSubmit = () => {
        if (submitting) return; // Prevent double submit
        setValidationError(null);

        // 1. Client-side Zod validation
        if (zodSchema) {
            try {
                zodSchema.parse(data);
            } catch (e) {
                if (e instanceof z.ZodError) {
                    console.error('Validation errors:', e.errors);
                    setValidationError("Please fill in all required fields and correct any errors.");
                }
                return;
            }
        }

        // 2. JsonForms Internal Validation Check
        if (errors.length === 0) {
            onSubmit(data);
        } else {
            setValidationError("Please fill in all required fields and correct any errors.");
        }
    };

    const handleAutoFill = () => {
        console.log("Autofill not implemented yet");
    };

    return (
        <div className="json-form-container">
            {/* Hide AutoFill if read-only */}
            {showAutoFillButton && !readonly && (
                <div style={{ marginBottom: '1rem', textAlign: 'right' }}>
                    <Button variant="soft" onClick={handleAutoFill} disabled={submitting}>{autoFillLabel}</Button>
                </div>
            )}

            {/* Inline Error Summary */}
            {validationError && (
                <Callout.Root color="red" mb="4">
                    <Callout.Icon>
                        <ExclamationTriangleIcon />
                    </Callout.Icon>
                    <Callout.Text>
                        {validationError}
                    </Callout.Text>
                </Callout.Root>
            )}

            <JsonForms
                schema={schema}
                uischema={uiSchema}
                data={data}
                renderers={radixRenderers}
                cells={radixCells}
                i18n={i18n}
                readonly={readonly || submitting}
                onChange={({ data, errors }) => {
                    setData(data as Record<string, unknown>);
                    setErrors(errors || []);
                    if (onChange) {
                        onChange(data as Record<string, unknown>);
                    }
                }}
            />

            {/* Hide Submit button if read-only */}
            {!readonly && (
                <div style={{ marginTop: '2rem', display: 'flex', justifyContent: 'flex-end' }}>
                    <Button
                        size="3"
                        onClick={handleSubmit}
                        disabled={submitting}
                        style={{ cursor: submitting ? 'not-allowed' : 'pointer' }}
                    >
                        {submitting && <Spinner />}
                        {submitLabel}
                    </Button>
                </div>
            )}
        </div>
    );
};
