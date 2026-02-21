import { type LabelElement, type RankedTester, rankWith, uiTypeIs } from '@jsonforms/core';
import { withJsonFormsLayoutProps } from '@jsonforms/react';
import { Heading, Box } from '@radix-ui/themes';

export const LabelRenderer = ({ uischema, visible }: any) => {
    const labelElement = uischema as LabelElement;

    if (visible === false) {
        return null;
    }

    return (
        <Box mb="4">
            <Heading size="4">{labelElement.text}</Heading>
        </Box>
    );
};

export const LabelTester: RankedTester = rankWith(1, uiTypeIs('Label'));

export default withJsonFormsLayoutProps(LabelRenderer);
