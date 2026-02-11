import type { VerticalLayout, LayoutProps } from '@jsonforms/core';
import { rankWith, uiTypeIs } from '@jsonforms/core';
import { withJsonFormsLayoutProps, JsonFormsDispatch } from '@jsonforms/react';
import { Flex } from '@radix-ui/themes';

const RadixVerticalLayoutRenderer = ({
    schema,
    uischema,
    path,
    renderers,
    cells,
}: LayoutProps) => {
    const layout = uischema as VerticalLayout;

    return (
        <Flex direction="column" gap="4">
            {layout.elements.map((element, index) => (
                <JsonFormsDispatch
                    key={`${path}-${index}`}
                    uischema={element}
                    schema={schema}
                    path={path}
                    renderers={renderers}
                    cells={cells}
                />
            ))}
        </Flex>
    );
};

export const RadixVerticalLayoutTester = rankWith(3, uiTypeIs('VerticalLayout'));
export const RadixVerticalLayout = withJsonFormsLayoutProps(RadixVerticalLayoutRenderer);
