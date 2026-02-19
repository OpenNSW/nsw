import { type LayoutProps, type RankedTester, rankWith, uiTypeIs, type Layout, type GroupLayout, type Categorization as CategorizationInterface, type Category as CategoryInterface } from '@jsonforms/core';
import { JsonFormsDispatch } from '@jsonforms/react';
import { Box, Flex, Tabs, Card, Heading } from '@radix-ui/themes';
import { useState } from 'react';

// Vertical Layout
export const VerticalLayoutRenderer = ({ uischema, schema, path, renderers, cells, enabled }: LayoutProps) => {
    const layout = uischema as Layout;
    const elements = layout.elements;

    return (
        <Flex direction="column" gap="4">
            {elements.map((element, index) => (
                <JsonFormsDispatch
                    key={`${path}-${index}`}
                    uischema={element}
                    schema={schema}
                    path={path}
                    renderers={renderers}
                    cells={cells}
                    enabled={enabled}
                />
            ))}
        </Flex>
    );
};

export const VerticalLayoutTester: RankedTester = rankWith(1, uiTypeIs('VerticalLayout'));

// Horizontal Layout
export const HorizontalLayoutRenderer = ({ uischema, schema, path, renderers, cells, enabled }: LayoutProps) => {
    const layout = uischema as Layout;
    const elements = layout.elements;

    return (
        <Flex direction="row" gap="4">
            {elements.map((element, index) => (
                <Box key={`${path}-${index}`} style={{ flex: 1 }}>
                    <JsonFormsDispatch
                        uischema={element}
                        schema={schema}
                        path={path}
                        renderers={renderers}
                        cells={cells}
                        enabled={enabled}
                    />
                </Box>
            ))}
        </Flex>
    );
};

export const HorizontalLayoutTester: RankedTester = rankWith(1, uiTypeIs('HorizontalLayout'));

// Group Layout
export const GroupLayoutRenderer = ({ uischema, schema, path, renderers, cells, enabled }: LayoutProps) => {
    const group = uischema as GroupLayout;
    const elements = group.elements;

    return (
        <Card mb="4">
            {group.label && (
                <Heading size="4" mb="4">{group.label}</Heading>
            )}
            <Flex direction="column" gap="4">
                {elements.map((element, index) => (
                    <JsonFormsDispatch
                        key={`${path}-${index}`}
                        uischema={element}
                        schema={schema}
                        path={path}
                        renderers={renderers}
                        cells={cells}
                        enabled={enabled}
                    />
                ))}
            </Flex>
        </Card>
    );
};

export const GroupLayoutTester: RankedTester = rankWith(1, uiTypeIs('Group'));

// Categorization Layout (Tabs)
export interface CategorizationLayoutProps extends LayoutProps {
    data: any;
}

export const CategorizationLayoutRenderer = ({ uischema, schema, path, renderers, cells, enabled }: CategorizationLayoutProps) => {
    const categorization = uischema as CategorizationInterface;
    const categories = categorization.elements.filter((e): e is CategoryInterface => e.type === 'Category');
    const [activeTab, setActiveTab] = useState(categories[0]?.label || '');

    if (!categories.length) return null;

    // Ensure activeTab is valid (might not be if elements change)
    const currentTab = categories.find(c => c.label === activeTab) ? activeTab : categories[0].label;

    return (
        <Tabs.Root value={currentTab} onValueChange={setActiveTab}>
            <Tabs.List>
                {categories.map((category, index) => (
                    <Tabs.Trigger key={`${path}-${index}`} value={category.label}>
                        {category.label}
                    </Tabs.Trigger>
                ))}
            </Tabs.List>
            <Box pt="3">
                {categories.map((category, index) => (
                    <Tabs.Content key={`${path}-${index}`} value={category.label}>
                        <Flex direction="column" gap="4">
                            {category.elements.map((element, elemIndex) => (
                                <JsonFormsDispatch
                                    key={`${path}-${index}-${elemIndex}`}
                                    uischema={element}
                                    schema={schema}
                                    path={path}
                                    renderers={renderers}
                                    cells={cells}
                                    enabled={enabled}
                                />
                            ))}
                        </Flex>
                    </Tabs.Content>
                ))}
            </Box>
        </Tabs.Root>
    );
}

export const CategorizationLayoutTester: RankedTester = rankWith(1, uiTypeIs('Categorization'));
