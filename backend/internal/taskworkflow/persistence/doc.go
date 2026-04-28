// Package persistence stores task workflow task records.
//
// The package exposes two store shapes with different authority levels:
//
//   - Store is the full task workflow repository. Use it only from orchestration
//     code that owns task workflow lifecycle decisions, such as creating task
//     rows during initialization, looking up tasks by macro workflow, or
//     deleting task records.
//
//   - TaskScopedStore is the restricted store passed to code executing a single
//     task. It is bound to one task ID at construction time and only allows that
//     task to read or update its own state and render data.
//
// In general, runtime/manager code should hold Store and construct
// TaskScopedStore values for subtasks. Subtasks, activities, plugins, and
// rendering helpers should not receive Store directly.
package persistence
