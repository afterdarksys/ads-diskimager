import React, { useEffect } from 'react';
import { useJobStore } from '../store/jobStore';
import { DiskSelector } from './DiskSelector';
import { JobQueue } from './JobQueue';
import { ProgressMonitor } from './ProgressMonitor';
import { Activity, HardDrive, Settings } from 'lucide-react';

export const Dashboard: React.FC = () => {
  const { jobs, disks, selectedJob } = useJobStore();

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-forensic-800 text-white shadow-lg">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold">Diskimager Forensics Suite</h1>
              <p className="text-forensic-200 mt-1">Web-Based Digital Evidence Acquisition</p>
            </div>
            <button className="flex items-center gap-2 px-4 py-2 bg-forensic-700 hover:bg-forensic-600 rounded-lg transition">
              <Settings size={20} />
              Settings
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Stats Cards */}
          <div className="lg:col-span-3 grid grid-cols-1 md:grid-cols-3 gap-4">
            <StatCard
              icon={<Activity className="text-forensic-600" size={24} />}
              title="Active Jobs"
              value={jobs.filter(j => j.status === 'running').length}
              subtitle="Currently running"
            />
            <StatCard
              icon={<HardDrive className="text-green-600" size={24} />}
              title="Available Disks"
              value={disks.length}
              subtitle="Ready for imaging"
            />
            <StatCard
              icon={<Activity className="text-blue-600" size={24} />}
              title="Completed"
              value={jobs.filter(j => j.status === 'completed').length}
              subtitle="Total jobs"
            />
          </div>

          {/* Disk Selector */}
          <div className="lg:col-span-1">
            <div className="bg-white rounded-lg shadow-md p-6">
              <h2 className="text-xl font-semibold mb-4">Available Disks</h2>
              <DiskSelector />
            </div>
          </div>

          {/* Job Queue */}
          <div className="lg:col-span-2">
            <div className="bg-white rounded-lg shadow-md p-6">
              <h2 className="text-xl font-semibold mb-4">Job Queue</h2>
              <JobQueue />
            </div>
          </div>

          {/* Progress Monitor */}
          {selectedJob && (
            <div className="lg:col-span-3">
              <div className="bg-white rounded-lg shadow-md p-6">
                <h2 className="text-xl font-semibold mb-4">Progress Monitor</h2>
                <ProgressMonitor job={selectedJob} />
              </div>
            </div>
          )}
        </div>
      </main>

      {/* Footer */}
      <footer className="bg-white border-t mt-auto">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between text-sm text-gray-600">
            <span>Diskimager Forensics Suite v1.0.0</span>
            <span>© 2026 AfterDark Systems</span>
          </div>
        </div>
      </footer>
    </div>
  );
};

interface StatCardProps {
  icon: React.ReactNode;
  title: string;
  value: number;
  subtitle: string;
}

const StatCard: React.FC<StatCardProps> = ({ icon, title, value, subtitle }) => {
  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <div className="flex items-center gap-4">
        <div className="p-3 bg-gray-100 rounded-lg">
          {icon}
        </div>
        <div className="flex-1">
          <p className="text-sm text-gray-600">{title}</p>
          <p className="text-3xl font-bold text-gray-900">{value}</p>
          <p className="text-xs text-gray-500">{subtitle}</p>
        </div>
      </div>
    </div>
  );
};
