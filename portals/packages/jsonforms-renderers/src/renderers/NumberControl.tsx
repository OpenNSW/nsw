import { type ControlProps, isNumberControl, type RankedTester, rankWith, isIntegerControl, or } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { TextField, Text, Flex, Box } from '@radix-ui/themes';


export const NumberControl = ({ data, handleChange, path, label, required, errors, uischema, schema, enabled }: ControlProps) => {
    const isValid = errors.length === 0;

    const handleNumberChange = (value: string) => {
        if (value === '') {
            handleChange(path, undefined);
            return;
        }
        const num = Number(value);
        if (!isNaN(num)) {
            handleChange(path, num);
        }
    };

    return (
        <Box mb="4">
            <Flex direction="column" gap="1">
                <Text as="label" size="2" weight="bold" htmlFor={path}>
                    {label} {required && <Text color="red">*</Text>}
                </Text>
                <TextField.Root
                    type="number"
                    value={data ?? ''}
                    onChange={(e) => handleNumberChange(e.target.value)}
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

export const NumberControlTester: RankedTester = rankWith(
    2,
    or(isNumberControl, isIntegerControl)
);

export default withJsonFormsControlProps(NumberControl);
