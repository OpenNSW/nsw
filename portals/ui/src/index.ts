import "@radix-ui/themes/styles.css";
import './index.css'

export * from './components/Button';
export * from './components/Card';
export { default as FileControl } from './renderers/FileControl';
export * from './renderers/FileControlTester';

// New Renderers
import TextControl, { TextControlTester } from './renderers/TextControl';
import NumberControl, { NumberControlTester } from './renderers/NumberControl';
import BooleanControl, { BooleanControlTester } from './renderers/BooleanControl';
import SelectControl, { SelectControlTester } from './renderers/SelectControl';
import DateControl, { DateControlTester } from './renderers/DateControl';
import {
    VerticalLayoutRenderer, VerticalLayoutTester,
    HorizontalLayoutRenderer, HorizontalLayoutTester,
    GroupLayoutRenderer, GroupLayoutTester,
    CategorizationLayoutRenderer, CategorizationLayoutTester
} from './renderers/LayoutRenderers';
import FileControl from './renderers/FileControl';
import { FileControlTester } from './renderers/FileControlTester';

export const radixRenderers = [
    { tester: TextControlTester, renderer: TextControl },
    { tester: NumberControlTester, renderer: NumberControl },
    { tester: BooleanControlTester, renderer: BooleanControl },
    { tester: SelectControlTester, renderer: SelectControl },
    { tester: DateControlTester, renderer: DateControl },
    { tester: VerticalLayoutTester, renderer: VerticalLayoutRenderer },
    { tester: HorizontalLayoutTester, renderer: HorizontalLayoutRenderer },
    { tester: GroupLayoutTester, renderer: GroupLayoutRenderer },
    { tester: CategorizationLayoutTester, renderer: CategorizationLayoutRenderer },
    { tester: FileControlTester, renderer: FileControl },
];

export * from './renderers/TextControl';
export * from './renderers/NumberControl';
export * from './renderers/BooleanControl';
export * from './renderers/SelectControl';
export * from './renderers/DateControl';
export * from './renderers/LayoutRenderers';