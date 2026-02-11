import {
    type ControlProps,
    isBooleanControl,
    rankWith,
} from '@jsonforms/core';
import { withJsonFormsControlProps } from '@jsonforms/react';
import { Checkbox, Text, Flex } from '@radix-ui/themes';

const RadixBooleanControlRenderer = ({
    data,
    handleChange,
    path,
    label,
    required,
    enabled,
}: ControlProps) => {
    return (
        <Flex align="center" gap="2" mb="2">
            <Checkbox
                checked={!!data}
                onCheckedChange={(checked) => handleChange(path, checked === true)}
                disabled={!enabled}
            />
            <Text as="label" size="2">
                {label} {required && <Text color="red">*</Text>}
            </Text>
        </Flex>
    );
};

export const RadixBooleanControlTester = rankWith(1, isBooleanControl);
export const RadixBooleanControl = withJsonFormsControlProps(RadixBooleanControlRenderer);
