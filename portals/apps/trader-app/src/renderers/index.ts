import {
    type JsonFormsRendererRegistryEntry,
    type JsonFormsCellRendererRegistryEntry,
} from '@jsonforms/core';
import { RadixTextControl, RadixTextControlTester } from './RadixTextControl';
import { RadixVerticalLayout, RadixVerticalLayoutTester } from './RadixVerticalLayout';
import { RadixBooleanControl, RadixBooleanControlTester } from './RadixBooleanControl';

export const radixRenderers: JsonFormsRendererRegistryEntry[] = [
    { tester: RadixTextControlTester, renderer: RadixTextControl },
    { tester: RadixVerticalLayoutTester, renderer: RadixVerticalLayout },
    { tester: RadixBooleanControlTester, renderer: RadixBooleanControl },
];

export const radixCells: JsonFormsCellRendererRegistryEntry[] = [];
