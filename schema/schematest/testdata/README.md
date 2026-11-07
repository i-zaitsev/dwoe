# Test fixtures

`sample.jsonl` contains one canonical line per (type, subtype, first
content block) combination observed in real workspace logs. Lines are
pulled verbatim so tests exercise the exact shapes `claude -p
--output-format stream-json` emits.

Line provenance (1-indexed, matches order in sample.jsonl):

 1. system_init                 38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:191
 2. assistant_thinking          38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:192
 3. assistant_tool_use          38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:193
 4. assistant_text              38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:325
 5. user_text                   38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:197
 6. user_tool_result            38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:198
 7. system_task_started         38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:196
 8. system_task_progress        38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:199
 9. system_task_notification    38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:322
10. rate_limit_event            38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:195
11. result_success              38022905-257c-4492-bab9-92efd9e7df2f/logs/agent/container.log:572
12. system_api_retry            4d0c0a02-89a0-41a1-ba5a-9d72fc31d62a/logs/agent/container.log:45

`faulty.jsonl` contains hand-crafted malformed or unknown inputs that
exercise Parse's error / catch-all paths. Lines are not expected to
decode to concrete types.
