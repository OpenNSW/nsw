import { type ControlProps, isStringControl, type RankedTester, rankWith } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { TextField, Text, Flex, Box } from '@radix-ui/themes';


export const TextControl = ({ data, handleChange, path, label, required, errors, uischema, schema, enabled }: ControlProps) => {
    const isValid = errors.length === 0;

    return (
        <Box mb="4">
            <Flex direction="column" gap="1">
                <Text as="label" size="2" weight="bold" htmlFor={path}>
                    {label} {required && <Text color="red">*</Text>}
                </Text>
                <TextField.Root
                    value={data || ''}
                    onChange={(e) => handleChange(path, e.target.value)}
                    disabled={!enabled}
                    placeholder={uischema.options?.placeholder}
                    color={!isValid ? 'red' : undefined}
                    id={path}
                />
                {!isValid && errors !== "is a required property" && (
                    <Text color="red" size="1">
                        {errors}
                    </Text>
                )}
                {schema.description && (
                    <Text size="1" color="gray">
                        {schema.description}
                    </Text>
                )}
            </Flex>
        </Box>
    );
};

export const TextControlTester: RankedTester = rankWith(
    1,
    isStringControl
);

export default withJsonFormsControlProps(TextControl);
