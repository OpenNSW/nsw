import { createDefaultValue, type ArrayControlProps } from '@jsonforms/core';
import { withJsonFormsArrayControlProps, JsonFormsDispatch } from '@jsonforms/react';
import { Card, Button, Flex, Text, Box } from '@radix-ui/themes';
import { PlusIcon, TrashIcon } from '@radix-ui/react-icons';

export const ArrayControl = ({
    data,
    path,
    schema,
    uischema,
    enabled,
    visible,
    addItem,
    removeItems,
    rootSchema,
}: ArrayControlProps) => {
    const itemsSchema = schema.items;

    if (visible === false) {
        return null;
    }
    if (!itemsSchema || typeof itemsSchema !== 'object' || Array.isArray(itemsSchema)) {
        return null; // Or render a more descriptive error message
    }

    // After the guard, we know itemsSchema is a valid single JsonSchema object
    const validItemsSchema = itemsSchema as import('@jsonforms/core').JsonSchema;

    const items = Array.isArray(data) ? data : [];
    const title = schema.title || 'Array Items';

    const handleAddItem = () => {
        const newItem = createDefaultValue(validItemsSchema, rootSchema);
        if (addItem) addItem(path, newItem)();
    };

    const handleRemoveItem = (indexToRemove: number) => {
        if (removeItems) {
            const removeFunc = removeItems(path, [indexToRemove]);
            if (removeFunc) removeFunc();
        }
    };

    return (
        <Box mb="6">
            <Flex direction="column" gap="4">
                <Text as="div" size="4" weight="bold">
                    {title}
                </Text>

                {items.length === 0 && (
                    <Box py="4" px="4" style={{ backgroundColor: 'var(--gray-3)', borderRadius: 'var(--radius-3)' }}>
                        <Text size="2" color="gray">No items have been added yet.</Text>
                    </Box>
                )}

                {items.map((_item, index) => {
                    const childPath = `${path}.${index}`;
                    return (
                        <Card key={childPath} size="3" variant="surface">
                            <Flex direction="column" gap="4">
                                <Flex justify="between" align="center">
                                    <Text size="3" weight="bold">
                                        Item {index + 1}
                                    </Text>
                                    {enabled && (
                                        <Button
                                            color="red"
                                            variant="soft"
                                            onClick={() => handleRemoveItem(index)}
                                            title="Remove item"
                                        >
                                            <TrashIcon />
                                            Remove
                                        </Button>
                                    )}
                                </Flex>

                                <Box>
                                    <JsonFormsDispatch
                                        schema={validItemsSchema}
                                        uischema={
                                            uischema.options?.detail || {
                                                type: 'VerticalLayout',
                                                elements: Object.keys(
                                                    validItemsSchema.properties || {}
                                                ).map((key) => ({
                                                    type: 'Control',
                                                    scope: `#/properties/${key}`,
                                                })),
                                            }
                                        }
                                        path={childPath}
                                        enabled={enabled}
                                        renderers={undefined} /* use inherited renderers */
                                        cells={undefined} /* use inherited cells */
                                    />
                                </Box>
                            </Flex>
                        </Card>
                    );
                })}

                {enabled && (
                    <Box mt="2">
                        <Button
                            variant="surface"
                            onClick={handleAddItem}
                        >
                            <PlusIcon />
                            Add Item
                        </Button>
                    </Box>
                )}
            </Flex>
        </Box>
    );
};

export default withJsonFormsArrayControlProps(ArrayControl);
