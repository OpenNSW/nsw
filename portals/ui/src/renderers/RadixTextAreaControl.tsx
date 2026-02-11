import type { ControlProps } from '@jsonforms/core';
import { isStringControl, rankWith, and, optionIs } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { Box, Text, TextArea } from '@radix-ui/themes';

export const RadixTextAreaControlRenderer = (props: ControlProps) => {
    const { data, handleChange, path, label, required, description, errors, enabled } = props;
    return (
        <Box mb="4">
            <Text as="label" size="2" weight="bold" mb="2" className="block">
                {label}{required ? <Text color="red"> *</Text> : ''}
            </Text>
            <TextArea
                value={data || ''}
                onChange={(e) => handleChange(path, e.target.value)}
                placeholder={description}
                size="3"
                rows={4}
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

export const RadixTextAreaControlTester = rankWith(5, and(isStringControl, optionIs('multi', true)));
export const RadixTextAreaControl = withJsonFormsControlProps(RadixTextAreaControlRenderer);
