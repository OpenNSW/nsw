import { type ControlProps, isEnumControl, type RankedTester, rankWith, isOneOfControl, or } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { Select, Text, Flex, Box } from '@radix-ui/themes';


export const SelectControl = ({ data, handleChange, path, label, required, errors, schema, uischema, enabled }: ControlProps) => {
    const isValid = errors.length === 0;

    // Derive options
    let options: { value: string; label: string }[] = [];

    if (schema.enum) {
        options = schema.enum.map(e => ({ value: String(e), label: String(e) }));
    } else if (schema.oneOf) {
        options = schema.oneOf.map(o => ({
            value: String(o.const),
            label: o.title || String(o.const)
        }));
    }

    const value = data !== undefined ? String(data) : "";

    return (
        <Box mb="4">
            <Flex direction="column" gap="1">
                <Text as="label" size="2" weight="bold" htmlFor={path}>
                    {label} {required && <Text color="red">*</Text>}
                </Text>
                <Select.Root
                    value={value}
                    onValueChange={(val) => handleChange(path, val)}
                    disabled={!enabled}
                >
                    <Select.Trigger
                        placeholder={uischema.options?.placeholder || "Select an option"}
                        color={!isValid ? 'red' : undefined}
                        id={path}
                    />
                    <Select.Content>
                        {options.map(opt => (
                            <Select.Item key={opt.value} value={opt.value}>{opt.label}</Select.Item>
                        ))}
                    </Select.Content>
                </Select.Root>
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

export const SelectControlTester: RankedTester = rankWith(
    2,
    or(isEnumControl, isOneOfControl)
);

export default withJsonFormsControlProps(SelectControl);
