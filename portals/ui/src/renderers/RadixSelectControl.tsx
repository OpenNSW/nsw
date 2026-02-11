import type { ControlProps } from '@jsonforms/core';
import { isEnumControl, rankWith } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { Box, Text, Select } from '@radix-ui/themes';

export const RadixSelectControlRenderer = (props: ControlProps) => {
    const { data, handleChange, path, label, required, schema, errors, enabled } = props;
    const options = schema.enum || [];

    return (
        <Box mb="4">
            <Text as="label" size="2" weight="bold" mb="2" className="block">
                {label}{required ? <Text color="red"> *</Text> : ''}
            </Text>
            <Box>
                <Select.Root
                    value={data || undefined}
                    onValueChange={(value) => handleChange(path, value)}
                    size="3"
                    disabled={!enabled}
                >
                    <Select.Trigger placeholder="Select..." className="w-full" />
                    <Select.Content>
                        {options.map((opt: string) => (
                            <Select.Item key={opt} value={opt}>
                                {opt}
                            </Select.Item>
                        ))}
                    </Select.Content>
                </Select.Root>
            </Box>
            {errors && errors.length > 0 && (
                <Text color="red" size="1" mt="1" className="block">
                    {errors}
                </Text>
            )}
        </Box>
    );
};

export const RadixSelectControlTester = rankWith(5, isEnumControl);
export const RadixSelectControl = withJsonFormsControlProps(RadixSelectControlRenderer);
