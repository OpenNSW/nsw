import { JsonForms } from '@jsonforms/react';
import { radixRenderers, radixCells } from '../../renderers';
import { useState, useEffect } from 'react';
import { schemaToZod } from './schemaToZod';
import { Button, Callout } from '@radix-ui/themes';
import { z } from 'zod';
import { FileControl, FileControlTester } from '@lsf/ui';
import { ExclamationTriangleIcon } from '@radix-ui/react-icons';

const renderers = [
  ...radixRenderers,
  { tester: FileControlTester, renderer: FileControl },
];

const i18n = {
  translate: (id: string, defaultMessage: string | undefined, _values: any): string => {
    if (id === 'error.required') {
      return '';
    }
    return defaultMessage ?? '';
  },
};



import { Spinner } from '@radix-ui/themes';

interface JsonFormProps {
  schema: any;
  uiSchema: any;
  data: any;
  onSubmit: (data: any) => void;
  submitLabel?: string;
  showAutoFillButton?: boolean;
  autoFillLabel?: string;
  readonly?: boolean;
  submitting?: boolean;
  onChange?: (data: any) => void;
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
  const [data, setData] = useState(initialData || {});
  const [errors, setErrors] = useState<any[]>([]);

  // State for inline validation error message (replaces alert)
  const [validationError, setValidationError] = useState<string | null>(null);

  useEffect(() => {
    setData(initialData || {});
  }, [initialData]);

  // Auto-clear validation error when user makes changes
  useEffect(() => {
    if (validationError) {
      setValidationError(null);
    }
  }, [data]);

  const handleSubmit = () => {
    if (submitting) return; // Prevent double submit
    setValidationError(null);

    // 1. Client-side Zod validation
    try {
      if (schema) {
        const zodSchema = schemaToZod(schema);
        zodSchema.parse(data);
      }
    } catch (e) {
      if (e instanceof z.ZodError) {
        console.error('Validation errors:', e.errors);
        setValidationError("Please fill in all required fields and correct any errors.");
      }
      return;
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
        renderers={renderers}
        cells={radixCells}
        i18n={i18n}
        readonly={readonly || submitting}
        onChange={({ data, errors }) => {
          setData(data);
          setErrors(errors || []);
          if (onChange) {
            onChange(data);
          }
        }}
      />

      {/* Hide Submit button if read-only */}
      {!readonly && (
        <div style={{ marginTop: '2rem', display: 'flex', justifyContent: 'flex-end' }}>
          <Button
            size="3"
            onClick={handleSubmit}
            // We disable strictly on internal errors, but allow click to trigger Zod check
            disabled={errors.length > 0 || submitting}
            style={{ cursor: (errors.length > 0 || submitting) ? 'not-allowed' : 'pointer' }}
          >
            {submitting && <Spinner loading />}
            {submitLabel}
          </Button>
        </div>
      )}
    </div>
  );
};