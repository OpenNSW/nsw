import type { ControlProps } from '@jsonforms/core';
import { isBooleanControl, rankWith } from '@jsonforms/core';
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
        <Flex align="center" gap="2" mb="4">
            <Checkbox
                checked={!!data}
                onCheckedChange={(checked) => handleChange(path, checked === true)}
                disabled={!enabled}
                size="3"
            />
            <Text as="label" size="2" weight="bold">
                {label}{required ? ' *' : ''}
            </Text>
        </Flex>
    );
};

export const RadixBooleanControlTester = rankWith(3, isBooleanControl);
export const RadixBooleanControl = withJsonFormsControlProps(RadixBooleanControlRenderer);
