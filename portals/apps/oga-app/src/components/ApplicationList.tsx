import { Card } from '@lsf/ui';
import './ApplicationList.css';
import type { Application } from '../api';

interface ApplicationListProps {
  applications: Application[];
  selectedTaskId?: string;
  onApplicationSelect: (taskId: string) => void;
}

export function ApplicationList({ applications, selectedTaskId, onApplicationSelect }: ApplicationListProps) {
  if (applications.length === 0) {
    return (
      <div className="empty-state">
        <h3>No applications pending review</h3>
        <p>All applications have been processed or there are no pending applications.</p>
      </div>
    );
  }

  return (
    <div className="application-list">
      {applications.map((application) => (
        <Card
          key={application.taskId}
          className={`application-item ${selectedTaskId === application.taskId ? 'selected' : ''}`}
          onClick={() => onApplicationSelect(application.taskId)}
        >
          <h3>Application Review</h3>
          <p><strong>Task ID:</strong> {application.taskId}</p>
          <p><strong>Consignment ID:</strong> {application.consignmentId}</p>
          <p><strong>Form ID:</strong> {application.formId}</p>
          <p><strong>Status:</strong> {application.status}</p>
        </Card>
      ))}
    </div>
  );
}
