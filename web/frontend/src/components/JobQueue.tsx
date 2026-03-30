import React, { useEffect } from 'react';
import { useJobStore } from '../store/jobStore';
import { Activity, CheckCircle, XCircle, Clock, Loader, Trash2 } from 'lucide-react';
import axios from 'axios';
import { Job } from '../types';

export const JobQueue: React.FC = () => {
  const { jobs, setJobs, removeJob, setSelectedJob } = useJobStore();
  const [loading, setLoading] = React.useState(true);

  useEffect(() => {
    fetchJobs();
    const interval = setInterval(fetchJobs, 2000); // Poll every 2 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchJobs = async () => {
    try {
      const response = await axios.get('/api/jobs');
      setJobs(response.data || []);
    } catch (err) {
      console.error('Error fetching jobs:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteJob = async (jobId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm('Are you sure you want to cancel/delete this job?')) {
      return;
    }

    try {
      await axios.delete(`/api/jobs/${jobId}`);
      removeJob(jobId);
    } catch (err) {
      console.error('Error deleting job:', err);
      alert('Failed to delete job');
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <Loader className="animate-spin text-blue-600" size={20} />;
      case 'completed':
        return <CheckCircle className="text-green-600" size={20} />;
      case 'failed':
        return <XCircle className="text-red-600" size={20} />;
      case 'pending':
        return <Clock className="text-yellow-600" size={20} />;
      default:
        return <Activity className="text-gray-600" size={20} />;
    }
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'running':
        return 'bg-blue-100 text-blue-800';
      case 'completed':
        return 'bg-green-100 text-green-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const formatBytes = (bytes: number): string => {
    const mb = bytes / (1024 * 1024);
    if (mb >= 1024) {
      return `${(mb / 1024).toFixed(2)} GB`;
    }
    return `${mb.toFixed(2)} MB`;
  };

  const formatSpeed = (bytesPerSec: number): string => {
    const mbps = bytesPerSec / (1024 * 1024);
    return `${mbps.toFixed(2)} MB/s`;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader className="animate-spin text-forensic-600" size={32} />
      </div>
    );
  }

  if (jobs.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        <Activity className="mx-auto mb-2" size={48} />
        <p>No jobs yet</p>
        <p className="text-sm mt-1">Create an imaging or restore job to get started</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {jobs.map((job) => (
        <JobCard
          key={job.id}
          job={job}
          onSelect={() => setSelectedJob(job)}
          onDelete={(e) => handleDeleteJob(job.id, e)}
          getStatusIcon={getStatusIcon}
          getStatusColor={getStatusColor}
          formatBytes={formatBytes}
          formatSpeed={formatSpeed}
        />
      ))}
    </div>
  );
};

interface JobCardProps {
  job: Job;
  onSelect: () => void;
  onDelete: (e: React.MouseEvent) => void;
  getStatusIcon: (status: string) => React.ReactNode;
  getStatusColor: (status: string) => string;
  formatBytes: (bytes: number) => string;
  formatSpeed: (bytesPerSec: number) => string;
}

const JobCard: React.FC<JobCardProps> = ({
  job,
  onSelect,
  onDelete,
  getStatusIcon,
  getStatusColor,
  formatBytes,
  formatSpeed,
}) => {
  return (
    <div
      onClick={onSelect}
      className="border border-gray-200 rounded-lg p-4 hover:border-forensic-500 hover:shadow-md transition cursor-pointer"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 flex-1 min-w-0">
          {getStatusIcon(job.status)}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="font-semibold text-gray-900 capitalize">{job.type} Job</h3>
              <span className={`px-2 py-1 rounded text-xs font-medium ${getStatusColor(job.status)}`}>
                {job.status_str}
              </span>
            </div>
            <p className="text-sm text-gray-600 truncate">
              {job.source_path} → {job.dest_path}
            </p>
            <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
              <span>{job.format}</span>
              {job.bytes_total > 0 && (
                <span>{formatBytes(job.bytes_total)}</span>
              )}
              {job.status === 'running' && job.speed > 0 && (
                <span className="text-blue-600 font-medium">
                  {formatSpeed(job.speed)}
                </span>
              )}
            </div>

            {/* Progress Bar */}
            {job.status === 'running' && (
              <div className="mt-3">
                <div className="flex items-center justify-between text-xs text-gray-600 mb-1">
                  <span>{job.phase}</span>
                  <span>{job.progress.toFixed(1)}%</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                    style={{ width: `${job.progress}%` }}
                  />
                </div>
                {job.eta && (
                  <p className="text-xs text-gray-500 mt-1">ETA: {job.eta}</p>
                )}
              </div>
            )}

            {/* Completed Info */}
            {job.status === 'completed' && job.hash && (
              <div className="mt-2 text-xs text-gray-500">
                <span className="font-medium">Hash:</span> {job.hash.substring(0, 16)}...
              </div>
            )}

            {/* Errors */}
            {job.errors && job.errors.length > 0 && (
              <div className="mt-2 text-xs text-red-600">
                {job.errors[job.errors.length - 1]}
              </div>
            )}
          </div>
        </div>

        {/* Delete Button */}
        <button
          onClick={onDelete}
          className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition"
          title="Cancel/Delete Job"
        >
          <Trash2 size={16} />
        </button>
      </div>
    </div>
  );
};
