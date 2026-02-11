import { vanillaRenderers } from '@jsonforms/vanilla-renderers';
import {
    FileControl, FileControlTester,
    RadixTextControl, RadixTextControlTester,
    RadixTextAreaControl, RadixTextAreaControlTester,
    RadixSelectControl, RadixSelectControlTester,
    RadixBooleanControl, RadixBooleanControlTester,
    RadixVerticalLayout, RadixVerticalLayoutTester
} from '@lsf/ui';

export { FileControl, FileControlTester };
export const customRenderers = [
    ...vanillaRenderers,
    { tester: FileControlTester, renderer: FileControl },
    { tester: RadixTextControlTester, renderer: RadixTextControl },
    { tester: RadixTextAreaControlTester, renderer: RadixTextAreaControl },
    { tester: RadixSelectControlTester, renderer: RadixSelectControl },
    { tester: RadixBooleanControlTester, renderer: RadixBooleanControl },
    { tester: RadixVerticalLayoutTester, renderer: RadixVerticalLayout },
];
