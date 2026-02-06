import { vanillaRenderers } from '@jsonforms/vanilla-renderers';
import FileControl from './FileControl';
import { FileControlTester } from './FileControlTester';

export const customRenderers = [
    ...vanillaRenderers,
    { tester: FileControlTester, renderer: FileControl },
];
