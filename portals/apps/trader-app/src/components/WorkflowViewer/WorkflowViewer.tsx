import { useCallback, useMemo, useState, useEffect } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  MarkerType,
} from '@xyflow/react'
import type { Edge, NodeTypes } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Button } from '@radix-ui/themes'
import { ReloadIcon } from '@radix-ui/react-icons'
import type { WorkflowV2 } from '../../services/types/workflow'
import type { WorkflowNode as LegacyWorkflowNode } from '../../services/types/consignment'
import { WorkflowNode } from './WorkflowNode'
import type { WorkflowNodeType } from './WorkflowNode'
import { ReactFlowProvider, useReactFlow } from '@xyflow/react'

interface WorkflowViewerProps {
  workflow?: WorkflowV2
  className?: string
  onRefresh?: () => void
  refreshing?: boolean
  steps?: LegacyWorkflowNode[]
}

const nodeTypes: NodeTypes = {
  workflowStep: WorkflowNode,
}

function getNodePosition(
  step: LegacyWorkflowNode,
  allSteps: LegacyWorkflowNode[]
): { x: number; y: number } {
  // Calculate depth based on dependencies (topological layer)
  const depths = new Map<string, number>()

  function calculateDepth(stepId: string): number {
    if (depths.has(stepId)) return depths.get(stepId)!

    const s = allSteps.find((st) => st.id === stepId)
    if (!s || s.depends_on.length === 0) {
      depths.set(stepId, 0)
      return 0
    }

    const maxParentDepth = Math.max(
      ...s.depends_on.map((depId) => calculateDepth(depId))
    )
    const depth = maxParentDepth + 1
    depths.set(stepId, depth)
    return depth
  }

  // Calculate depths for all steps
  allSteps.forEach((s) => calculateDepth(s.id))

  const depth = depths.get(step.id) || 0

  // Group steps by depth to calculate horizontal position
  const stepsAtSameDepth = allSteps.filter(
    (s) => depths.get(s.id) === depth
  )
  const indexAtDepth = stepsAtSameDepth.findIndex((s) => s.id === step.id)
  const totalAtDepth = stepsAtSameDepth.length

  // Center nodes horizontally within their depth layer (vertical flow)
  const verticalSpacing = 200
  const horizontalSpacing = 300
  const startX = -(totalAtDepth - 1) * horizontalSpacing / 2

  return {
    x: startX + indexAtDepth * horizontalSpacing,
    y: depth * verticalSpacing,
  }
}

function convertToReactFlow(workflow: WorkflowV2): {
  nodes: WorkflowNodeType[]
  edges: Edge[]
} {
  const nodes: WorkflowNodeType[] = workflow.nodes.map((node) => ({
    id: node.id,
    type: 'workflowStep' as const,
    position: { x: node.x, y: node.y },
    data: {
      step: node,
    },
  }))

  const edges: Edge[] = workflow.edges.map((edge) => ({
    id: edge.id,
    source: edge.source_id,
    target: edge.target_id,
    label: edge.condition,
    labelStyle: { fontSize: 10, fill: '#64748b' },
    markerEnd: {
      type: MarkerType.ArrowClosed,
      width: 20,
      height: 20,
      color: '#64748b',
    },
    style: {
      strokeWidth: 2,
      stroke: '#64748b',
    },
  }))

  return { nodes, edges }
}

function convertLegacyToReactFlow(steps: LegacyWorkflowNode[]): {
  nodes: WorkflowNodeType[]
  edges: Edge[]
} {
  const nodes: WorkflowNodeType[] = steps.map((step) => ({
    id: step.id,
    type: 'workflowStep' as const,
    position: getNodePosition(step, steps),
    data: {
      step,
    },
  }))

  const edges: Edge[] = []
  steps.forEach((step) => {
    step.depends_on.forEach((depId) => {
      const sourceStep = steps.find(s => s.id === depId)
      const isCompleted = sourceStep?.state === 'COMPLETED'

      edges.push({
        id: `${depId}-${step.id}`,
        source: depId,
        target: step.id,
        markerEnd: {
          type: MarkerType.ArrowClosed,
          width: 20,
          height: 20,
          color: isCompleted ? '#10b981' : '#64748b',
        },
        style: {
          strokeWidth: 2,
          stroke: isCompleted ? '#10b981' : '#64748b',
        },
      })
    })
  })

  return { nodes, edges }
}

function WorkflowViewerContent({ workflow, steps, className = '', onRefresh, refreshing = false }: WorkflowViewerProps) {
  const [isSpacePressed, setIsSpacePressed] = useState(false)
  const { fitView } = useReactFlow()

  const { nodes: initialNodes, edges: initialEdges } = useMemo(() => {
    if (workflow) return convertToReactFlow(workflow)
    if (steps && steps.length > 0) return convertLegacyToReactFlow(steps)
    return { nodes: [], edges: [] }
  }, [workflow, steps])

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)

  const focusOnReadyNodes = useCallback(() => {
    let readyNodeIds: string[] = []
    if (workflow) {
      readyNodeIds = workflow.nodes.filter((s) => s.state === 'READY').map((s) => s.id)
    } else if (steps) {
      readyNodeIds = steps.filter((s) => s.state === 'READY').map((s) => s.id)
    }

    setTimeout(() => {
      if (readyNodeIds.length > 0) {
        fitView({
          nodes: readyNodeIds.map((id) => ({ id })),
          padding: {
            x: 2,
            y: 0,
          },
          maxZoom: 1.0,
          minZoom: 0.5,
          duration: 800,
          interpolate: "linear",
        })
      } else {
        fitView({ padding: 0.5, maxZoom: 1.0, duration: 800 })
      }
    }, 100)
  }, [workflow, fitView])

  // Update nodes and edges when workflow changes
  useEffect(() => {
    setNodes(initialNodes)
    setEdges(initialEdges)
    focusOnReadyNodes()
  }, [initialNodes, initialEdges, setNodes, setEdges, focusOnReadyNodes])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === ' ') {
        setIsSpacePressed(true)
      }
    }

    const handleKeyUp = (e: KeyboardEvent) => {
      if (e.key === ' ') {
        setIsSpacePressed(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    window.addEventListener('keyup', handleKeyUp)

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
      window.removeEventListener('keyup', handleKeyUp)
    }
  }, [])

  return (
    <div className={`w-full bg-slate-50 rounded-lg border border-gray-200 relative flex flex-col ${className}`}>
      {onRefresh && (
        <div className="absolute top-3 right-3 z-10">
          <Button
            variant="soft"
            color="gray"
            size="2"
            onClick={onRefresh}
            disabled={refreshing}
          >
            <ReloadIcon className={refreshing ? 'animate-spin' : ''} />
            Refresh
          </Button>
        </div>
      )}
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.5, maxZoom: 1.0 }}
        nodesDraggable={isSpacePressed}
        nodesConnectable={false}
        panOnDrag={true}
        style={{ cursor: isSpacePressed ? 'move' : 'grab' }}
      >
        <Background color="#e2e8f0" gap={16} />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  )
}

export function WorkflowViewer(props: WorkflowViewerProps) {
  return (
    <ReactFlowProvider>
      <WorkflowViewerContent {...props} />
    </ReactFlowProvider>
  )
}
