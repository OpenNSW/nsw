import { type ControlProps, isBooleanControl, type RankedTester, rankWith } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { Checkbox, Text, Flex, Box } from '@radix-ui/themes';


export const BooleanControl = ({ data, handleChange, path, label, required, errors, schema, enabled }: ControlProps) => {
    const isValid = errors.length === 0;

    return (
        <Box mb="4">
            <Flex gap="2" align="center">
                <Checkbox
                    checked={!!data}
                    onCheckedChange={(checked) => handleChange(path, checked === true)}
                    disabled={!enabled}
                    id={path}
                />
                <Text as="label" size="2" htmlFor={path}>
                    {label} {required && '*'}
                </Text>
            </Flex>
            {!isValid && (
                <Text color="red" size="1" className="block mt-1">
                    {errors}
                </Text>
            )}
            {schema.description && (
                <Text size="1" color="gray" className="block mt-1">
                    {schema.description}
                </Text>
            )}
        </Box>
    );
};

export const BooleanControlTester: RankedTester = rankWith(
    2,
    isBooleanControl
);

export default withJsonFormsControlProps(BooleanControl);
