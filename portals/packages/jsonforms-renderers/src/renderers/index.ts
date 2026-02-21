import TextControl, { TextControlTester } from './TextControl';
import NumberControl, { NumberControlTester } from './NumberControl';
import BooleanControl, { BooleanControlTester } from './BooleanControl';
import SelectControl, { SelectControlTester } from './SelectControl';
import DateControl, { DateControlTester } from './DateControl';
import {
    VerticalLayoutRenderer, VerticalLayoutTester,
    HorizontalLayoutRenderer, HorizontalLayoutTester,
    GroupLayoutRenderer, GroupLayoutTester,
    CategorizationLayoutRenderer, CategorizationLayoutTester,
    type CategorizationLayoutProps
} from './LayoutRenderers';
import FileControl from './FileControl';
import { FileControlTester } from './FileControlTester';
import LabelRenderer, { LabelTester } from './LabelRenderer';

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
    { tester: LabelTester, renderer: LabelRenderer },
];

export * from './TextControl';
export * from './NumberControl';
export * from './BooleanControl';
export * from './SelectControl';
export * from './DateControl';
export * from './LayoutRenderers';
export type { CategorizationLayoutProps };
export { default as FileControl } from './FileControl';
export * from './FileControlTester';
export * from './LabelRenderer';