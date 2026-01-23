import { useState, useEffect } from 'react';
import { Card, Button } from '@lsf/ui';
import { fetchApplication, fetchFormByTaskId, approveApplication, type Application, type FormResponse } from '../api';
import './ApplicationDetail.css';

interface ApplicationDetailProps {
  taskId: string;
  onApproved: () => void;
}

export function ApplicationDetail({ taskId, onApproved }: ApplicationDetailProps) {
  const [application, setApplication] = useState<Application | null>(null);
  const [form, setForm] = useState<FormResponse | null>(null);
  const [formData, setFormData] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    const abortController = new AbortController();
    void loadApplicationData(abortController.signal);
    return () => abortController.abort();
  }, [taskId]);

  const loadApplicationData = async (signal?: AbortSignal) => {
    try {
      setLoading(true);
      setError(null);

      // Fetch application and form in parallel
      const [app, formData] = await Promise.all([
        fetchApplication(taskId, signal),
        fetchFormByTaskId(taskId, signal),
      ]);

      setApplication(app);
      setForm(formData);
      
      // Initialize form data (empty for OGA to fill)
      setFormData({});
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') return;
      setError(err instanceof Error ? err.message : 'Failed to load application');
      console.error('Error loading application:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleFormChange = (field: string, value: unknown) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleApprove = async () => {
    if (!application) return;

    setIsSubmitting(true);
    setError(null);
    setSuccess(false);

    try {
      await approveApplication(
        application.taskId,
        application.consignmentId,
        formData,
        'APPROVED'
      );
      setSuccess(true);
      setTimeout(() => {
        onApproved();
      }, 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve application');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleReject = async () => {
    if (!application) return;

    setIsSubmitting(true);
    setError(null);
    setSuccess(false);

    try {
      await approveApplication(
        application.taskId,
        application.consignmentId,
        formData,
        'REJECTED'
      );
      setSuccess(true);
      setTimeout(() => {
        onApproved();
      }, 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject application');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loading) {
    return (
      <Card className="application-detail">
        <div className="loading">Loading application...</div>
      </Card>
    );
  }

  if (error && !application) {
    return (
      <Card className="application-detail">
        <div className="error">{error}</div>
      </Card>
    );
  }

  if (!application || !form) {
    return (
      <Card className="application-detail">
        <div className="error">Application not found</div>
      </Card>
    );
  }

  return (
    <Card className="application-detail">
      <h2>Application Review</h2>

      {error && (
        <div className="error">
          {error}
        </div>
      )}

      {success && (
        <div className="success">
          Application processed successfully!
        </div>
      )}

      {/* Application Info */}
      <div className="section">
        <h3>Application Information</h3>
        <div className="info-grid">
          <div className="info-item">
            <label>Task ID:</label>
            <span>{application.taskId}</span>
          </div>
          <div className="info-item">
            <label>Consignment ID:</label>
            <span>{application.consignmentId}</span>
          </div>
          <div className="info-item">
            <label>Form ID:</label>
            <span>{application.formId}</span>
          </div>
          <div className="info-item">
            <label>Status:</label>
            <span>{application.status}</span>
          </div>
        </div>
      </div>

      {/* Form Fields - Simple form rendering (same logic as trader portal) */}
      <div className="section">
        <h3>Review Form</h3>
        <p className="form-description">Fill out the form fields below:</p>
        
        {/* Simple form rendering based on schema */}
        {form.schema && typeof form.schema === 'object' && 'properties' in form.schema && (
          <div className="form-fields">
            {Object.entries((form.schema as { properties?: Record<string, unknown> }).properties || {}).map(([key, fieldSchema]) => {
              const field = fieldSchema as { type?: string; title?: string };
              const fieldType = field.type || 'string';
              const fieldTitle = field.title || key;

              return (
                <div key={key} className="form-field">
                  <label htmlFor={key}>{fieldTitle}</label>
                  {fieldType === 'string' && (
                    <input
                      id={key}
                      type="text"
                      value={(formData[key] as string) || ''}
                      onChange={(e) => handleFormChange(key, e.target.value)}
                      disabled={isSubmitting || success}
                    />
                  )}
                  {fieldType === 'boolean' && (
                    <input
                      id={key}
                      type="checkbox"
                      checked={(formData[key] as boolean) || false}
                      onChange={(e) => handleFormChange(key, e.target.checked)}
                      disabled={isSubmitting || success}
                    />
                  )}
                  {fieldType === 'number' && (
                    <input
                      id={key}
                      type="number"
                      value={(formData[key] as number) || ''}
                      onChange={(e) => handleFormChange(key, parseFloat(e.target.value) || 0)}
                      disabled={isSubmitting || success}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Action Buttons */}
      <div className="action-buttons">
        <Button
          className="approve-button"
          onClick={handleApprove}
          disabled={isSubmitting || success}
        >
          {isSubmitting ? 'Processing...' : 'Approve'}
        </Button>
        <Button
          className="reject-button"
          onClick={handleReject}
          disabled={isSubmitting || success}
        >
          {isSubmitting ? 'Processing...' : 'Reject'}
        </Button>
      </div>
    </Card>
  );
}
