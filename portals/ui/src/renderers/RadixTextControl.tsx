import type { ControlProps } from '@jsonforms/core';
import { isStringControl, rankWith } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { Box, Text, TextField } from '@radix-ui/themes';

export const RadixTextControlRenderer = (props: ControlProps) => {
    const { data, handleChange, path, label, required, description, errors, enabled } = props;
    return (
        <Box mb="4">
            <Text as="label" size="2" weight="bold" mb="2" className="block">
                {label}{required ? <Text color="red"> *</Text> : ''}
            </Text>
            <TextField.Root
                value={data || ''}
                onChange={(e) => handleChange(path, e.target.value)}
                placeholder={description}
                size="3"
                variant="surface"
                disabled={!enabled}
            />
            {errors && errors.length > 0 && (
                <Text color="red" size="1" mt="1" className="block">
                    {errors}
                </Text>
            )}
        </Box>
    );
};

export const RadixTextControlTester = rankWith(3, isStringControl);
export const RadixTextControl = withJsonFormsControlProps(RadixTextControlRenderer);
