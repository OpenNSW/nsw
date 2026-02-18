import { JsonForms } from '@jsonforms/react';
import { radixRenderers } from '@lsf/ui';
import { sendTaskCommand } from "../services/task.ts";

import { useLocation, useNavigate, useParams } from "react-router-dom";
import { useState, useCallback } from "react";
import { Button } from "@radix-ui/themes";
import type { JsonSchema, UISchemaElement } from '@jsonforms/core';
import { autoFillForm } from "../utils/formUtils";



export interface TaskFormData {
  title: string
  schema: JsonSchema
  uiSchema: UISchemaElement
  formData: Record<string, unknown>
}

export type SimpleFormConfig = {
  traderFormInfo: TaskFormData
  ogaReviewForm?: TaskFormData
}

function TraderForm(props: { formInfo: TaskFormData, pluginState: string }) {
  const { consignmentId, preConsignmentId, taskId } = useParams<{
    consignmentId?: string
    preConsignmentId?: string
    taskId?: string
  }>()
  const location = useLocation()
  const navigate = useNavigate()
  const [data, setData] = useState<Record<string, unknown>>(props.formInfo.formData || {})
  const [errors, setErrors] = useState<any[]>([])
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const READ_ONLY_STATES = ['OGA_REVIEWED', 'SUBMITTED', 'OGA_ACKNOWLEDGED'];
  const isReadOnly = READ_ONLY_STATES.includes(props.pluginState);

  const isPreConsignment = location.pathname.includes('/pre-consignments/')
  const workflowId = preConsignmentId || consignmentId



  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!workflowId || !taskId) {
      setSubmitError('Workflow ID or Task ID is missing.')
      return
    }

    if (errors.length > 0) {
      setSubmitError('Please fix validation errors before submitting.');
      return;
    }

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      // Send form submission
      const preparedData = data

      const response = await sendTaskCommand({
        command: 'SUBMISSION',
        taskId,
        workflowId,
        data: preparedData,
      })

      if (response.success) {
        // Navigate back to appropriate workflow list
        navigate(isPreConsignment ? '/pre-consignments' : `/consignments/${workflowId}`)
      } else {
        setSubmitError(response.error?.message || 'Failed to submit form.')
      }
    } catch (err) {
      console.error('Error submitting form:', err)
      setSubmitError('Failed to submit form. Please try again.')
    } finally {
      setIsSubmitting(false);
    }
  }

  const handleAutoFill = useCallback(() => {
    const filledData = autoFillForm(props.formInfo.schema, data);
    setData(filledData);
  }, [props.formInfo.schema, data]);

  const showAutoFillButton = import.meta.env.VITE_SHOW_AUTOFILL_BUTTON === 'true'

  return (
    <>
      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <h1 className="text-2xl font-bold text-gray-800">{props.formInfo.title}</h1>
      </div>

      <div className="bg-white rounded-lg shadow-md p-6">
        <form onSubmit={handleSubmit} noValidate>
          <JsonForms
            schema={props.formInfo.schema}
            uischema={props.formInfo.uiSchema}
            data={data}
            renderers={radixRenderers}
            readonly={isReadOnly}
            onChange={({ data, errors }) => {
              setData(data);
              setErrors(errors || []);
            }}
          />
          {!isReadOnly && (
            <div className={`mt-4 flex gap-3 ${showAutoFillButton ? 'justify-between' : ''}`}>
              {showAutoFillButton && (
                <Button
                  type="button"
                  variant="soft"
                  color="purple"
                  size={"3"}
                  className={"flex-1!"}
                  onClick={handleAutoFill}
                  disabled={isSubmitting}
                >
                  Demo - Auto Fill
                </Button>
              )}
              <Button
                type="submit"
                disabled={isSubmitting}
                className={'flex-1!'}
                size={"3"}
              >
                {isSubmitting ? 'Submitting...' : 'Submit Form'}
              </Button>
            </div>
          )}
        </form>
      </div>

      {submitError && (
        <div className="bg-red-100 text-red-700 rounded-lg p-4 mt-4">
          <p>{submitError}</p>
        </div>
      )}
    </>
  )
}

function OgaReviewForm(props: { formInfo: TaskFormData }) {
  const [data] = useState(props.formInfo.formData)

  return (
    <>
      <div className="bg-blue-50 border border-blue-200 rounded-lg shadow-md p-6 mb-6 mt-6">
        <h1 className="text-2xl font-bold text-blue-800">{props.formInfo.title}</h1>
      </div>

      <div className="bg-blue-50 border border-blue-200 rounded-lg shadow-md p-6">
        <JsonForms
          schema={props.formInfo.schema}
          uischema={props.formInfo.uiSchema}
          data={data}
          renderers={radixRenderers}
          readonly={true}
        />
      </div>
    </>
  )
}

export default function SimpleForm(props: { configs: SimpleFormConfig, pluginState: string }) {
  return (
    <div>
      <TraderForm formInfo={props.configs.traderFormInfo} pluginState={props.pluginState} />

      {props.configs.ogaReviewForm && (
        <OgaReviewForm formInfo={props.configs.ogaReviewForm} />
      )}
    </div>
  )
}
