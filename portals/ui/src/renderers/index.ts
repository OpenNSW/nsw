import FileControl from './FileControl';
import { FileControlTester } from './FileControlTester';
import { RadixTextControl, RadixTextControlTester } from './RadixTextControl';
import { RadixTextAreaControl, RadixTextAreaControlTester } from './RadixTextAreaControl';
import { RadixBooleanControl, RadixBooleanControlTester } from './RadixBooleanControl';
import { RadixSelectControl, RadixSelectControlTester } from './RadixSelectControl';
import { RadixVerticalLayout, RadixVerticalLayoutTester } from './RadixVerticalLayout';

export {
    FileControl, FileControlTester,
    RadixTextControl, RadixTextControlTester,
    RadixTextAreaControl, RadixTextAreaControlTester,
    RadixBooleanControl, RadixBooleanControlTester,
    RadixSelectControl, RadixSelectControlTester,
    RadixVerticalLayout, RadixVerticalLayoutTester
};

export const radixRenderers = [
    { tester: FileControlTester, renderer: FileControl },
    { tester: RadixTextControlTester, renderer: RadixTextControl },
    { tester: RadixTextAreaControlTester, renderer: RadixTextAreaControl },
    { tester: RadixBooleanControlTester, renderer: RadixBooleanControl },
    { tester: RadixSelectControlTester, renderer: RadixSelectControl },
    { tester: RadixVerticalLayoutTester, renderer: RadixVerticalLayout },
];

export const radixCells = [];