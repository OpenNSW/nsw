import { vanillaRenderers } from '@jsonforms/vanilla-renderers';
import { FileControl, FileControlTester } from '@lsf/ui';

export { FileControl, FileControlTester };
export const customRenderers = [
    ...vanillaRenderers,
    { tester: FileControlTester, renderer: FileControl },
];
