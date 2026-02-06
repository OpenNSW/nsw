import { type ControlProps, isStringControl, rankWith } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { TextField, Text, Flex } from '@radix-ui/themes';

const RadixTextControlRenderer = ({ data, handleChange, path, label, errors, required, enabled }: ControlProps) => {
    const isValid = errors.length === 0;

    return (
        <Flex direction="column" gap="1" mb="2">
            <Text as="label" size="2" weight="bold">
                {label} {required && <Text color="red">*</Text>}
            </Text>
            <TextField.Root
                value={data || ''}
                onChange={(e) => handleChange(path, e.target.value)}
                color={!isValid ? 'red' : undefined}
                disabled={!enabled}
            />
            {!isValid && (
                <Text color="red" size="1">
                    {errors}
                </Text>
            )}
        </Flex>
    );
};

export const RadixTextControlTester = rankWith(1, isStringControl);
export const RadixTextControl = withJsonFormsControlProps(RadixTextControlRenderer);
