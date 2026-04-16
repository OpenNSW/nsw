import { type ControlProps, isEnumControl, isOneOfControl, or, type RankedTester } from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { Box, Flex, Text, RadioGroup } from '@radix-ui/themes';

export const RadioControl = ({ data, handleChange, path, label, required, errors, schema, enabled }: ControlProps) => {
    const isValid = errors.length === 0;

    let options: { value: any; label: string }[] = [];

    if (schema.enum) {
        options = schema.enum.map((e) => ({ value: e, label: String(e) }));
    } else if (schema.oneOf) {
        options = schema.oneOf.map((o: any) => ({
            value: o.const,
            label: o.title || String(o.const),
        }));
    }

    const value = data;

    return (
        <Box mb="4">
            <Flex direction="column" gap="1">
                <Text as="div" size="2" weight="bold">
                    {label} {required && <Text color="red">*</Text>}
                </Text>

                <RadioGroup.Root
                    value={value !== undefined ? String(value) : ''}
                    onValueChange={(val) => {
                        const selected = options.find((o) => String(o.value) === val);
                        if (selected) {
                            handleChange(path, selected.value);
                        }
                    }}
                    disabled={!enabled}
                >
                    <Flex
                        direction="row"
                        style={{
                            flexWrap: 'wrap',
                            columnGap: 'clamp(14px, 3vw, 36px)',
                            rowGap: 'clamp(8px, 1.6vw, 14px)',
                        }}
                    >
                        {options.map((opt) => (
                            <Flex key={String(opt.value)} align="center" gap="2" style={{ minWidth: 'max-content' }}>
                                <RadioGroup.Item value={String(opt.value)} />
                                <Text size="2" color={enabled ? undefined : 'gray'}>{opt.label}</Text>
                            </Flex>
                        ))}
                    </Flex>
                </RadioGroup.Root>

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

export const RadioControlTester: RankedTester = (uischema, schema, context) => {
    const isEnumLikeControl = or(isEnumControl, isOneOfControl)(uischema, schema, context);
    if (!isEnumLikeControl) {
        return -1;
    }

    return uischema?.options?.format === 'radio' ? 3 : -1;
};

export default withJsonFormsControlProps(RadioControl);