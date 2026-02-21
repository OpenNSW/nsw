DELETE FROM forms
WHERE id = '71e5e73a-ebb2-4750-aaa4-f71087adac43';

UPDATE workflow_node_templates
SET config = '{"formId": "11111111-1111-1111-1111-111111111111", "submission": {"url": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec", "response": {"mapping": {"assesmentNo": "gi:cusdec:assesmentNo"}}}, "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}'
WHERE id = 'c0000003-0003-0003-0003-000000000002';